package index

import (
	"math"
	"sort"
	"strings"
	"sync"
	"unicode"

	"github.com/MdSaifAliMolla/GoGoL/internal/crawler"
)

// Indexer represents an inverted index.
type Indexer struct {
	mu    sync.RWMutex
	store map[string]map[string]int // word -> url -> count (TF)
	docs  map[string]crawler.Page // url -> Page details
}

func New() *Indexer {
	return &Indexer{
		store: make(map[string]map[string]int),
		docs:  make(map[string]crawler.Page),
	}
}

// Add adds a page to the index.
func (idx *Indexer) Add(p crawler.Page) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	idx.docs[p.URL] = p
	words := tokenize(p.Title + " " + p.Content)

	for _, w := range words {
		if idx.store[w] == nil {
			idx.store[w] = make(map[string]int)
		}
		idx.store[w][p.URL]++
	}
}

// Search searches for keywords and returns ranked results.
func (idx *Indexer) Search(query string) []crawler.Page {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	terms := tokenize(query)
	if len(terms) == 0 {
		return nil
	}

	// Score map: URL -> Score
	scores := make(map[string]float64)
	
	// Total docs
	N := float64(len(idx.docs))

	for _, term := range terms {
		if docsWithTerm, ok := idx.store[term]; ok {
			// IDF: log(N / df)
			df := float64(len(docsWithTerm))
			idf := math.Log(1 + (N / df)) // +1 to avoid division by zero or log(0) if df=0? (df wont be 0 here)

			for url, tf := range docsWithTerm {
				// TF-IDF = tf * idf
				// TF can be raw count or log(1+count). Using raw for simplicity.
				scores[url] += float64(tf) * idf
			}
		}
	}

	// Convert to list and sort
	type ranked struct {
		url   string
		score float64
	}
	var ranking []ranked
	for u, s := range scores {
		ranking = append(ranking, ranked{u, s})
	}

	sort.Slice(ranking, func(i, j int) bool {
		return ranking[i].score > ranking[j].score
	})

	var results []crawler.Page
	for _, r := range ranking {
		if page, ok := idx.docs[r.url]; ok {
			// Highlight snippet
			page.Snippet = idx.highlight(page.Content, terms)
			results = append(results, page)
		}
	}
	return results
}

func (idx *Indexer) highlight(content string, terms []string) string {
	// Find first occurrence of any term
	lowerContent := strings.ToLower(content)
	lowIndex := -1
	for _, t := range terms {
		idx := strings.Index(lowerContent, t)
		if idx != -1 {
			if lowIndex == -1 || idx < lowIndex {
				lowIndex = idx
			}
		}
	}

	if lowIndex == -1 {
		if len(content) > 200 {
			return content[:200] + "..."
		}
		return content
	}

	// Context window
	start := lowIndex - 50
	if start < 0 { start = 0 }
	end := start + 200
	if end > len(content) { end = len(content) }

	snippet := content[start:end]
	highlighted := snippet

	for _, t := range terms {
		// simple case-insensitive replace
		highlighted = strings.ReplaceAll(
			highlighted,
			t,
			"**"+t+"**",
		)
		highlighted = strings.ReplaceAll(
			highlighted,
			strings.Title(t),
			"**"+strings.Title(t)+"**",
		)
	}
	return "..." + highlighted + "..."
}

// Stats returns index statistics.
func (idx *Indexer) Stats() map[string]interface{} {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	return map[string]interface{}{
		"total_pages": len(idx.docs),
		"total_terms": len(idx.store),
	}
}



func tokenize(text string) []string {
	f := func(c rune) bool {
		return !unicode.IsLetter(c) && !unicode.IsNumber(c)
	}
	fields := strings.FieldsFunc(text, f)
	var words []string
	for _, w := range fields {
		if len(w) > 2 {
			words = append(words, strings.ToLower(w))
		}
	}
	return words
}

