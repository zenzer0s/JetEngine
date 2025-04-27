package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/dgraph-io/badger/v4"
	"github.com/sirupsen/logrus"

	// Adjust the import path based on your go.mod file
	"jetengine/internal/domain"
)

// BadgerRepository implements the Repository interface using BadgerDB.
type BadgerRepository struct {
	db  *badger.DB
	log logrus.FieldLogger
}

// NewBadgerRepository creates and initializes a new BadgerDB repository.
// It opens the database at the specified path.
func NewBadgerRepository(dbPath string, logger logrus.FieldLogger) (*BadgerRepository, error) {
	opts := badger.DefaultOptions(dbPath)
	// Add logger to Badger options for internal logging
	opts.Logger = &badgerLogger{logger.WithField("component", "badgerdb")}

	db, err := badger.Open(opts)
	if err != nil {
		logger.WithError(err).Error("Failed to open BadgerDB")
		return nil, fmt.Errorf("failed to open badger db at %s: %w", dbPath, err)
	}
	logger.Info("BadgerDB opened successfully at path: ", dbPath)

	repo := &BadgerRepository{
		db:  db,
		log: logger.WithField("component", "repository"), // Add component field to repo logs
	}

	// Optional: Start garbage collection routine
	// Consider making GC interval configurable
	// go repo.runGC(ctx) // Need to manage context/cancellation for this goroutine

	return repo, nil
}

// Close closes the BadgerDB database connection.
func (r *BadgerRepository) Close() error {
	r.log.Info("Closing BadgerDB...")
	err := r.db.Close()
	if err != nil {
		r.log.WithError(err).Error("Error closing BadgerDB")
		return err
	}
	r.log.Info("BadgerDB closed.")
	return nil
}

// generateLinkKey creates a unique key for storing a link.
// Format: user:{userID}:link:{linkURL}
func generateLinkKey(userID int64, linkURL string) []byte {
	return []byte(fmt.Sprintf("user:%d:link:%s", userID, linkURL))
}

// generateUserPrefix creates a key prefix for scanning all links belonging to a user.
// Format: user:{userID}:link:
func generateUserPrefix(userID int64) []byte {
	return []byte(fmt.Sprintf("user:%d:link:", userID))
}

// SaveLink stores or updates a link in BadgerDB.
func (r *BadgerRepository) SaveLink(ctx context.Context, link domain.Link) error {
	log := r.log.WithFields(logrus.Fields{
		"user_id": link.UserID,
		"url":     link.URL,
	})
	log.Info("Attempting to save link")

	// Ensure timestamp is set
	if link.Timestamp.IsZero() {
		link.Timestamp = time.Now()
	}

	// Serialize the link struct to JSON bytes
	linkBytes, err := json.Marshal(link)
	if err != nil {
		log.WithError(err).Error("Failed to marshal link to JSON")
		return fmt.Errorf("failed to marshal link: %w", err)
	}

	// Generate the unique key for this link
	key := generateLinkKey(link.UserID, link.URL)

	// Perform the save operation within a transaction
	err = r.db.Update(func(txn *badger.Txn) error {
		// Set the key-value pair. This will overwrite if the key already exists.
		// Consider adding TTL (Time To Live) if needed: e := badger.NewEntry(key, linkBytes).WithTTL(time.Hour)
		e := badger.NewEntry(key, linkBytes)
		return txn.SetEntry(e)
	})

	if err != nil {
		log.WithError(err).Error("Failed to save link to BadgerDB")
		return fmt.Errorf("failed to save link: %w", err)
	}

	log.Info("Link saved successfully")
	return nil
}

