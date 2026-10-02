package search

import (
	"testing"
)

func TestSearchRankingAndTags(t *testing.T) {
	idx := NewIndex()

	entries := []SearchEntry{
		{
			BookSlug:     "work",
			BookName:     "Work",
			ChapterID:    "01HQ",
			ChapterTitle: "Projects",
			NoteID:       "note-1",
			NoteTitle:    "Kosha Architecture",
			Body:         "Encrypted vault using argon2 and chacha20poly1305. See also [[Meeting Notes]].",
			Tags:         []string{"security", "crypto", "golang"},
			Breadcrumb:   "Work › Projects › Kosha Architecture",
		},
		{
			BookSlug:     "work",
			BookName:     "Work",
			ChapterID:    "01HQ",
			ChapterTitle: "Projects",
			NoteID:       "note-2",
			NoteTitle:    "Meeting Notes",
			Body:         "Discussed release roadmap and testing strategy.",
			Tags:         []string{"meeting", "planning"},
			Breadcrumb:   "Work › Projects › Meeting Notes",
		},
		{
			BookSlug:     "personal",
			BookName:     "Personal",
			ChapterID:    "01HP",
			ChapterTitle: "Journal",
			NoteID:       "note-3",
			NoteTitle:    "Daily Reflection",
			Body:         "Worked on kosha crypto tests and verified zeroing.",
			Tags:         []string{"journal", "crypto"},
			Breadcrumb:   "Personal › Journal › Daily Reflection",
		},
	}

	idx.Build(entries)

	// 1. Title match ranking: "Kosha" should rank note-1 higher than note-3 (note-1 has it in title, note-3 in body)
	results := idx.Search("Kosha", 10)
	if len(results) < 2 {
		t.Fatalf("expected at least 2 results for 'Kosha', got %d", len(results))
	}
	if results[0].Entry.NoteID != "note-1" {
		t.Errorf("expected note-1 first for title match, got %s", results[0].Entry.NoteID)
	}

	// 2. Tag query
	cryptoNotes := idx.GetNotesByTag("crypto")
	if len(cryptoNotes) != 2 {
		t.Errorf("expected 2 notes with tag 'crypto', got %d", len(cryptoNotes))
	}

	// 3. All tags
	allTags := idx.GetAllTags()
	if len(allTags) != 5 { // security, crypto, golang, meeting, planning, journal
		// Let's check counts
		foundCrypto := false
		for _, tag := range allTags {
			if tag.Name == "crypto" {
				foundCrypto = true
				if tag.Count != 2 {
					t.Errorf("expected crypto count 2, got %d", tag.Count)
				}
			}
		}
		if !foundCrypto {
			t.Errorf("tag 'crypto' not found in GetAllTags")
		}
	}

	// 4. Backlinks: note-2 is linked from note-1 via [[Meeting Notes]]
	backlinks := idx.FindBacklinks("Meeting Notes")
	if len(backlinks) != 1 {
		t.Fatalf("expected 1 backlink for 'Meeting Notes', got %d", len(backlinks))
	}
	if backlinks[0].NoteID != "note-1" {
		t.Errorf("expected backlink from note-1, got %s", backlinks[0].NoteID)
	}
}

func TestGenerateSnippet(t *testing.T) {
	text := "The quick brown fox jumps over the lazy dog in the beautiful forest during autumn."
	matchPos := 20
	snippet := GenerateSnippet(text, matchPos, 30)

	if len(snippet) == 0 {
		t.Fatal("empty snippet generated")
	}
	if snippet == text {
		t.Errorf("expected snippet to be truncated, got %s", snippet)
	}
}
