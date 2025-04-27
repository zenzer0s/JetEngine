package storage

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	// Adjust the import path based on your go.mod file
	"jetengine/internal/domain"
)

// setupTestDB creates a temporary BadgerDB instance for testing.
// It returns the repository instance and a cleanup function.
func setupTestDB(t *testing.T) (*BadgerRepository, func()) {
	t.Helper() // Marks this function as a test helper

	// Create a temporary directory for the database
	// Using t.TempDir() automatically handles cleanup after the test (and subtests) complete.
	tempDir := t.TempDir()

	// Use a null logger for tests unless debugging is needed
	// testLogger := logrus.New()
	// testLogger.SetOutput(os.Stdout) // Or io.Discard
	// testLogger.SetLevel(logrus.DebugLevel)
	testLogger := logrus.New()
	testLogger.SetOutput(os.Stderr)        // Send logs to stderr during tests
	testLogger.SetLevel(logrus.ErrorLevel) // Only show errors by default

	repo, err := NewBadgerRepository(tempDir, testLogger)
	require.NoError(t, err, "Failed to create test BadgerDB repository") // Use require to stop test if setup fails

	// Define the cleanup function
	cleanup := func() {
		err := repo.Close()
		assert.NoError(t, err, "Failed to close test BadgerDB repository")
		// t.TempDir() handles directory removal automatically
		// If not using t.TempDir(), you would add: os.RemoveAll(tempDir)
	}

	return repo, cleanup
}

// TestBadgerRepository_SaveAndGetLinks tests saving and retrieving links.
func TestBadgerRepository_SaveAndGetLinks(t *testing.T) {
	repo, cleanup := setupTestDB(t)
	defer cleanup() // Ensure cleanup runs even if the test panics

	ctx := context.Background()
	userID1 := int64(123)
	userID2 := int64(456)

	link1 := domain.Link{
		URL:         "https://example.com/page1",
		Title:       "Example Page 1",
		Description: "Desc 1",
		UserID:      userID1,
		Timestamp:   time.Now().Add(-time.Hour), // Older timestamp
	}
	link2 := domain.Link{
		URL:         "https://example.com/page2",
		Title:       "Example Page 2",
		Description: "Desc 2",
		UserID:      userID1,
		Timestamp:   time.Now(), // Newer timestamp
	}
	link3 := domain.Link{
		URL:         "https://anothersite.net",
		Title:       "Another Site",
		Description: "Desc 3",
		UserID:      userID2,
		Timestamp:   time.Now(),
	}

	// --- Test SaveLink ---
	err := repo.SaveLink(ctx, link1)
	require.NoError(t, err, "Failed to save link1")
	err = repo.SaveLink(ctx, link2)
	require.NoError(t, err, "Failed to save link2")
	err = repo.SaveLink(ctx, link3)
	require.NoError(t, err, "Failed to save link3")

	// --- Test GetLinksByUser for userID1 ---
	linksUser1, err := repo.GetLinksByUser(ctx, userID1)
	require.NoError(t, err, "Failed to get links for user 1")
	require.Len(t, linksUser1, 2, "Expected 2 links for user 1")

	// Verify content and order (newest first due to sorting in GetLinksByUser)
	assert.Equal(t, link2.URL, linksUser1[0].URL, "First link for user 1 should be link2 (newest)")
	assert.Equal(t, link2.Title, linksUser1[0].Title)
	assert.Equal(t, link1.URL, linksUser1[1].URL, "Second link for user 1 should be link1 (older)")
	assert.Equal(t, link1.Title, linksUser1[1].Title)

	// --- Test GetLinksByUser for userID2 ---
	linksUser2, err := repo.GetLinksByUser(ctx, userID2)
	require.NoError(t, err, "Failed to get links for user 2")
	require.Len(t, linksUser2, 1, "Expected 1 link for user 2")
	assert.Equal(t, link3.URL, linksUser2[0].URL)
	assert.Equal(t, link3.Title, linksUser2[0].Title)

	// --- Test GetLinksByUser for non-existent user ---
	linksUser3, err := repo.GetLinksByUser(ctx, int64(999))
	require.NoError(t, err, "Getting links for non-existent user should not error")
	assert.Empty(t, linksUser3, "Expected no links for non-existent user")

	// --- Test Overwriting a link (SaveLink should update) ---
	updatedLink1 := domain.Link{
		URL:         link1.URL, // Same URL
		Title:       "Updated Title 1",
		Description: "Updated Desc 1",
		UserID:      userID1,
		Timestamp:   time.Now().Add(time.Minute), // Make it newest
	}
	err = repo.SaveLink(ctx, updatedLink1)
	require.NoError(t, err, "Failed to update link1")

	linksUser1AfterUpdate, err := repo.GetLinksByUser(ctx, userID1)
	require.NoError(t, err, "Failed to get links for user 1 after update")
	require.Len(t, linksUser1AfterUpdate, 2, "Expected 2 links for user 1 after update")

	// Check if the updated link is now first and has new title
	assert.Equal(t, updatedLink1.URL, linksUser1AfterUpdate[0].URL)
	assert.Equal(t, updatedLink1.Title, linksUser1AfterUpdate[0].Title)
	assert.Equal(t, link2.URL, linksUser1AfterUpdate[1].URL) // link2 should be second now
}

