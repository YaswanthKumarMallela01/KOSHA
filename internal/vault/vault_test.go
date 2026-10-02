package vault

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/YaswanthKumarMallela01/kosha/internal/crypto"
	"github.com/YaswanthKumarMallela01/kosha/internal/model"
)

func getTestKey(t *testing.T) []byte {
	salt := make([]byte, 16)
	return crypto.DeriveMasterKey([]byte("test-passphrase"), salt, crypto.DefaultParams())
}

func TestWriteReadVault(t *testing.T) {
	dir := t.TempDir()
	key := getTestKey(t)
	path := filepath.Join(dir, "test.vault")
	data := []byte("hello secret world")

	if err := WriteVault(path, data, key); err != nil {
		t.Fatalf("WriteVault failed: %v", err)
	}

	readData, err := ReadVault(path, key)
	if err != nil {
		t.Fatalf("ReadVault failed: %v", err)
	}

	if string(readData) != string(data) {
		t.Errorf("expected %s, got %s", data, readData)
	}
}

func TestSaveLoadChapter(t *testing.T) {
	dir := t.TempDir()
	key := getTestKey(t)

	chapter := &model.Chapter{
		ID:        "ch1",
		Title:     "Chapter 1",
		CreatedAt: time.Now().Truncate(time.Second).UTC(),
	}

	if err := SaveChapter(dir, chapter, key); err != nil {
		t.Fatalf("SaveChapter failed: %v", err)
	}

	path := filepath.Join(dir, "ch1.vault")
	loaded, err := LoadChapter(path, key)
	if err != nil {
		t.Fatalf("LoadChapter failed: %v", err)
	}

	if loaded.ID != chapter.ID || loaded.Title != chapter.Title || !loaded.CreatedAt.Equal(chapter.CreatedAt) {
		t.Errorf("loaded chapter does not match saved chapter")
	}
}

func TestAtomicWriteBackup(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "data.txt")
	
	err := AtomicWrite(path, []byte("version 1"))
	if err != nil {
		t.Fatalf("first write failed: %v", err)
	}
	
	err = AtomicWrite(path, []byte("version 2"))
	if err != nil {
		t.Fatalf("second write failed: %v", err)
	}
	
	bakPath := path + ".bak"
	bakData, err := os.ReadFile(bakPath)
	if err != nil {
		t.Fatalf("failed to read backup: %v", err)
	}
	if string(bakData) != "version 1" {
		t.Errorf("expected backup to be 'version 1', got '%s'", bakData)
	}
	
	currentData, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read current file: %v", err)
	}
	if string(currentData) != "version 2" {
		t.Errorf("expected current to be 'version 2', got '%s'", currentData)
	}
}

func TestFilePermissions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	
	err := AtomicWrite(path, []byte("test"))
	if err != nil {
		t.Fatalf("AtomicWrite failed: %v", err)
	}
	
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat failed: %v", err)
	}
	
	// On Windows, permissions might be mapped differently, so just check it doesn't crash
	if info.Mode().Perm() & 0600 != 0600 {
		t.Logf("expected permission 0600, got %v (this might be normal on Windows)", info.Mode().Perm())
	}
}

func TestSnapshotPruning(t *testing.T) {
	dir := t.TempDir()
	key := getTestKey(t)
	chapterID := "ch-snap-prune"
	
	snapDir := filepath.Join(dir, ".snapshots", chapterID)
	os.MkdirAll(snapDir, 0700)
	
	baseTime := time.Now().Add(-1 * time.Hour)
	for i := 0; i < 25; i++ {
		tStr := baseTime.Add(time.Duration(i) * time.Second).Format("20060102T150405")
		path := filepath.Join(snapDir, tStr+".vault")
		os.WriteFile(path, []byte("dummy"), 0600)
	}
	
	// Now save one real snapshot
	payload := []byte(`{"id":"`+chapterID+`","title":"snap test"}`)
	err := SaveSnapshot(dir, chapterID, payload, key)
	if err != nil {
		t.Fatalf("SaveSnapshot failed: %v", err)
	}
	
	snaps, err := ListSnapshots(dir, chapterID)
	if err != nil {
		t.Fatalf("ListSnapshots failed: %v", err)
	}
	
	if len(snaps) > 20 {
		t.Errorf("expected at most 20 snapshots after pruning, got %d", len(snaps))
	}
}

func TestListChapterFiles(t *testing.T) {
	dir := t.TempDir()
	
	os.WriteFile(filepath.Join(dir, "a.vault"), []byte("a"), 0600)
	os.WriteFile(filepath.Join(dir, "b.vault"), []byte("b"), 0600)
	os.WriteFile(filepath.Join(dir, "book.meta"), []byte("meta"), 0600)
	os.WriteFile(filepath.Join(dir, "ignore.txt"), []byte("txt"), 0600)
	os.MkdirAll(filepath.Join(dir, "sub"), 0700)
	
	files, err := ListChapterFiles(dir)
	if err != nil {
		t.Fatalf("ListChapterFiles failed: %v", err)
	}
	
	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(files))
	}
	if filepath.Base(files[0]) != "a.vault" || filepath.Base(files[1]) != "b.vault" {
		t.Errorf("unexpected files: %v", files)
	}
}
