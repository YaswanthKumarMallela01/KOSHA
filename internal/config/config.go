package config

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/joho/godotenv"
	"github.com/YaswanthKumarMallela01/kosha/internal/crypto"
)

// Base64Bytes is a helper type for JSON marshaling/unmarshaling of byte slices as base64 strings.
type Base64Bytes []byte

// MarshalJSON encodes the byte slice as a base64 JSON string.
func (b Base64Bytes) MarshalJSON() ([]byte, error) {
	if b == nil {
		return []byte("null"), nil
	}
	return json.Marshal(base64.StdEncoding.EncodeToString(b))
}

// UnmarshalJSON decodes a base64 JSON string into a byte slice.
func (b *Base64Bytes) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	decoded, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return err
	}
	*b = decoded
	return nil
}

// Config represents the application configuration.
type Config struct {
	DataDir      string             `json:"-"`
	MasterSalt   Base64Bytes        `json:"masterSalt"`
	ArgonParams  crypto.ArgonParams `json:"argonParams"`
	KeyCheck     Base64Bytes        `json:"keyCheck"`
	GeminiAPIKey string             `json:"-"`
	GeminiModel  string             `json:"-"`
	LockMinutes  int                `json:"-"`
}

// DefaultDataDir returns the default directory for Kosha data.
func DefaultDataDir() (string, error) {
	if dir := os.Getenv("KOSHA_HOME"); dir != "" {
		return dir, nil
	}
	// Check if D:\ exists (preferred location: D:\books)
	if _, err := os.Stat(`D:\`); err == nil {
		return `D:\books`, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".kosha"), nil
}

// LoadEnv loads environment variables from .env files.
func LoadEnv() {
	dataDir, err := DefaultDataDir()
	if err == nil {
		dataDirEnv := filepath.Join(dataDir, ".env")
		_ = godotenv.Load(dataDirEnv)
	}
	_ = godotenv.Overload(".env")
}

// LoadConfig loads the configuration from the config file and environment variables.
func LoadConfig() (*Config, error) {
	LoadEnv()

	dataDir, err := DefaultDataDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get data dir: %w", err)
	}

	configPath := filepath.Join(dataDir, "config.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config json: %w", err)
	}

	cfg.DataDir = dataDir
	cfg.GeminiAPIKey = os.Getenv("GEMINI_API_KEY")

	model := os.Getenv("GEMINI_MODEL")
	if model == "" {
		model = "models/gemini-flash-latest"
	}
	cfg.GeminiModel = model

	lockMinStr := os.Getenv("KOSHA_LOCK_MINUTES")
	lockMin := 5
	if lockMinStr != "" {
		if parsed, err := strconv.Atoi(lockMinStr); err == nil {
			lockMin = parsed
		}
	}
	cfg.LockMinutes = lockMin

	return &cfg, nil
}

// SaveConfig saves the configuration to the config file.
func SaveConfig(cfg *Config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if cfg.DataDir == "" {
		return fmt.Errorf("config DataDir is empty")
	}

	configPath := filepath.Join(cfg.DataDir, "config.json")
	tmpPath := configPath + ".tmp"

	if err := os.WriteFile(tmpPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write tmp config file: %w", err)
	}

	if err := os.Rename(tmpPath, configPath); err != nil {
		os.Remove(tmpPath) // clean up on rename failure
		return fmt.Errorf("failed to rename config file: %w", err)
	}

	return nil
}

// ConfigExists returns true if the configuration file exists.
func ConfigExists() bool {
	dataDir, err := DefaultDataDir()
	if err != nil {
		return false
	}
	configPath := filepath.Join(dataDir, "config.json")
	_, err = os.Stat(configPath)
	return err == nil
}

// EnsureDataDir ensures that the data directory and its subdirectories exist.
func EnsureDataDir(dir string) error {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create data dir: %w", err)
	}
	booksDir := BooksDir(dir)
	if err := os.MkdirAll(booksDir, 0700); err != nil {
		return fmt.Errorf("failed to create books dir: %w", err)
	}
	return nil
}

// BooksDir returns the path to the books directory.
func BooksDir(dataDir string) string {
	clean := filepath.Clean(dataDir)
	if filepath.Base(clean) == "books" || filepath.Base(clean) == "Books" {
		return clean
	}
	return filepath.Join(clean, "books")
}

// BookDir returns the path to a specific book's directory.
func BookDir(dataDir, slug string) string {
	return filepath.Join(BooksDir(dataDir), slug)
}

// SnapshotsDir returns the path to a chapter's snapshots directory.
func SnapshotsDir(bookDir, chapterID string) string {
	return filepath.Join(bookDir, ".snapshots", chapterID)
}
