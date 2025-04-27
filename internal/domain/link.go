package domain

import "time"

// Link represents the core data structure for a saved website link.
type Link struct {
	// URL is the unique identifier for the link (and the primary key conceptually).
	URL string `json:"url" bson:"url"`

	// Title scraped from the website's <title> tag.
	Title string `json:"title" bson:"title"`

	// Description scraped from the website's meta description tag.
	Description string `json:"description" bson:"description"`

	// UserID is the Telegram User ID of the user who saved the link.
	UserID int64 `json:"user_id" bson:"user_id"`

	// Timestamp indicates when the link was saved.
	Timestamp time.Time `json:"timestamp" bson:"timestamp"`

	// Tags is an optional list of tags for categorizing the link.
	Tags []string `json:"tags,omitempty" bson:"tags,omitempty"`

	// Read indicates whether the user has marked the link as read.
	Read bool `json:"read" bson:"read"`

	// PreviewImageURL is an optional URL to a preview image (e.g., Open Graph image).
	PreviewImageURL string `json:"preview_image_url,omitempty" bson:"preview_image_url,omitempty"`
}

// Note: Add methods (e.g., validation) and corresponding unit tests in internal/domain/link_test.go as needed.
