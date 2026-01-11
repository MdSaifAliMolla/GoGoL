package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/MdSaifAliMolla/GoGoL/internal/api"
	"github.com/MdSaifAliMolla/GoGoL/internal/crawler"
	"github.com/MdSaifAliMolla/GoGoL/internal/index"
	"github.com/MdSaifAliMolla/GoGoL/internal/storage"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	if len(os.Args) < 2 {
		fmt.Println("Usage: web-crawler <command> [args]")
		fmt.Println("Commands: crawl, search, serve")
		os.Exit(1)
	}

	mongoURI := os.Getenv("MONGODB_URI")
	if mongoURI == "" {
		mongoURI = "mongodb://localhost:27017"
	}

	store, err := storage.NewMongoStorage(mongoURI, "web_crawler", "pages")
	if err != nil {
		fmt.Printf("Warning: Could not connect to MongoDB (%v). Persistence disabled.\n", err)
		store = nil
	} else {
		defer store.Close()
	}

	switch os.Args[1] {
	case "crawl":
		crawlCmd := flag.NewFlagSet("crawl", flag.ExitOnError)
		seed := crawlCmd.String("seed", "https://example.com", "Seed URL")
		depth := crawlCmd.Int("depth", 1, "Max depth")
		crawlCmd.Parse(os.Args[2:])

		idx := index.New()
		
		cfg := crawler.Config{
			SeedURL:       *seed,
			MaxDepth:      *depth,
			MaxConcurrent: 10,
		}

		c := crawler.New(cfg)
		c.OnPage = func(p crawler.Page) {
			fmt.Printf("[Crawled] %s | %s\n", p.URL, p.Title)
			idx.Add(p) 
			if store != nil {
				// Fire and forget save (or handle error)
				go func() {
					if err := store.SavePage(p); err != nil {
						fmt.Printf("Error saving to DB: %v\n", err)
					}
				}()
			}
		}

		fmt.Printf("Starting crawler on %s...\n", *seed)
		start := time.Now()
		c.Start()
		fmt.Printf("Crawling completed in %v\n", time.Since(start))
		
	case "search":
		searchCmd := flag.NewFlagSet("search", flag.ExitOnError)
		searchCmd.Parse(os.Args[2:])
		query := searchCmd.Arg(0)
		
		if query == "" {
			fmt.Println("Usage: web-crawler search <query>")
			return
		}
		
		idx := index.New()
		if store != nil {
			fmt.Println("Loading pages from MongoDB...")
			pages, err := store.GetPages()
			if err != nil {
				fmt.Printf("Error loading pages: %v\n", err)
			}
			for _, p := range pages {
				idx.Add(p)
			}
			fmt.Printf("Indexed %d pages.\n", len(pages))
		} else {
			fmt.Println("MongoDB not available. Search will be empty.")
		}
		
		results := idx.Search(query)
		fmt.Printf("Found %d results for '%s':\n", len(results), query)
		for _, p := range results {
			fmt.Printf("- %s\n  Title: %s\n  Snippet: %s\n", p.URL, p.Title, p.Snippet)
		}
		
	case "serve":
		idx := index.New()
		if store != nil {
			fmt.Println("Loading pages from MongoDB...")
			pages, err := store.GetPages()
			if err != nil {
				fmt.Printf("Error loading pages: %v\n", err)
			}
			for _, p := range pages {
				idx.Add(p)
			}
			fmt.Printf("Indexed %d pages.\n", len(pages))
		}

		srv := api.NewServer(idx)
		if err := srv.Start("8080"); err != nil {
			fmt.Printf("Server failed: %v\n", err)
		}

	default:

		fmt.Println("Unknown command")
	}
}
