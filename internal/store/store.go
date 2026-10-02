package store

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/YaswanthKumarMallela01/kosha/internal/config"
	"github.com/YaswanthKumarMallela01/kosha/internal/model"
	"github.com/YaswanthKumarMallela01/kosha/internal/vault"
)

// Library represents the state of the books and chapters in the terminal vault.
type Library struct {
	DataDir   string
	MasterKey []byte
	Books     []*model.Book
	// Cache of loaded chapters: map[bookSlug]map[chapterID]*model.Chapter
	chapters map[string]map[string]*model.Chapter
	mu       sync.RWMutex
}

// NoteRef is a reference to a specific note in the library.
type NoteRef struct {
	BookSlug     string
	BookName     string
	ChapterID    string
	ChapterTitle string
	Note         *model.Note
}

// NewLibrary initializes a new Library instance.
func NewLibrary(dataDir string, masterKey []byte) *Library {
	return &Library{
		DataDir:   dataDir,
		MasterKey: masterKey,
		Books:     make([]*model.Book, 0),
		chapters:  make(map[string]map[string]*model.Chapter),
	}
}

// LoadBooks scans the books directory and populates l.Books.
func (l *Library) LoadBooks() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	booksDir := config.BooksDir(l.DataDir)
	if err := os.MkdirAll(booksDir, 0700); err != nil {
		return fmt.Errorf("failed to create books directory: %w", err)
	}

	entries, err := os.ReadDir(booksDir)
	if err != nil {
		return fmt.Errorf("failed to read books directory: %w", err)
	}

	var loadedBooks []*model.Book
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		slug := entry.Name()
		bookDir := config.BookDir(l.DataDir, slug)
		meta, err := vault.LoadBookMeta(bookDir, l.MasterKey)
		if err != nil {
			// Skip invalid/unreadable books
			continue
		}
		loadedBooks = append(loadedBooks, &model.Book{
			Slug:        slug,
			DisplayName: meta.DisplayName,
			CreatedAt:   meta.CreatedAt,
			Path:        bookDir,
		})
	}

	sort.Slice(loadedBooks, func(i, j int) bool {
		return loadedBooks[i].CreatedAt.After(loadedBooks[j].CreatedAt)
	})

	l.Books = loadedBooks
	return nil
}

// CreateBook creates a new book and adds it to the library.
func (l *Library) CreateBook(displayName string) (*model.Book, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	slug := model.Slugify(displayName)
	bookDir := config.BookDir(l.DataDir, slug)

	if err := os.MkdirAll(bookDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create book directory: %w", err)
	}

	meta := &model.BookMeta{
		DisplayName: displayName,
		CreatedAt:   time.Now(),
	}

	if err := vault.SaveBookMeta(bookDir, meta, l.MasterKey); err != nil {
		return nil, fmt.Errorf("failed to save book meta: %w", err)
	}

	newBook := &model.Book{
		Slug:        slug,
		DisplayName: displayName,
		CreatedAt:   meta.CreatedAt,
		Path:        bookDir,
	}

	l.Books = append(l.Books, newBook)
	return newBook, nil
}

// RenameBook updates a book's display name.
func (l *Library) RenameBook(slug string, newName string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	var bookToRename *model.Book
	for _, b := range l.Books {
		if b.Slug == slug {
			bookToRename = b
			break
		}
	}

	if bookToRename == nil {
		return fmt.Errorf("book not found: %s", slug)
	}

	bookDir := config.BookDir(l.DataDir, slug)
	meta := &model.BookMeta{
		DisplayName: newName,
		CreatedAt:   bookToRename.CreatedAt,
	}

	if err := vault.SaveBookMeta(bookDir, meta, l.MasterKey); err != nil {
		return fmt.Errorf("failed to save renamed book meta: %w", err)
	}

	bookToRename.DisplayName = newName
	return nil
}

// DeleteBook removes a book and all its contents.
func (l *Library) DeleteBook(slug string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	bookDir := config.BookDir(l.DataDir, slug)
	if err := os.RemoveAll(bookDir); err != nil {
		return fmt.Errorf("failed to delete book directory: %w", err)
	}

	for i, b := range l.Books {
		if b.Slug == slug {
			l.Books = append(l.Books[:i], l.Books[i+1:]...)
			break
		}
	}
	delete(l.chapters, slug)
	return nil
}

// LoadChapters lists and loads all chapters for a given book.
func (l *Library) LoadChapters(bookSlug string) ([]*model.Chapter, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	bookDir := config.BookDir(l.DataDir, bookSlug)
	chapterFiles, err := vault.ListChapterFiles(bookDir)
	if err != nil {
		return nil, fmt.Errorf("failed to list chapters: %w", err)
	}

	if l.chapters[bookSlug] == nil {
		l.chapters[bookSlug] = make(map[string]*model.Chapter)
	}

	var loadedChapters []*model.Chapter
	for _, filePath := range chapterFiles {
		chapter, err := vault.LoadChapter(filePath, l.MasterKey)
		if err != nil {
			continue // Skip unreadable chapters
		}
		l.chapters[bookSlug][chapter.ID] = chapter
		loadedChapters = append(loadedChapters, chapter)
	}

	sort.Slice(loadedChapters, func(i, j int) bool {
		return loadedChapters[i].CreatedAt.Before(loadedChapters[j].CreatedAt)
	})

	return loadedChapters, nil
}

