package scraper

import "context"

// Scraper defines the interface for fetching metadata from a URL.
type Scraper interface {
	// ScrapeMetadata fetches the title and description for a given URL.
	// It returns the title, description, and an error if scraping fails.
	ScrapeMetadata(ctx context.Context, url string) (title string, description string, err error)

	// TODO: Consider adding a Close() method if the scraper needs resource cleanup (like a persistent browser instance).
}

