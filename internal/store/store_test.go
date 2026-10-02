package store

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/YaswanthKumarMallela01/kosha/internal/crypto"
	"github.com/YaswanthKumarMallela01/kosha/internal/model"
)

func setupTestLibrary(t *testing.T) (*Library, string, []byte) {
	t.Helper()
	tempDir, err := os.MkdirTemp("", "kosha-store-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	pass := []byte("test-master-passphrase-123")
	salt, _ := crypto.GenerateSalt()
	key := crypto.DeriveMasterKey(pass, salt, crypto.DefaultParams())

	lib := NewLibrary(tempDir, key)
	return lib, tempDir, key
}

func TestStoreOperations(t *testing.T) {
	lib, tempDir, key := setupTestLibrary(t)
	defer os.RemoveAll(tempDir)
	defer crypto.ZeroBytes(key)

	// 1. EnsureInbox
	if err := lib.EnsureInbox(); err != nil {
		t.Fatalf("EnsureInbox failed: %v", err)
	}

	// Verify Inbox book exists in library
	if err := lib.LoadBooks(); err != nil {
		t.Fatalf("LoadBooks failed: %v", err)
	}
	if len(lib.Books) != 1 {
		t.Fatalf("expected 1 book (Inbox), got %d", len(lib.Books))
	}
	if lib.Books[0].Slug != "inbox" {
		t.Errorf("expected book slug 'inbox', got %s", lib.Books[0].Slug)
	}

	// 2. QuickCapture
	if err := lib.QuickCapture("My quick thought\nWith some details."); err != nil {
		t.Fatalf("QuickCapture failed: %v", err)
	}

	// 3. Load Chapters
	chaps, err := lib.LoadChapters("inbox")
	if err != nil {
		t.Fatalf("LoadChapters failed: %v", err)
	}
	if len(chaps) != 1 {
		t.Fatalf("expected 1 chapter in Inbox, got %d", len(chaps))
	}
	qcChap := chaps[0]
	if len(qcChap.Notes) != 1 {
		t.Fatalf("expected 1 note in Quick Capture, got %d", len(qcChap.Notes))
	}
	if qcChap.Notes[0].Title != "My quick thought" {
		t.Errorf("note title mismatch: got %s", qcChap.Notes[0].Title)
	}

	// 4. Create custom Book & Chapter & Note
	customBook, err := lib.CreateBook("Work Projects")
	if err != nil {
		t.Fatalf("CreateBook failed: %v", err)
	}
	if customBook.Slug != "work-projects" {
		t.Errorf("expected slug 'work-projects', got %s", customBook.Slug)
	}

	customChap, err := lib.CreateChapter(customBook.Slug, "Q4 Roadmap")
	if err != nil {
		t.Fatalf("CreateChapter failed: %v", err)
	}

	note := &model.Note{
		ID:        model.NewID(),
		Title:     "API Redesign",
		Body:      "Plan to use **XChaCha20-Poly1305** for #security.",
		CreatedAt: time.Now().Add(-10 * 24 * time.Hour), // 10 days old for resurface test
		UpdatedAt: time.Now(),
		Pinned:    true,
	}
	if err := lib.AddNote(customBook.Slug, customChap.ID, note); err != nil {
		t.Fatalf("AddNote failed: %v", err)
	}

	// 5. GetAllNotes
	allNotes := lib.GetAllNotes()
	if len(allNotes) != 2 {
		t.Errorf("expected 2 notes total, got %d", len(allNotes))
	}

	// 6. FindNoteByTitle
	ref, err := lib.FindNoteByTitle("API Redesign", customChap.ID, customBook.Slug)
	if err != nil || ref == nil {
		t.Fatalf("FindNoteByTitle failed: %v", err)
	}
	if ref.Note.Title != "API Redesign" {
		t.Errorf("note title mismatch: got %s", ref.Note.Title)
	}

	// 7. UpdateNote
	note.Body = "Updated body content with #crypto tag."
	if err := lib.UpdateNote(customBook.Slug, customChap.ID, note); err != nil {
		t.Fatalf("UpdateNote failed: %v", err)
	}

	// Verify update persisted
	reloadedChap, err := lib.GetChapter(customBook.Slug, customChap.ID)
	if err != nil {
		t.Fatalf("GetChapter failed: %v", err)
	}
	if reloadedChap.Notes[0].Body != note.Body {
		t.Errorf("updated body mismatch: got %s", reloadedChap.Notes[0].Body)
	}

	// 8. Snapshot creation
	if err := lib.CreateSnapshot(customBook.Slug, customChap.ID); err != nil {
		t.Fatalf("CreateSnapshot failed: %v", err)
	}

	// Verify snapshot directory exists
	snapDir := filepath.Join(tempDir, "books", customBook.Slug, ".snapshots", customChap.ID)
	entries, err := os.ReadDir(snapDir)
	if err != nil || len(entries) == 0 {
		t.Fatalf("expected snapshot file in %s, got err: %v", snapDir, err)
	}

	// 9. Lock
	lib.Lock()
	if lib.MasterKey != nil {
		t.Errorf("expected MasterKey to be nil after Lock")
	}
	if len(lib.Books) != 0 {
		t.Errorf("expected Books slice to be cleared after Lock")
	}
}