// GetChapter returns a chapter from cache or loads it from disk.
func (l *Library) GetChapter(bookSlug, chapterID string) (*model.Chapter, error) {
	l.mu.RLock()
	bookCache, ok := l.chapters[bookSlug]
	if ok {
		if chapter, ok := bookCache[chapterID]; ok {
			l.mu.RUnlock()
			return chapter, nil
		}
	}
	l.mu.RUnlock()

	// Load from disk
	l.mu.Lock()
	defer l.mu.Unlock()
	bookDir := config.BookDir(l.DataDir, bookSlug)
	chapterPath := filepath.Join(bookDir, chapterID+".vault")
	chapter, err := vault.LoadChapter(chapterPath, l.MasterKey)
	if err != nil {
		return nil, fmt.Errorf("failed to load chapter: %w", err)
	}

	if l.chapters[bookSlug] == nil {
		l.chapters[bookSlug] = make(map[string]*model.Chapter)
	}
	l.chapters[bookSlug][chapterID] = chapter
	return chapter, nil
}

// CreateChapter creates a new chapter with the given title.
func (l *Library) CreateChapter(bookSlug, title string) (*model.Chapter, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	chapter := &model.Chapter{
		ID:        model.NewID(),
		Title:     title,
		Notes:     make([]*model.Note, 0),
		CreatedAt: time.Now(),
	}

	bookDir := config.BookDir(l.DataDir, bookSlug)
	if err := vault.SaveChapter(bookDir, chapter, l.MasterKey); err != nil {
		return nil, fmt.Errorf("failed to save new chapter: %w", err)
	}

	if l.chapters[bookSlug] == nil {
		l.chapters[bookSlug] = make(map[string]*model.Chapter)
	}
	l.chapters[bookSlug][chapter.ID] = chapter
	return chapter, nil
}

// SaveChapter saves modifications to a chapter.
func (l *Library) SaveChapter(bookSlug string, chapter *model.Chapter) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	bookDir := config.BookDir(l.DataDir, bookSlug)
	if err := vault.SaveChapter(bookDir, chapter, l.MasterKey); err != nil {
		return fmt.Errorf("failed to save chapter: %w", err)
	}

	if l.chapters[bookSlug] == nil {
		l.chapters[bookSlug] = make(map[string]*model.Chapter)
	}
	l.chapters[bookSlug][chapter.ID] = chapter
	return nil
}

// RenameChapter changes the title of an existing chapter.
func (l *Library) RenameChapter(bookSlug, chapterID, newTitle string) error {
	chapter, err := l.GetChapter(bookSlug, chapterID)
	if err != nil {
		return err
	}
	chapter.Title = newTitle
	return l.SaveChapter(bookSlug, chapter)
}

// DeleteChapter removes a chapter's vault file and clears it from cache.
func (l *Library) DeleteChapter(bookSlug, chapterID string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	bookDir := config.BookDir(l.DataDir, bookSlug)
	chapterFile := filepath.Join(bookDir, chapterID+".vault")
	if err := os.Remove(chapterFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete chapter file: %w", err)
	}

	if bookCache, ok := l.chapters[bookSlug]; ok {
		delete(bookCache, chapterID)
	}
	return nil
}

// AddNote appends a note to a chapter and saves it.
func (l *Library) AddNote(bookSlug, chapterID string, note *model.Note) error {
	chapter, err := l.GetChapter(bookSlug, chapterID)
	if err != nil {
		return err
	}
	chapter.Notes = append(chapter.Notes, note)
	return l.SaveChapter(bookSlug, chapter)
}

// UpdateNote modifies an existing note in a chapter.
func (l *Library) UpdateNote(bookSlug, chapterID string, note *model.Note) error {
	chapter, err := l.GetChapter(bookSlug, chapterID)
	if err != nil {
		return err
	}
	for i, n := range chapter.Notes {
		if n.ID == note.ID {
			chapter.Notes[i] = note
			return l.SaveChapter(bookSlug, chapter)
		}
	}
	return fmt.Errorf("note not found: %s", note.ID)
}

// DeleteNote removes a note from a chapter.
func (l *Library) DeleteNote(bookSlug, chapterID, noteID string) error {
	chapter, err := l.GetChapter(bookSlug, chapterID)
	if err != nil {
		return err
	}
	for i, n := range chapter.Notes {
		if n.ID == noteID {
			chapter.Notes = append(chapter.Notes[:i], chapter.Notes[i+1:]...)
			return l.SaveChapter(bookSlug, chapter)
		}
	}
	return fmt.Errorf("note not found: %s", noteID)
}

