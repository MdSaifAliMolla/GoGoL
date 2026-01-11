package crawler

import (
	"sync"
	"time"
)

// Page represents a crawled webpage.
type Page struct {
	URL     string `json:"url"`
	Title   string `json:"title"`
	Snippet string `json:"snippet"`
	Content string `json:"-"` // Hide full content from API if not requested
}

// Config holds configuration for the crawler.
type Config struct {
	MaxDepth      int
	MaxConcurrent int
	SeedURL       string
}

// Crawler manages the crawling process.
type Crawler struct {
	Config      Config
	visited     map[string]bool
	visitedMu   sync.RWMutex
	rateLimiter *RateLimiter
	OnPage      func(Page) // Callback when a page is crawled
}

// New creates a new Crawler instance.
func New(cfg Config) *Crawler {
	return &Crawler{
		Config:      cfg,
		visited:     make(map[string]bool),
		rateLimiter: NewRateLimiter(1 * time.Second), // Default 1s per domain
	}
}


