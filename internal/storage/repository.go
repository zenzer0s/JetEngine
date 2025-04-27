package storage

import (
	"context"

	"jetengine/internal/domain"
)

// Repository defines the interface for data storage operations.
// This allows us to swap storage implementations (e.g., BadgerDB, PostgreSQL)
// without changing the core application logic that uses it.
type Repository interface {
	// SaveLink stores a new link or updates an existing one for a specific user.
	// The combination of UserID and link.URL should be unique.
	SaveLink(ctx context.Context, link domain.Link) error

	// GetLinksByUser retrieves all links saved by a specific user, ordered perhaps by timestamp.
	GetLinksByUser(ctx context.Context, userID int64) ([]domain.Link, error)

	// DeleteLink removes a specific link for a given user.
	DeleteLink(ctx context.Context, userID int64, linkURL string) error

	// Close gracefully shuts down the repository connection.
	Close() error
}
