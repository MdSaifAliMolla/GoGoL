package crawler

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/html"
)

type RateLimiter struct {
	mu          sync.Mutex
	lastRequest map[string]time.Time
	interval    time.Duration
}

func NewRateLimiter(interval time.Duration) *RateLimiter {
	return &RateLimiter{
		lastRequest: make(map[string]time.Time),
		interval:    interval,
	}
}

func (r *RateLimiter) Wait(u string) {
	domain := u
	if parsed, err := url.Parse(u); err == nil {
		domain = parsed.Host
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	last, ok := r.lastRequest[domain]
	now := time.Now()
	if ok {
		sleep := r.interval - now.Sub(last)
		if sleep > 0 {
			r.lastRequest[domain] = now.Add(sleep)
			r.mu.Unlock()
			time.Sleep(sleep)
			r.mu.Lock()
			return
		}
	}
	r.lastRequest[domain] = now
}


// job represents a URL to be crawled at a specific depth.
type job struct {
	url   string
	depth int
}

// result holds crawl output
type result struct {
	links []string
	page  Page
}


// Start initiates the crawling process.
func (c *Crawler) Start() {
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, c.Config.MaxConcurrent)
	
	wg.Add(1)
	go c.process(c.Config.SeedURL, 0, &wg, semaphore)
	wg.Wait()
}

func (c *Crawler) process(url string, depth int, wg *sync.WaitGroup, sem chan struct{}) {
	defer wg.Done()
	if depth > c.Config.MaxDepth { return }
	if !c.markVisited(url) { return }

	sem <- struct{}{}
	c.rateLimiter.Wait(url)
	links, page := c.crawlPage(job{url: url, depth: depth})
	<-sem

	// Hook for Indexer
	if c.OnPage != nil {
		c.OnPage(page)
	} else {
		// Default logging
		fmt.Printf("[Crawled] %s | Title: %s\n", page.URL, page.Title)
	}

	if depth < c.Config.MaxDepth {
		for _, link := range links {
			if !c.isVisited(link) {
				wg.Add(1)
				go c.process(link, depth+1, wg, sem)
			}
		}
	}
}

func (c *Crawler) markVisited(url string) bool {
	c.visitedMu.Lock()
	defer c.visitedMu.Unlock()
	if c.visited[url] {
		return false
	}
	c.visited[url] = true
	return true
}

func (c *Crawler) isVisited(url string) bool {
	c.visitedMu.RLock()
	defer c.visitedMu.RUnlock()
	return c.visited[url]
}

// crawlPage fetches and parses.
func (c *Crawler) crawlPage(j job) ([]string, Page) {

	page := Page{URL: j.url}
	resp, err := http.Get(j.url)
	if err != nil { return nil, page }
	defer resp.Body.Close()

	if resp.StatusCode != 200 { return nil, page }

	tokenizer := html.NewTokenizer(resp.Body)
	var links []string
	var textBuilder strings.Builder
	inScript := false
	
	for {
		tt := tokenizer.Next()
		if tt == html.ErrorToken { break }
		
		token := tokenizer.Token()
		switch tt {
		case html.StartTagToken:
			if token.Data == "script" || token.Data == "style" {
				inScript = true
			}
			if token.Data == "a" {
				for _, a := range token.Attr {
					if a.Key == "href" && strings.HasPrefix(a.Val, "http") {
						links = append(links, a.Val)
					}
				}
			}
			if token.Data == "title" {
				tokenizer.Next()
				page.Title = tokenizer.Token().Data
			}
		case html.EndTagToken:
			if token.Data == "script" || token.Data == "style" {
				inScript = false
			}
		case html.TextToken:
			if !inScript {
				text := strings.TrimSpace(token.Data)
				if len(text) > 0 {
					textBuilder.WriteString(text + " ")
				}
			}
		}
	}
	
	page.Content = textBuilder.String()
	// Snippet is first 200 chars
	if len(page.Content) > 200 {
		page.Snippet = page.Content[:200] + "..."
	} else {
		page.Snippet = page.Content
	}
	
	return links, page
}

