package scraper

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/sirupsen/logrus"
)

// RodScraper implements the Scraper interface using the rod library.
type RodScraper struct {
	log logrus.FieldLogger
	// browser *rod.Browser // Optional: Keep a persistent browser instance
}

// NewRodScraper creates a new scraper service instance.
func NewRodScraper(logger logrus.FieldLogger) *RodScraper {
	// Optional: Initialize a persistent browser here if desired
	// path, _ := launcher.LookPath()
	// u := launcher.New().Bin(path).MustLaunch()
	// browser := rod.New().ControlURL(u).MustConnect()
	// logger.Info("Persistent rod browser instance created")

	return &RodScraper{
		log: logger.WithField("component", "scraper"),
		// browser: browser, // Assign if using persistent browser
	}
}

// Optional: Add a Close method if using a persistent browser
// func (s *RodScraper) Close() error {
// 	if s.browser != nil {
// 		s.log.Info("Closing persistent rod browser instance")
// 		return s.browser.Close()
// 	}
// 	return nil
// }

// ScrapeMetadata fetches the title and description using rod.
func (s *RodScraper) ScrapeMetadata(ctx context.Context, url string) (title string, description string, err error) {
	log := s.log.WithField("url", url)
	log.Info("Attempting to scrape metadata")

	// --- Browser Setup ---
	// Option 1: Launch a new browser for each scrape (simpler, more resource-intensive)
	path, exists := launcher.LookPath()
	if !exists {
		log.Error("Cannot find browser executable for rod")
		return "", "", errors.New("rod browser dependency not found")
	}
	// Use launcher.New().Headless(false).MustLaunch() to see the browser window for debugging
	u := launcher.New().Bin(path).MustLaunch()
	browser := rod.New().ControlURL(u)
	err = browser.Connect()
	if err != nil {
		log.WithError(err).Error("Failed to connect to rod browser")
		return "", "", fmt.Errorf("failed to connect to browser: %w", err)
	}
	// Ensure the browser is closed when the function exits
	defer func() {
		closeErr := browser.Close()
		if closeErr != nil {
			log.WithError(closeErr).Error("Error closing rod browser instance")
			// Decide if this error should overwrite the primary return error
			if err == nil {
				err = fmt.Errorf("error closing browser: %w", closeErr)
			}
		} else {
			log.Debug("Rod browser instance closed")
		}
	}()

	// Option 2: Use a persistent browser instance (more complex setup/cleanup)
	// if s.browser == nil {
	// 	log.Error("Persistent browser instance is not initialized")
	// 	return "", "", errors.New("scraper not initialized correctly")
	// }
	// browser := s.browser
	// --- End Browser Setup ---

	// --- Page Navigation and Scraping ---
	var page *rod.Page
	page, err = browser.Page(proto.TargetCreateTarget{URL: url}) // Use Page for simpler navigation
	if err != nil {
		log.WithError(err).Error("Failed to create rod page")
		return "", "", fmt.Errorf("failed to create page: %w", err)
	}
	// Ensure page is closed
	defer func() {
		closeErr := page.Close()
		if closeErr != nil {
			log.WithError(closeErr).Error("Error closing rod page")
			if err == nil {
				err = fmt.Errorf("error closing page: %w", closeErr)
			}
		} else {
			log.Debug("Rod page closed")
		}
	}()

	// Set a timeout for the entire scraping operation for this page
	// Use a context that respects the overall request context if available
	pageCtx, cancel := context.WithTimeout(ctx, 30*time.Second) // 30-second timeout
	defer cancel()
	page = page.Context(pageCtx)

	// Wait for the page to load completely (adjust wait condition if needed)
	err = page.WaitLoad()
	if err != nil {
		// Handle context deadline exceeded specifically
		if errors.Is(pageCtx.Err(), context.DeadlineExceeded) {
			log.WithError(pageCtx.Err()).Warn("Scraping timed out")
			return "", "", fmt.Errorf("scraping timed out for %s: %w", url, pageCtx.Err())
		}
		log.WithError(err).Error("Failed to wait for page load")
		return "", "", fmt.Errorf("failed waiting for page load: %w", err)
	}

	// --- Extract Title ---
	titleElement, err := page.Element("title")
	if err != nil {
		// It's possible a page might not have a title, treat as warning?
		log.WithError(err).Warn("Could not find title element")
		title = "" // Default to empty title
	} else {
		title, err = titleElement.Text()
		if err != nil {
			log.WithError(err).Error("Failed to get text from title element")
			// Don't return error, just use empty title maybe?
			title = ""
		}
		title = strings.TrimSpace(title)
		log.WithField("title", title).Debug("Extracted title")
	}

	// --- Extract Description ---
	// Try common meta description tags
	descSelectors := []string{
		`meta[name="description"]`,
		`meta[property="og:description"]`,
		// Add more selectors if needed
	}
	description = "" // Default
	for _, selector := range descSelectors {
		descElement, err := page.Element(selector)
		if err == nil { // Found an element
			descContent, err := descElement.Attribute("content")
			if err == nil && descContent != nil {
				description = strings.TrimSpace(*descContent)
				if description != "" {
					log.WithField("description", description).Debug("Extracted description")
					break // Stop searching once a non-empty description is found
				}
			} else if err != nil {
				log.WithError(err).WithField("selector", selector).Warn("Failed to get content attribute from meta tag")
			}
		} else if !strings.Contains(err.Error(), "element not found") { // err is guaranteed non-nil here
			log.WithError(err).WithField("selector", selector).Warn("Error searching for meta description tag")
		}
	}
	if description == "" {
		log.Warn("Could not find description meta tag")
	}

	log.Info("Metadata scraping completed successfully")
	// Return the extracted title and description, err should be nil here if successful
	return title, description, nil
}
