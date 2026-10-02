package vault

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/YaswanthKumarMallela01/kosha/internal/crypto"
	"github.com/YaswanthKumarMallela01/kosha/internal/model"
)

// SnapshotInfo holds information about a saved chapter snapshot.
type SnapshotInfo struct {
	Path      string
	Timestamp time.Time
}

// AtomicWrite writes data to a temporary file and atomically renames it to path.
// It also creates a backup of the original file if it exists, and sets 0600 permissions.
func AtomicWrite(path string, data []byte) error {
	tmpPath := path + ".tmp"
	f, err := os.OpenFile(tmpPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpPath)

	if _, err := f.Write(data); err != nil {
		f.Close()
		return fmt.Errorf("failed to write data: %w", err)
	}

	if err := f.Sync(); err != nil {
		f.Close()
		return fmt.Errorf("failed to sync temp file: %w", err)
	}

	if err := f.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	// Create a backup if the target file already exists
	if _, err := os.Stat(path); err == nil {
		bakPath := path + ".bak"
		if err := copyFile(path, bakPath); err != nil {
			return fmt.Errorf("failed to backup existing file: %w", err)
		}
	}

	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("failed to rename temp file to target: %w", err)
	}
	
	if err := os.Chmod(path, 0600); err != nil {
		return fmt.Errorf("failed to set permissions: %w", err)
	}

	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	
	if err := out.Sync(); err != nil {
		return err
	}
	
	return nil
}

// WriteVault encrypts the plaintext using the masterKey and writes it to path.
func WriteVault(path string, plaintext []byte, masterKey []byte) error {
	encrypted, err := crypto.Seal(plaintext, masterKey)
	if err != nil {
		return fmt.Errorf("failed to encrypt vault data: %w", err)
	}
	
	if err := AtomicWrite(path, encrypted); err != nil {
		return fmt.Errorf("failed to write vault file: %w", err)
	}
	
	return nil
}

// ReadVault reads the file at path and decrypts it using the masterKey.
func ReadVault(path string, masterKey []byte) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read vault file: %w", err)
	}
	
	plaintext, err := crypto.Open(data, masterKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt vault data: %w", err)
	}
	
	return plaintext, nil
}

// SaveChapter serializes a Chapter, encrypts it, and saves it as a .vault file.
func SaveChapter(bookDir string, chapter *model.Chapter, masterKey []byte) error {
	payload := model.ChapterPayload{
		ID:              chapter.ID,
		Title:           chapter.Title,
		Notes:           chapter.Notes,
		CreatedAt:       chapter.CreatedAt,
		ProcessedHashes: chapter.ProcessedHashes,
	}
	
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal chapter: %w", err)
	}
	
	path := filepath.Join(bookDir, chapter.ID+".vault")
	return WriteVault(path, data, masterKey)
}

// LoadChapter reads and decrypts a .vault file, returning the parsed Chapter.
func LoadChapter(path string, masterKey []byte) (*model.Chapter, error) {
	data, err := ReadVault(path, masterKey)
	if err != nil {
		return nil, fmt.Errorf("failed to load chapter vault: %w", err)
	}
	
	var payload model.ChapterPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, fmt.Errorf("failed to unmarshal chapter payload: %w", err)
	}
	
	chapter := &model.Chapter{
		ID:              payload.ID,
		Title:           payload.Title,
		Notes:           payload.Notes,
		CreatedAt:       payload.CreatedAt,
		ProcessedHashes: payload.ProcessedHashes,
	}
	
	return chapter, nil
}

// SaveBookMeta encrypts and saves BookMeta data to book.meta.
func SaveBookMeta(bookDir string, meta *model.BookMeta, masterKey []byte) error {
	data, err := json.Marshal(meta)
	if err != nil {
		return fmt.Errorf("failed to marshal book meta: %w", err)
	}
	
	path := filepath.Join(bookDir, "book.meta")
	return WriteVault(path, data, masterKey)
}

// LoadBookMeta reads and decrypts BookMeta data from book.meta.
func LoadBookMeta(bookDir string, masterKey []byte) (*model.BookMeta, error) {
	path := filepath.Join(bookDir, "book.meta")
	data, err := ReadVault(path, masterKey)
	if err != nil {
		return nil, fmt.Errorf("failed to load book meta vault: %w", err)
	}
	
	var meta model.BookMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, fmt.Errorf("failed to unmarshal book meta: %w", err)
	}
	
	return &meta, nil
}

// SaveSnapshot encrypts and saves a raw payload as a snapshot for a specific chapter.
// It also prunes old snapshots, keeping only the most recent 20.
func SaveSnapshot(bookDir string, chapterID string, payload []byte, masterKey []byte) error {
	snapshotDir := filepath.Join(bookDir, ".snapshots", chapterID)
	if err := os.MkdirAll(snapshotDir, 0700); err != nil {
		return fmt.Errorf("failed to create snapshot dir: %w", err)
	}
	
	timestamp := time.Now().Format("20060102T150405")
	filename := timestamp + ".vault"
	path := filepath.Join(snapshotDir, filename)
	
	if err := WriteVault(path, payload, masterKey); err != nil {
		return fmt.Errorf("failed to write snapshot: %w", err)
	}
	
	// Prune old snapshots (keep last 20)
	snapshots, err := ListSnapshots(bookDir, chapterID)
	if err != nil {
		return fmt.Errorf("failed to list snapshots for pruning: %w", err)
	}
	
	if len(snapshots) > 20 {
		for i := 0; i < len(snapshots)-20; i++ {
			os.Remove(snapshots[i].Path)
		}
	}
	
	return nil
}

// ListSnapshots returns a time-sorted list of snapshots available for a given chapter.
func ListSnapshots(bookDir string, chapterID string) ([]SnapshotInfo, error) {
	snapshotDir := filepath.Join(bookDir, ".snapshots", chapterID)
	entries, err := os.ReadDir(snapshotDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read snapshot dir: %w", err)
	}
	
	var snapshots []SnapshotInfo
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".vault") {
			continue
		}
		
		name := strings.TrimSuffix(entry.Name(), ".vault")
		t, err := time.Parse("20060102T150405", name)
		if err != nil {
			continue // skip invalid filenames
		}
		
		snapshots = append(snapshots, SnapshotInfo{
			Path:      filepath.Join(snapshotDir, entry.Name()),
			Timestamp: t,
		})
	}
	
	sort.Slice(snapshots, func(i, j int) bool {
		return snapshots[i].Timestamp.Before(snapshots[j].Timestamp)
	})
	
	return snapshots, nil
}

// LoadSnapshot reads and parses a snapshot .vault file.
func LoadSnapshot(path string, masterKey []byte) (*model.Chapter, error) {
	return LoadChapter(path, masterKey)
}

// ListChapterFiles lists all .vault files in the given book directory (excluding book.meta).
func ListChapterFiles(bookDir string) ([]string, error) {
	entries, err := os.ReadDir(bookDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read book dir: %w", err)
	}
	
	var files []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if entry.Name() == "book.meta" || !strings.HasSuffix(entry.Name(), ".vault") {
			continue
		}
		files = append(files, filepath.Join(bookDir, entry.Name()))
	}
	
	sort.Strings(files)
	return files, nil
}