// EnsureInbox ensures the "inbox" book and "Quick Capture" chapter exist.
func (l *Library) EnsureInbox() error {
	inboxSlug := "inbox"

	// Check if inbox book exists
	var inboxBook *model.Book
	l.mu.RLock()
	for _, b := range l.Books {
		if b.Slug == inboxSlug {
			inboxBook = b
			break
		}
	}
	l.mu.RUnlock()

	if inboxBook == nil {
		_, err := l.CreateBook("Inbox")
		if err != nil {
			return fmt.Errorf("failed to create inbox book: %w", err)
		}
	}

	chapters, err := l.LoadChapters(inboxSlug)
	if err != nil {
		return fmt.Errorf("failed to load inbox chapters: %w", err)
	}

	var quickCapture *model.Chapter
	for _, c := range chapters {
		if c.Title == "Quick Capture" {
			quickCapture = c
			break
		}
	}

	if quickCapture == nil {
		_, err := l.CreateChapter(inboxSlug, "Quick Capture")
		if err != nil {
			return fmt.Errorf("failed to create quick capture chapter: %w", err)
		}
	}

	return nil
}

// QuickCapture creates a quick note in the Inbox -> Quick Capture.
func (l *Library) QuickCapture(text string) error {
	if err := l.EnsureInbox(); err != nil {
		return err
	}

	inboxSlug := "inbox"
	chapters, err := l.LoadChapters(inboxSlug)
	if err != nil {
		return err
	}

	var qcChapterID string
	for _, c := range chapters {
		if c.Title == "Quick Capture" {
			qcChapterID = c.ID
			break
		}
	}

	if qcChapterID == "" {
		return fmt.Errorf("quick capture chapter not found after ensuring")
	}

	// generate title
	lines := strings.Split(strings.TrimSpace(text), "\n")
	title := lines[0]
	if len(title) > 50 {
		title = title[:47] + "..."
	}

	note := &model.Note{
		ID:        model.NewID(),
		Title:     title,
		Body:      text,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	return l.AddNote(inboxSlug, qcChapterID, note)
}

// GetAllNotes returns all notes currently cached in the library.
func (l *Library) GetAllNotes() []NoteRef {
	l.mu.RLock()
	defer l.mu.RUnlock()

	var notes []NoteRef
	for bookSlug, bookCache := range l.chapters {
		var bookName string
		for _, b := range l.Books {
			if b.Slug == bookSlug {
				bookName = b.DisplayName
				break
			}
		}

		for _, chapter := range bookCache {
			for _, note := range chapter.Notes {
				notes = append(notes, NoteRef{
					BookSlug:     bookSlug,
					BookName:     bookName,
					ChapterID:    chapter.ID,
					ChapterTitle: chapter.Title,
					Note:         note,
				})
			}
		}
	}
	return notes
}

// FindNoteByTitle searches for a note by exact title match.
func (l *Library) FindNoteByTitle(title string, preferChapterID string, preferBookSlug string) (*NoteRef, error) {
	allNotes := l.GetAllNotes()

	var bestMatch *NoteRef
	bestScore := -1

	for i := range allNotes {
		ref := &allNotes[i]
		if ref.Note.Title == title {
			score := 0
			if ref.BookSlug == preferBookSlug {
				score++
				if ref.ChapterID == preferChapterID {
					score++
				}
			}
			if score > bestScore {
				bestMatch = ref
				bestScore = score
			}
		}
	}

	if bestMatch != nil {
		return bestMatch, nil
	}
	return nil, nil
}

// GetRandomOldNote returns a random note older than minAge.
func (l *Library) GetRandomOldNote(minAge time.Duration) *NoteRef {
	allNotes := l.GetAllNotes()
	if len(allNotes) == 0 {
		return nil
	}

	var candidates []NoteRef
	threshold := time.Now().Add(-minAge)

	for _, ref := range allNotes {
		if ref.Note.CreatedAt.Before(threshold) && !ref.Note.Pinned {
			candidates = append(candidates, ref)
		}
	}

	// Fallback if no old unpinned notes
	if len(candidates) == 0 {
		candidates = allNotes
	}

	src := rand.NewSource(time.Now().UnixNano())
	r := rand.New(src)
	idx := r.Intn(len(candidates))
	return &candidates[idx]
}

// CreateSnapshot takes a snapshot of a chapter.
func (l *Library) CreateSnapshot(bookSlug, chapterID string) error {
	chapter, err := l.GetChapter(bookSlug, chapterID)
	if err != nil {
		return err
	}

	bookDir := config.BookDir(l.DataDir, bookSlug)
	payload, err := json.Marshal(chapter)
	if err != nil {
		return fmt.Errorf("failed to marshal chapter for snapshot: %w", err)
	}

	return vault.SaveSnapshot(bookDir, chapterID, payload, l.MasterKey)
}

// Lock clears the master key and all cached data from memory.
func (l *Library) Lock() {
	l.mu.Lock()
	defer l.mu.Unlock()

	for i := range l.MasterKey {
		l.MasterKey[i] = 0
	}
	l.MasterKey = nil
	l.Books = make([]*model.Book, 0)
	l.chapters = make(map[string]map[string]*model.Chapter)
}