// TestBadgerRepository_DeleteLink tests deleting links.
func TestBadgerRepository_DeleteLink(t *testing.T) {
	repo, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	userID := int64(789)
	linkURLToDelete := "https://example.com/to_delete"
	linkURLToKeep := "https://example.com/to_keep"

	linkToDelete := domain.Link{URL: linkURLToDelete, Title: "Delete Me", UserID: userID}
	linkToKeep := domain.Link{URL: linkURLToKeep, Title: "Keep Me", UserID: userID}

	// Save both links
	err := repo.SaveLink(ctx, linkToDelete)
	require.NoError(t, err)
	err = repo.SaveLink(ctx, linkToKeep)
	require.NoError(t, err)

	// Verify both exist initially
	linksBeforeDelete, err := repo.GetLinksByUser(ctx, userID)
	require.NoError(t, err)
	require.Len(t, linksBeforeDelete, 2)

	// --- Test DeleteLink ---
	err = repo.DeleteLink(ctx, userID, linkURLToDelete)
	require.NoError(t, err, "Failed to delete link")

	// Verify the link is gone
	linksAfterDelete, err := repo.GetLinksByUser(ctx, userID)
	require.NoError(t, err, "Failed to get links after delete")
	require.Len(t, linksAfterDelete, 1, "Expected 1 link after delete")
	assert.Equal(t, linkURLToKeep, linksAfterDelete[0].URL, "The remaining link should be the one to keep")

	// --- Test Deleting a non-existent link ---
	err = repo.DeleteLink(ctx, userID, "https://example.com/does_not_exist")
	assert.NoError(t, err, "Deleting a non-existent link should not return an error")

	// Verify the list hasn't changed
	linksAfterNonExistentDelete, err := repo.GetLinksByUser(ctx, userID)
	require.NoError(t, err)
	require.Len(t, linksAfterNonExistentDelete, 1, "Link count should still be 1 after deleting non-existent link")

	// --- Test Deleting the same link again ---
	err = repo.DeleteLink(ctx, userID, linkURLToDelete)
	assert.NoError(t, err, "Deleting an already deleted link should not return an error")

	// Verify the list hasn't changed
	linksAfterDeleteAgain, err := repo.GetLinksByUser(ctx, userID)
	require.NoError(t, err)
	require.Len(t, linksAfterDeleteAgain, 1, "Link count should still be 1 after deleting again")
}

// Add more tests as needed, e.g., for error conditions like marshalling failures
// or concurrent access if that becomes relevant.
