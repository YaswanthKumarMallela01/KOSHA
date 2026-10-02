package search

import (
	"sort"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/sahilm/fuzzy"
)

// SearchEntry represents a searchable item, usually a note.
type SearchEntry struct {
	BookSlug     string
	BookName     string
	ChapterID    string
	ChapterTitle string
	NoteID       string
	NoteTitle    string
	Body         string // plain text (stripped of markup)
	RawBody      string // raw note body with markup
	Tags         []string
	Breadcrumb   string // "Book › Chapter › Note"
}

// Index holds the search index for entries.
type Index struct {
	entries []SearchEntry
	tagMap  map[string][]int // tag -> indices into entries
	mu      sync.RWMutex
}

// NewIndex creates and returns a new empty search Index.
func NewIndex() *Index {
	return &Index{
		entries: make([]SearchEntry, 0),
		tagMap:  make(map[string][]int),
	}
}

// Build initializes the index with the given entries and builds the tag map.
func (idx *Index) Build(entries []SearchEntry) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	idx.entries = make([]SearchEntry, len(entries))
	copy(idx.entries, entries)
	idx.rebuildTagMap()
}

// AddOrUpdate adds a new entry or updates an existing entry by NoteID.
func (idx *Index) AddOrUpdate(entry SearchEntry) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	found := false
	for i, e := range idx.entries {
		if e.NoteID == entry.NoteID {
			idx.entries[i] = entry
			found = true
			break
		}
	}
	if !found {
		idx.entries = append(idx.entries, entry)
	}

	idx.rebuildTagMap()
}

// Remove removes an entry by NoteID.
func (idx *Index) Remove(noteID string) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	for i, e := range idx.entries {
		if e.NoteID == noteID {
			idx.entries = append(idx.entries[:i], idx.entries[i+1:]...)
			break
		}
	}
	idx.rebuildTagMap()
}

// rebuildTagMap rebuilds the internal tag map. Must be called with a lock held.
func (idx *Index) rebuildTagMap() {
	idx.tagMap = make(map[string][]int)
	for i, entry := range idx.entries {
		for _, tag := range entry.Tags {
			idx.tagMap[tag] = append(idx.tagMap[tag], i)
		}
	}
}

// SearchResult represents a single search match with its score and context.
type SearchResult struct {
	Entry        SearchEntry
	Score        int
	MatchedChars []int  // positions of matched characters in the matched string
	MatchField   string // "title", "tag", "body"
	Snippet      string // short excerpt around the match
}

// stringSlice is a helper type to implement fuzzy.Source interface.
type stringSlice []string

func (s stringSlice) Len() int { return len(s) }
func (s stringSlice) String(i int) string { return s[i] }

// Search performs a fuzzy search across titles, tags, and body text.
func (idx *Index) Search(query string, maxResults int) []SearchResult {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	if query == "" {
		return nil
	}

	var titles, tags, bodies stringSlice
	for _, entry := range idx.entries {
		titles = append(titles, entry.NoteTitle)
		tags = append(tags, strings.Join(entry.Tags, " "))
		bodies = append(bodies, entry.Body)
	}

	titleMatches := fuzzy.FindFrom(query, titles)
	tagMatches := fuzzy.FindFrom(query, tags)
	bodyMatches := fuzzy.FindFrom(query, bodies)

	resultsMap := make(map[string]SearchResult)

	// Process title matches (bonus: +1000)
	for _, m := range titleMatches {
		entry := idx.entries[m.Index]
		resultsMap[entry.NoteID] = SearchResult{
			Entry:        entry,
			Score:        m.Score + 1000,
			MatchedChars: m.MatchedIndexes,
			MatchField:   "title",
			Snippet:      entry.NoteTitle,
		}
	}

	// Process tag matches (bonus: +500)
	for _, m := range tagMatches {
		entry := idx.entries[m.Index]
		score := m.Score + 500
		if existing, ok := resultsMap[entry.NoteID]; !ok || score > existing.Score {
			resultsMap[entry.NoteID] = SearchResult{
				Entry:        entry,
				Score:        score,
				MatchedChars: m.MatchedIndexes,
				MatchField:   "tag",
				Snippet:      strings.Join(entry.Tags, ", "),
			}
		}
	}

	// Process body matches (bonus: +0)
	for _, m := range bodyMatches {
		entry := idx.entries[m.Index]
		score := m.Score
		if existing, ok := resultsMap[entry.NoteID]; !ok || score > existing.Score {
			matchPos := 0
			if len(m.MatchedIndexes) > 0 {
				matchPos = m.MatchedIndexes[0]
			}
			resultsMap[entry.NoteID] = SearchResult{
				Entry:        entry,
				Score:        score,
				MatchedChars: m.MatchedIndexes,
				MatchField:   "body",
				Snippet:      GenerateSnippet(entry.Body, matchPos, 80),
			}
		}
	}

	var results []SearchResult
	for _, r := range resultsMap {
		results = append(results, r)
	}

	// Sort by score descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	if maxResults > 0 && len(results) > maxResults {
		results = results[:maxResults]
	}

	return results
}

// TagInfo contains information about a tag and its usage frequency.
type TagInfo struct {
	Name  string
	Count int
}

// GetAllTags returns a list of all unique tags with their counts, sorted alphabetically.
func (idx *Index) GetAllTags() []TagInfo {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	var tags []TagInfo
	for name, indices := range idx.tagMap {
		tags = append(tags, TagInfo{
			Name:  name,
			Count: len(indices),
		})
	}

	sort.Slice(tags, func(i, j int) bool {
		return tags[i].Name < tags[j].Name
	})

	return tags
}

// GetNotesByTag returns all entries that have the given tag.
func (idx *Index) GetNotesByTag(tag string) []SearchEntry {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	indices, ok := idx.tagMap[tag]
	if !ok {
		return nil
	}

	var entries []SearchEntry
	for _, i := range indices {
		entries = append(entries, idx.entries[i])
	}
	return entries
}

// FindBacklinks returns all entries whose Body or RawBody contains a link to noteTitle.
func (idx *Index) FindBacklinks(noteTitle string) []SearchEntry {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	targetLink := "[[" + strings.ToLower(noteTitle) + "]]"
	var entries []SearchEntry

	for _, entry := range idx.entries {
		if strings.Contains(strings.ToLower(entry.RawBody), targetLink) ||
			strings.Contains(strings.ToLower(entry.Body), targetLink) {
			entries = append(entries, entry)
		}
	}

	return entries
}

// GenerateSnippet extracts a snippet of snippetLen chars centered around matchPos.
func GenerateSnippet(text string, matchPos int, snippetLen int) string {
	if text == "" {
		return ""
	}

	// Calculate rune index for match position (which is a byte index)
	var runePos int
	if matchPos > 0 && matchPos < len(text) {
		runePos = utf8.RuneCountInString(text[:matchPos])
	} else if matchPos >= len(text) {
		runePos = utf8.RuneCountInString(text)
	}

	runes := []rune(text)
	if len(runes) <= snippetLen {
		return text
	}

	start := runePos - snippetLen/2
	if start < 0 {
		start = 0
	}
	end := start + snippetLen
	if end > len(runes) {
		end = len(runes)
		start = end - snippetLen
		if start < 0 {
			start = 0
		}
	}

	snippet := string(runes[start:end])
	if start > 0 {
		snippet = "..." + snippet
	}
	if end < len(runes) {
		snippet = snippet + "..."
	}

	return snippet
}
