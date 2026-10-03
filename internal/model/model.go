package model

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/oklog/ulid/v2"
)

// Book represents a top-level collection of chapters.
type Book struct {
	Slug        string
	DisplayName string
	CreatedAt   time.Time
	Path        string
}

// BookMeta is the JSON payload for a book's metadata.
type BookMeta struct {
	DisplayName string    `json:"displayName"`
	CreatedAt   time.Time `json:"createdAt"`
}

// Chapter represents a collection of notes within a book.
type Chapter struct {
	ID              string
	Title           string
	Notes           []*Note
	CreatedAt       time.Time
	ProcessedHashes map[string]bool
}

// ChapterPayload is the JSON-serializable representation of a Chapter.
type ChapterPayload struct {
	ID              string          `json:"id"`
	Title           string          `json:"title"`
	Notes           []*Note         `json:"notes"`
	CreatedAt       time.Time       `json:"createdAt"`
	ProcessedHashes map[string]bool `json:"processedHashes,omitempty"`
}

// NoteImage represents an embedded image in a note.
type NoteImage struct {
	ID    string `json:"id"`
	Path  string `json:"path"`  // absolute or relative file path
	Alt   string `json:"alt"`   // alt text / caption
	Align string `json:"align"` // "left", "center", "right"
	Width int    `json:"width"` // width percentage (10-100), 0 = auto
}

// Note represents an individual note within a chapter.
type Note struct {
	ID        string       `json:"id"`
	Title     string       `json:"title"`
	Body      string       `json:"body"`
	CreatedAt time.Time    `json:"createdAt"`
	UpdatedAt time.Time    `json:"updatedAt"`
	Pinned    bool         `json:"pinned"`
	Images    []*NoteImage `json:"images,omitempty"`
}

// Block represents a single piece of content (paragraph, heading, etc) for AI processing.
type Block struct {
	ID   string `json:"id"`
	Text string `json:"text"`
	Hash string `json:"-"`
}

// NewID generates a new ULID string.
func NewID() string {
	entropy := ulid.Monotonic(rand.Reader, 0)
	return ulid.MustNew(ulid.Timestamp(time.Now()), entropy).String()
}

// Slugify converts a display name into a URL-safe slug.
func Slugify(name string) string {
	name = strings.ToLower(name)
	var builder strings.Builder
	for _, r := range name {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			builder.WriteRune(r)
		} else if r == ' ' || r == '-' || r == '_' {
			builder.WriteRune('-')
		}
	}
	slug := builder.String()
	for strings.Contains(slug, "--") {
		slug = strings.ReplaceAll(slug, "--", "-")
	}
	return strings.Trim(slug, "-")
}

// HashBlock computes the SHA-256 hash of a string and returns it as hex.
func HashBlock(text string) string {
	hash := sha256.Sum256([]byte(text))
	return hex.EncodeToString(hash[:])
}

// SplitBlocks splits a note body into blocks.
func SplitBlocks(body string) []Block {
	body = strings.ReplaceAll(body, "\r\n", "\n")
	parts := strings.Split(body, "\n\n")

	var blocks []Block
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			blocks = append(blocks, Block{
				ID:   NewID(),
				Text: trimmed,
				Hash: HashBlock(trimmed),
			})
		}
	}
	return blocks
}

// MergeBlocks joins a slice of blocks back into a body string.
func MergeBlocks(blocks []Block) string {
	var parts []string
	for _, b := range blocks {
		parts = append(parts, b.Text)
	}
	return strings.Join(parts, "\n\n")
}

var tagRegexp = regexp.MustCompile(`(?:^|\s)#([a-zA-Z0-9_\-]+)`)

// ExtractTags extracts hashtags from a body string.
func ExtractTags(body string) []string {
	matches := tagRegexp.FindAllStringSubmatch(body, -1)
	tagMap := make(map[string]bool)
	var tags []string
	for _, match := range matches {
		tag := match[1]
		if !tagMap[tag] {
			tagMap[tag] = true
			tags = append(tags, tag)
		}
	}
	sort.Strings(tags)
	return tags
}

var linkRegexp = regexp.MustCompile(`\[\[([^\]]+)\]\]`)

// ExtractLinks extracts note links in the form [[Note Title]] from a body string.
func ExtractLinks(body string) []string {
	matches := linkRegexp.FindAllStringSubmatch(body, -1)
	linkMap := make(map[string]bool)
	var links []string
	for _, match := range matches {
		link := match[1]
		if !linkMap[link] {
			linkMap[link] = true
			links = append(links, link)
		}
	}
	sort.Strings(links)
	return links
}

// SortNotes sorts a slice of Notes in place.
func SortNotes(notes []*Note, ascending bool) {
	sort.Slice(notes, func(i, j int) bool {
		if notes[i].Pinned != notes[j].Pinned {
			return notes[i].Pinned // Pinned notes always come first
		}
		if ascending {
			return notes[i].CreatedAt.Before(notes[j].CreatedAt)
		}
		return notes[i].CreatedAt.After(notes[j].CreatedAt)
	})
}

// SortChapters sorts a slice of Chapters in place.
func SortChapters(chapters []*Chapter, ascending bool) {
	sort.Slice(chapters, func(i, j int) bool {
		if ascending {
			return chapters[i].CreatedAt.Before(chapters[j].CreatedAt)
		}
		return chapters[i].CreatedAt.After(chapters[j].CreatedAt)
	})
}

// SortBooks sorts a slice of Books in place.
func SortBooks(books []*Book, ascending bool) {
	sort.Slice(books, func(i, j int) bool {
		if ascending {
			return books[i].CreatedAt.Before(books[j].CreatedAt)
		}
		return books[i].CreatedAt.After(books[j].CreatedAt)
	})
}