// GetLinksByUser retrieves all links for a specific user.
func (r *BadgerRepository) GetLinksByUser(ctx context.Context, userID int64) ([]domain.Link, error) {
	log := r.log.WithField("user_id", userID)
	log.Info("Attempting to get links for user")

	var links []domain.Link

	// Start a read-only transaction
	err := r.db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()

		// Generate the prefix for the user's links
		prefix := generateUserPrefix(userID)

		// Iterate over keys with the specified prefix
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()
			err := item.Value(func(val []byte) error {
				var link domain.Link
				// Make a copy of the value slice before unmarshalling
				valCopy := make([]byte, len(val))
				copy(valCopy, val)
				if err := json.Unmarshal(valCopy, &link); err != nil {
					log.WithError(err).WithField("key", string(item.Key())).Error("Failed to unmarshal link from DB")
					// Decide whether to skip this item or return an error for the whole operation
					return fmt.Errorf("failed to unmarshal link data for key %s: %w", string(item.Key()), err)
				}
				links = append(links, link)
				return nil
			})
			if err != nil {
				// Handle error retrieving value or unmarshalling
				return err // Stop iteration on error
			}
		}
		return nil
	})

	if err != nil {
		log.WithError(err).Error("Failed to retrieve links from BadgerDB")
		return nil, fmt.Errorf("failed to get links for user %d: %w", userID, err)
	}

	// Sort links by timestamp (newest first) before returning
	sort.Slice(links, func(i, j int) bool {
		return links[i].Timestamp.After(links[j].Timestamp)
	})

	log.WithField("link_count", len(links)).Info("Links retrieved successfully")
	return links, nil
}

// DeleteLink removes a specific link for a user.
func (r *BadgerRepository) DeleteLink(ctx context.Context, userID int64, linkURL string) error {
	log := r.log.WithFields(logrus.Fields{
		"user_id": userID,
		"url":     linkURL,
	})
	log.Info("Attempting to delete link")

	key := generateLinkKey(userID, linkURL)

	// Perform the delete operation within a transaction
	err := r.db.Update(func(txn *badger.Txn) error {
		// Check if the item exists before deleting (optional, Delete is idempotent)
		// _, err := txn.Get(key)
		// if err == badger.ErrKeyNotFound {
		//  log.Warn("Attempted to delete non-existent link")
		//  return nil // Or return a specific "not found" error if needed
		// } else if err != nil {
		//  return err // Propagate other errors
		// }
		return txn.Delete(key)
	})

	if err != nil {
		log.WithError(err).Error("Failed to delete link from BadgerDB")
		return fmt.Errorf("failed to delete link %s for user %d: %w", linkURL, userID, err)
	}

	log.Info("Link deleted successfully")
	return nil
}

// --- BadgerDB Internal Logger ---

// badgerLogger adapts logrus.FieldLogger to Badger's logger interface.
type badgerLogger struct {
	logger logrus.FieldLogger
}

func (l *badgerLogger) Errorf(f string, v ...interface{}) {
	l.logger.Errorf(f, v...)
}
func (l *badgerLogger) Warningf(f string, v ...interface{}) {
	l.logger.Warningf(f, v...)
}
func (l *badgerLogger) Infof(f string, v ...interface{}) {
	l.logger.Infof(f, v...)
}
func (l *badgerLogger) Debugf(f string, v ...interface{}) {
	l.logger.Debugf(f, v...)
}

// --- Optional: Background Garbage Collection ---
// BadgerDB requires periodic garbage collection (GC) to reclaim disk space.
// func (r *BadgerRepository) runGC(ctx context.Context) {
// 	// Consider making the ticker duration configurable
// 	ticker := time.NewTicker(5 * time.Minute) // Run GC every 5 minutes
// 	defer ticker.Stop()
// 	for {
// 		select {
// 		case <-ticker.C:
// 			err := r.db.RunValueLogGC(0.7) // 0.7 is a recommended threshold
// 			if err != nil {
// 				// Log GC errors, but don't necessarily stop the loop unless it's badger.ErrDBClosed
// 				if err == badger.ErrNoRewrite {
// 					r.log.Debug("BadgerDB GC: No rewrite needed")
// 				} else {
// 					r.log.WithError(err).Error("BadgerDB GC failed")
// 				}
// 			} else {
// 				r.log.Info("BadgerDB GC completed successfully")
// 			}
// 		case <-ctx.Done(): // Ensure GC stops when the application context is cancelled
// 			r.log.Info("Stopping BadgerDB GC routine due to context cancellation")
// 			return
// 		}
// 	}
// }
