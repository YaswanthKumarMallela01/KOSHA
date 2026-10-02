package model

import (
	"testing"
	"time"

	"github.com/oklog/ulid/v2"
)

func TestNewID(t *testing.T) {
	id1 := NewID()
	id2 := NewID()

	if id1 == "" || id2 == "" {
		t.Error("Expected non-empty ULID")
	}
	if id1 == id2 {
		t.Error("Expected unique ULIDs")
	}
	
	_, err := ulid.Parse(id1)
	if err != nil {
		t.Errorf("Expected valid ULID, got error: %v", err)
	}
}

func TestSlugify(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Hello World", "hello-world"},
		{"My Book_Name", "my-book-name"},
		{"Space   Between", "space-between"},
		{"!@#Special$%^Chars", "specialchars"},
		{"-Leading-And-Trailing-", "leading-and-trailing"},
	}

	for _, tt := range tests {
		actual := Slugify(tt.input)
		if actual != tt.expected {
			t.Errorf("Slugify(%q): expected %q, got %q", tt.input, tt.expected, actual)
		}
	}
}

func TestSplitAndMergeBlocks(t *testing.T) {
	body := "Paragraph 1.\n\nParagraph 2 is here.\n\nParagraph 3."
	blocks := SplitBlocks(body)

	if len(blocks) != 3 {
		t.Fatalf("Expected 3 blocks, got %d", len(blocks))
	}

	if blocks[0].Text != "Paragraph 1." {
		t.Errorf("Expected block 1 text 'Paragraph 1.', got %q", blocks[0].Text)
	}
	
	if blocks[0].ID == "" || blocks[0].Hash == "" {
		t.Error("Expected blocks to have ID and Hash populated")
	}

	merged := MergeBlocks(blocks)
	if merged != body {
		t.Errorf("MergeBlocks round trip failed. Expected:\n%s\nGot:\n%s", body, merged)
	}
}

func TestHashBlock(t *testing.T) {
	text := "Sample text"
	hash1 := HashBlock(text)
	hash2 := HashBlock(text)

	if hash1 != hash2 {
		t.Error("Expected identical hashes for identical text")
	}
	
	hash3 := HashBlock("Different text")
	if hash1 == hash3 {
		t.Error("Expected different hashes for different text")
	}
}

func TestExtractTags(t *testing.T) {
	body := "This is a #tag inside some text. Here is another #tag, and a #new_tag-123! ## Heading should be ignored. Email to some#body should be ignored."
	tags := ExtractTags(body)

	if len(tags) != 2 {
		t.Fatalf("Expected 2 tags, got %v", tags)
	}
	
	// Expect sorted output
	if tags[0] != "new_tag-123" || tags[1] != "tag" {
		t.Errorf("Unexpected tags: %v", tags)
	}
}

func TestExtractLinks(t *testing.T) {
	body := "Check out [[Note A]] and [[Note B]]. Also see [[Note A]] again."
	links := ExtractLinks(body)

	if len(links) != 2 {
		t.Fatalf("Expected 2 links, got %v", links)
	}

	// Expect sorted output
	if links[0] != "Note A" || links[1] != "Note B" {
		t.Errorf("Unexpected links: %v", links)
	}
}

func TestSortNotes(t *testing.T) {
	t1 := time.Now().Add(-2 * time.Hour)
	t2 := time.Now().Add(-1 * time.Hour)
	t3 := time.Now()

	notes := []*Note{
		{ID: "3", CreatedAt: t3, Pinned: false},
		{ID: "2", CreatedAt: t2, Pinned: true},
		{ID: "1", CreatedAt: t1, Pinned: false},
	}

	// Ascending
	SortNotes(notes, true)
	if notes[0].ID != "2" || notes[1].ID != "1" || notes[2].ID != "3" {
		t.Errorf("SortNotes (ascending) failed. Order: %s, %s, %s", notes[0].ID, notes[1].ID, notes[2].ID)
	}

	// Descending
	SortNotes(notes, false)
	if notes[0].ID != "2" || notes[1].ID != "3" || notes[2].ID != "1" {
		t.Errorf("SortNotes (descending) failed. Order: %s, %s, %s", notes[0].ID, notes[1].ID, notes[2].ID)
	}
}

func TestBlockHashingAndDirtyDetection(t *testing.T) {
	// Simulating chapter payload and detecting dirty blocks
	processedHashes := map[string]bool{
		HashBlock("Old text"): true,
	}

	newText := "New text"
	block := Block{
		ID:   NewID(),
		Text: newText,
		Hash: HashBlock(newText),
	}

	if processedHashes[block.Hash] {
		t.Error("Expected block to not be processed")
	}

	processedHashes[block.Hash] = true
	if !processedHashes[block.Hash] {
		t.Error("Expected block to be processed after update")
	}
}
