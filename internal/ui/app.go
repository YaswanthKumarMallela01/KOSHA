package ui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/YaswanthKumarMallela01/kosha/internal/ai"
	"github.com/YaswanthKumarMallela01/kosha/internal/config"
	"github.com/YaswanthKumarMallela01/kosha/internal/crypto"
	"github.com/YaswanthKumarMallela01/kosha/internal/editor"
	"github.com/YaswanthKumarMallela01/kosha/internal/export"
	"github.com/YaswanthKumarMallela01/kosha/internal/markup"
	"github.com/YaswanthKumarMallela01/kosha/internal/model"
	"github.com/YaswanthKumarMallela01/kosha/internal/search"
	"github.com/YaswanthKumarMallela01/kosha/internal/store"
)

type Screen int

const (
	ScreenLoading Screen = iota
	ScreenPassphrase
	ScreenLibrary
	ScreenBook
	ScreenChapter
	ScreenNoteView
	ScreenNoteEdit
	ScreenSearch
	ScreenDiffReview
	ScreenSnapshots
	ScreenHelp
	ScreenTagFilter
	ScreenInit
	ScreenConfirmDelete
	ScreenRename
	ScreenNewItem
	ScreenLinkPicker
)

type aiDiffResultMsg struct {
	diffs []ai.DiffBlock
}

type aiErrorMsg struct {
	err error
}

type App struct {
	screen         Screen
	prevScreens    []Screen
	library        *store.Library
	config         *config.Config
	searchIndex    *search.Index
	width, height  int
	currentBook    *model.Book
	currentChapter *model.Chapter
	currentNote    *model.Note

	passphrase    PassphraseModel
	libraryList   LibraryModel
	bookList      BookModel
	chapterList   ChapterModel
	noteView      NoteViewModel
	editorModel   editor.Model
	searchModel   SearchModel
	diffModel     DiffReviewModel
	snapshotModel SnapshotModel
	tagModel      TagFilterModel
	inputOverlay  InputOverlayModel
	confirmDialog ConfirmDialogModel

	statusMsg     string
	statusTimer   int
	err           error
	lastActivity  time.Time
	locked        bool
	lockMinutes   int
	ascending     bool
	resurfaceNote *store.NoteRef
	aiClient      *ai.Client
	aiProcessing  bool
	showHelp        bool
	theme           *markup.Theme
	loadingStep     int
	maxLoadingSteps int
}

type loadingTickMsg struct{}

func loadingTick() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return loadingTickMsg{}
	})
}

func NewApp(cfg *config.Config, lib *store.Library, idx *search.Index, aiCli *ai.Client) *App {
	a := &App{
		screen:          ScreenLoading,
		loadingStep:     0,
		maxLoadingSteps: 30, // 30 * 100ms = 3.0 seconds
		library:         lib,
		config:          cfg,
		searchIndex:     idx,
		aiClient:        aiCli,
		lastActivity:    time.Now(),
		lockMinutes:     cfg.LockMinutes,
		theme:           markup.NewDefaultTheme(),
	}

	a.passphrase = NewPassphraseModel()
	a.passphrase.app = a
	a.libraryList = NewLibraryModel(a)
	a.bookList = NewBookModel(a)
	a.chapterList = NewChapterModel(a)
	a.noteView = NewNoteViewModel(a)
	a.searchModel = NewSearchModel(a)
	a.diffModel = NewDiffReviewModel(a)
	a.snapshotModel = NewSnapshotModel(a)
	a.tagModel = NewTagFilterModel(a)
	a.inputOverlay = NewInputOverlayModel()
	a.confirmDialog = NewConfirmDialogModel()

	// Initial data population
	a.libraryList.Refresh()
	a.resurfaceNote = a.library.GetRandomOldNote(7 * 24 * time.Hour)
	a.buildSearchIndex()

	return a
}

func (a *App) Init() tea.Cmd {
	return tea.Batch(
		textinput.Blink,
		loadingTick(),
		tea.Tick(time.Minute, func(t time.Time) tea.Msg {
			return tickMsg(t)
		}),
	)
}

type tickMsg time.Time

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		a.lastActivity = time.Now()
		a.statusMsg = "" // clear status on key press

		if a.screen == ScreenLoading {
			a.screen = ScreenLibrary
			a.libraryList.Refresh()
			return a, nil
		}

		if msg.String() == "ctrl+c" {
			// Save active note if in editor before exit
			if a.screen == ScreenNoteEdit && a.currentNote != nil && a.currentChapter != nil {
				a.currentNote.Body = a.editorModel.Content()
				a.currentNote.UpdatedAt = time.Now()
				_ = a.library.SaveChapter(a.currentBook.Slug, a.currentChapter)
			}
			return a, tea.Quit
		}

		if msg.String() == "ctrl+l" && !a.locked {
			a.lock()
			return a, tea.ClearScreen
		}

		if msg.String() == "?" && !a.locked && a.screen != ScreenNoteEdit {
			a.showHelp = !a.showHelp
			return a, nil
		}

		if a.showHelp {
			if msg.String() == "esc" || msg.String() == "?" || msg.String() == "q" {
				a.showHelp = false
				return a, nil
			}
			return a, nil
		}

		if (msg.String() == "ctrl+n" || msg.String() == "ctrl+N") && a.screen != ScreenNoteEdit && a.screen != ScreenNewItem && a.screen != ScreenRename {
			if a.screen == ScreenLibrary {
				a.inputOverlay.Open("Create New Book in D:\\books (e.g. Operating Systems):", "", "new_book", "")
				a.pushScreen(ScreenNewItem)
				return a, nil
			} else if a.screen == ScreenBook && a.currentBook != nil {
				a.inputOverlay.Open("Create New Vault Section in "+a.currentBook.DisplayName+" (e.g. Memory Management):", "", "new_vault", "")
				a.pushScreen(ScreenNewItem)
				return a, nil
			}
		}

	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.libraryList.SetSize(a.width, a.height)
		a.bookList.SetSize(a.width, a.height)
		a.chapterList.SetSize(a.width, a.height)
		a.noteView.SetSize(a.width, a.height)
		a.searchModel.SetSize(a.width, a.height)
		a.snapshotModel.SetSize(a.width, a.height)
		a.tagModel.SetSize(a.width, a.height)
		a.editorModel.SetSize(a.width-8, a.height-6)

	case tickMsg:
		if a.lockMinutes > 0 && !a.locked && time.Since(a.lastActivity) > time.Duration(a.lockMinutes)*time.Minute {
			a.lock()
			return a, tea.ClearScreen
		}
		cmds = append(cmds, tea.Tick(time.Minute, func(t time.Time) tea.Msg {
			return tickMsg(t)
		}))

	case aiDiffResultMsg:
		a.aiProcessing = false
		a.statusMsg = "AI refinement complete. Reviewing changes..."
		a.diffModel.SetBlocks(msg.diffs)
		a.pushScreen(ScreenDiffReview)
		return a, nil

	case loadingTickMsg:
		if a.screen == ScreenLoading {
			a.loadingStep++
			if a.loadingStep >= a.maxLoadingSteps {
				a.screen = ScreenLibrary
				a.libraryList.Refresh()
				return a, nil
			}
			return a, loadingTick()
		}

	case aiErrorMsg:
		a.aiProcessing = false
		errStr := msg.err.Error()
		if strings.Contains(errStr, "429") {
			a.statusMsg = "Gemini Quota Exceeded (429): API limit reached for your region/key. Verify in Google AI Studio."
		} else {
			a.statusMsg = fmt.Sprintf("AI error: %v", msg.err)
		}
		return a, nil
	}

	if a.locked {
		var pCmd tea.Cmd
		a.passphrase, pCmd = a.passphrase.Update(msg)
		return a, pCmd
	}

	// Route based on screen
	switch a.screen {
	case ScreenLibrary:
		a.libraryList, cmd = a.libraryList.Update(msg)
	case ScreenBook:
		a.bookList, cmd = a.bookList.Update(msg)
	case ScreenChapter:
		a.chapterList, cmd = a.chapterList.Update(msg)
	case ScreenNoteView:
		a.noteView, cmd = a.noteView.Update(msg)
	case ScreenNoteEdit:
		cmd = a.updateEditor(msg)
	case ScreenSearch:
		a.searchModel, cmd = a.searchModel.Update(msg)
	case ScreenDiffReview:
		a.diffModel, cmd = a.diffModel.Update(msg)
	case ScreenSnapshots:
		a.snapshotModel, cmd = a.snapshotModel.Update(msg)
	case ScreenTagFilter:
		a.tagModel, cmd = a.tagModel.Update(msg)
	case ScreenNewItem:
		cmd = a.updateNewItem(msg)
	case ScreenRename:
		cmd = a.updateRename(msg)
	case ScreenConfirmDelete:
		cmd = a.updateConfirmDelete(msg)
	}

	cmds = append(cmds, cmd)
	return a, tea.Batch(cmds...)
}

func (a *App) updateEditor(msg tea.Msg) tea.Cmd {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "esc":
			// Save and return to Book (vaults shelf)
			if a.currentNote != nil && a.currentChapter != nil && a.currentBook != nil {
				a.currentNote.Body = a.editorModel.Content()
				a.currentNote.UpdatedAt = time.Now()
				_ = a.library.SaveChapter(a.currentBook.Slug, a.currentChapter)
				a.buildSearchIndex()
				a.bookList.Refresh()
			}
			a.currentNote = nil
			a.screen = ScreenBook
			return nil
		case "ctrl+r":
			// Toggle to peaceful Read Mode
			if a.currentNote != nil && a.currentChapter != nil && a.currentBook != nil {
				a.currentNote.Body = a.editorModel.Content()
				a.currentNote.UpdatedAt = time.Now()
				_ = a.library.SaveChapter(a.currentBook.Slug, a.currentChapter)
				a.buildSearchIndex()
				a.noteView.Refresh()
			}
			a.screen = ScreenNoteView
			return nil
		case "ctrl+g":
			return a.triggerAIRefine()
		case "ctrl+s":
			if a.currentBook != nil && a.currentChapter != nil {
				_ = a.library.CreateSnapshot(a.currentBook.Slug, a.currentChapter.ID)
				a.statusMsg = "Chapter snapshot saved"
			}
			return nil
		case "ctrl+i":
			// Image insertion - open input overlay to get image path
			a.inputOverlay.Open("Insert image — enter file path:", "", "insert_image", "")
			a.pushScreen(ScreenNewItem)
			return nil
		case "ctrl+p":
			// Export PDF
			if a.currentNote != nil {
				a.currentNote.Body = a.editorModel.Content()
				a.currentNote.UpdatedAt = time.Now()
				_ = a.library.SaveChapter(a.currentBook.Slug, a.currentChapter)
				a.confirmDialog.Open("Enter passphrase to export as PDF:", "export_pdf", "")
				a.pushScreen(ScreenConfirmDelete)
			}
			return nil
		case "ctrl+w":
			// Export DOCX
			if a.currentNote != nil {
				a.currentNote.Body = a.editorModel.Content()
				a.currentNote.UpdatedAt = time.Now()
				_ = a.library.SaveChapter(a.currentBook.Slug, a.currentChapter)
				a.confirmDialog.Open("Enter passphrase to export as Word (.docx):", "export_docx", "")
				a.pushScreen(ScreenConfirmDelete)
			}
			return nil
		}
	}

	var edModel tea.Model
	var edCmd tea.Cmd
	edModel, edCmd = a.editorModel.Update(msg)
	if em, ok := edModel.(editor.Model); ok {
		a.editorModel = em
	}
	return edCmd
}

func (a *App) updateNewItem(msg tea.Msg) tea.Cmd {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "enter":
			val := strings.TrimSpace(a.inputOverlay.Input.Value())
			if val == "" {
				a.inputOverlay.Close()
				a.popScreen()
				return nil
			}
			switch a.inputOverlay.Action {
			case "new_book":
				b, err := a.library.CreateBook(val)
				if err == nil && b != nil {
					a.currentBook = b
					a.libraryList.Refresh()
					a.bookList.Refresh()
					a.screen = ScreenBook
					a.statusMsg = "Book created: " + b.DisplayName
				} else if err != nil {
					a.statusMsg = "Error creating book: " + err.Error()
				}
				a.inputOverlay.Close()
				return nil
			case "new_vault", "new_chapter":
				if a.currentBook != nil {
					ch, err := a.library.CreateChapter(a.currentBook.Slug, val)
					if err == nil && ch != nil {
						note := &model.Note{
							ID:        model.NewID(),
							Title:     val,
							Body:      "",
							CreatedAt: time.Now(),
							UpdatedAt: time.Now(),
						}
						_ = a.library.AddNote(a.currentBook.Slug, ch.ID, note)
						a.currentChapter = ch
						a.currentNote = note
						a.bookList.Refresh()
						a.inputOverlay.Close()
						a.openEditor(note)
						a.statusMsg = "Vault created: " + ch.Title + " — start writing notes!"
						return nil
					} else if err != nil {
						a.statusMsg = "Error creating vault: " + err.Error()
					}
				}
				a.inputOverlay.Close()
				a.popScreen()
				return nil
			case "insert_image":
				// val is the file path; insert with center alignment and auto width
				a.insertImage(val, filepath.Base(val), "center", 0)
				a.inputOverlay.Close()
				a.popScreen()
				return nil
			}
			a.inputOverlay.Close()
			a.popScreen()
			return nil
		case "esc":
			a.inputOverlay.Close()
			a.popScreen()
			return nil
		}
	}

	var cmd tea.Cmd
	a.inputOverlay, cmd = a.inputOverlay.Update(msg)
	return cmd
}

func (a *App) updateRename(msg tea.Msg) tea.Cmd {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "enter":
			val := strings.TrimSpace(a.inputOverlay.Input.Value())
			if val != "" {
				switch a.inputOverlay.Action {
				case "rename_book":
					_ = a.library.RenameBook(a.inputOverlay.TargetID, val)
					a.libraryList.Refresh()
					a.statusMsg = "Book renamed"
				case "rename_chapter":
					_ = a.library.RenameChapter(a.currentBook.Slug, a.inputOverlay.TargetID, val)
					a.bookList.Refresh()
					a.statusMsg = "Chapter renamed"
				case "rename_note":
					if a.currentNote != nil {
						a.currentNote.Title = val
						a.currentNote.UpdatedAt = time.Now()
						_ = a.library.UpdateNote(a.currentBook.Slug, a.currentChapter.ID, a.currentNote)
						a.chapterList.Refresh()
						a.statusMsg = "Note renamed"
					}
				}
			}
			a.inputOverlay.Close()
			a.popScreen()
			return nil
		case "esc":
			a.inputOverlay.Close()
			a.popScreen()
			return nil
		}
	}

	var cmd tea.Cmd
	a.inputOverlay, cmd = a.inputOverlay.Update(msg)
	return cmd
}

func (a *App) updateConfirmDelete(msg tea.Msg) tea.Cmd {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "enter":
			pass := strings.TrimSpace(a.confirmDialog.PasswordInput.Value())
			if pass == "" {
				a.statusMsg = "Passphrase required. Deletion aborted."
				a.confirmDialog.Close()
				a.popScreen()
				return nil
			}

			// Validate passphrase against master key check
			derived := crypto.DeriveMasterKey([]byte(pass), []byte(a.config.MasterSalt), a.config.ArgonParams)
			crypto.ZeroBytes([]byte(pass))
			if !crypto.ValidateKeyCheck([]byte(a.config.KeyCheck), derived) {
				crypto.ZeroBytes(derived)
				a.statusMsg = "Incorrect passphrase! Deletion aborted."
				a.confirmDialog.Close()
				a.popScreen()
				return nil
			}
			crypto.ZeroBytes(derived)

			switch a.confirmDialog.Action {
			case "delete_book":
				_ = a.library.DeleteBook(a.confirmDialog.TargetID)
				a.libraryList.Refresh()
				a.statusMsg = "Book deleted successfully"
			case "delete_chapter":
				_ = a.library.DeleteChapter(a.currentBook.Slug, a.confirmDialog.TargetID)
				a.bookList.Refresh()
				a.statusMsg = "Vault chapter deleted successfully"
			case "delete_note":
				_ = a.library.DeleteNote(a.currentBook.Slug, a.currentChapter.ID, a.confirmDialog.TargetID)
				a.chapterList.Refresh()
				a.statusMsg = "Note deleted successfully"
			case "export_pdf":
				a.exportCurrentNotePDF()
			case "export_docx":
				a.exportCurrentNoteDOCX()
			}
			a.confirmDialog.Close()
			a.popScreen()
			return nil
		case "esc":
			a.confirmDialog.Close()
			a.popScreen()
			return nil
		}
	}

	var cmd tea.Cmd
	a.confirmDialog, cmd = a.confirmDialog.Update(msg)
	return cmd
}

func (a *App) openEditor(note *model.Note) {
	a.currentNote = note
	a.editorModel = editor.New(note.Body, a.theme)
	a.editorModel.SetSize(a.width-8, a.height-6)
	a.editorModel.Focus()

	// Set autosave callback
	bookSlug := a.currentBook.Slug
	noteID := note.ID
	a.editorModel.OnSave = func(content string) {
		if a.currentChapter != nil {
			for _, n := range a.currentChapter.Notes {
				if n.ID == noteID {
					n.Body = content
					n.UpdatedAt = time.Now()
					_ = a.library.SaveChapter(bookSlug, a.currentChapter)
					a.buildSearchIndex()
					break
				}
			}
		}
	}

	a.pushScreen(ScreenNoteEdit)
}

func (a *App) jumpToNote(ref *store.NoteRef) {
	if ref == nil {
		return
	}
	a.jumpToNoteID(ref.BookSlug, ref.ChapterID, ref.Note.ID)
}

func (a *App) jumpToNoteID(bookSlug, chapterID, noteID string) {
	for _, b := range a.library.Books {
		if b.Slug == bookSlug {
			a.currentBook = b
			break
		}
	}
	if a.currentBook == nil {
		return
	}

	ch, err := a.library.GetChapter(bookSlug, chapterID)
	if err != nil || ch == nil {
		return
	}
	a.currentChapter = ch

	for _, n := range ch.Notes {
		if n.ID == noteID {
			a.currentNote = n
			a.noteView.Refresh()
			a.pushScreen(ScreenNoteView)
			return
		}
	}
}

func (a *App) exportBook(slug string) {
	chapters, err := a.library.LoadChapters(slug)
	if err != nil {
		a.statusMsg = fmt.Sprintf("Export error: %v", err)
		return
	}

	outDir := filepath.Join(".", slug+"_export")
	_ = os.MkdirAll(outDir, 0700)
	count := 0

	for _, ch := range chapters {
		for _, note := range ch.Notes {
			fn := filepath.Join(outDir, fmt.Sprintf("%s_%s.md", ch.Title, note.Title))
			md := fmt.Sprintf("# %s\n\n%s\n", note.Title, markup.RenderToMarkdown(note.Body))
			_ = os.WriteFile(fn, []byte(md), 0600)
			count++
		}
	}
	a.statusMsg = fmt.Sprintf("Exported %d notes to %s", count, outDir)
}

func (a *App) exportChapter(slug, chapterTitle string) {
	outDir := filepath.Join(".", slug+"_export")
	_ = os.MkdirAll(outDir, 0700)
	count := 0

	if a.currentChapter != nil {
		for _, note := range a.currentChapter.Notes {
			fn := filepath.Join(outDir, fmt.Sprintf("%s_%s.md", chapterTitle, note.Title))
			md := fmt.Sprintf("# %s\n\n%s\n", note.Title, markup.RenderToMarkdown(note.Body))
			_ = os.WriteFile(fn, []byte(md), 0600)
			count++
		}
	}
	a.statusMsg = fmt.Sprintf("Exported %d notes to %s", count, outDir)
}

func (a *App) exportCurrentNote() {
	if a.currentNote == nil {
		return
	}
	fn := fmt.Sprintf("%s.md", a.currentNote.Title)
	md := fmt.Sprintf("# %s\n\n%s\n", a.currentNote.Title, markup.RenderToMarkdown(a.currentNote.Body))
	_ = os.WriteFile(fn, []byte(md), 0600)
	a.statusMsg = fmt.Sprintf("Exported note to %s", fn)
}

func (a *App) exportCurrentNotePDF() {
	if a.currentNote == nil {
		return
	}
	outDir := "."
	if a.currentBook != nil {
		outDir = filepath.Join(a.config.DataDir, "books", a.currentBook.Slug)
		outDir = export.GetExportDir(outDir)
	}
	fn := filepath.Join(outDir, sanitizeForFilename(a.currentNote.Title)+".pdf")
	if err := export.ExportNoteToPDF(a.currentNote, fn); err != nil {
		a.statusMsg = fmt.Sprintf("PDF export error: %v", err)
		return
	}
	a.statusMsg = fmt.Sprintf("✓ Exported PDF to %s", fn)
}

func (a *App) exportCurrentNoteDOCX() {
	if a.currentNote == nil {
		return
	}
	outDir := "."
	if a.currentBook != nil {
		outDir = filepath.Join(a.config.DataDir, "books", a.currentBook.Slug)
		outDir = export.GetExportDir(outDir)
	}
	fn := filepath.Join(outDir, sanitizeForFilename(a.currentNote.Title)+".docx")
	if err := export.ExportNoteToDOCX(a.currentNote, fn); err != nil {
		a.statusMsg = fmt.Sprintf("DOCX export error: %v", err)
		return
	}
	a.statusMsg = fmt.Sprintf("✓ Exported Word to %s", fn)
}

func sanitizeForFilename(name string) string {
	r := strings.NewReplacer("/", "_", "\\", "_", ":", "_", "*", "_", "?", "_", "\"", "_", "<", "_", ">", "_", "|", "_", " ", "_")
	s := r.Replace(name)
	if len(s) > 50 {
		s = s[:50]
	}
	return s
}

func (a *App) insertImage(path, alt, align string, widthPct int) {
	if a.currentNote == nil || a.screen != ScreenNoteEdit {
		return
	}
	widthStr := "auto"
	if widthPct > 0 && widthPct <= 100 {
		widthStr = fmt.Sprintf("%d%%", widthPct)
	}
	imgLine := fmt.Sprintf("\n!img[%s](%s|%s|%s)\n", alt, path, align, widthStr)

	// Add image ref to the note's Images list
	img := &model.NoteImage{
		ID:    model.NewID(),
		Path:  path,
		Alt:   alt,
		Align: align,
		Width: widthPct,
	}
	a.currentNote.Images = append(a.currentNote.Images, img)

	// Insert markup into editor
	a.editorModel.InsertText(imgLine)
	a.statusMsg = fmt.Sprintf("Image inserted: %s [%s, %s]", filepath.Base(path), align, widthStr)
}

func (a *App) triggerAIRefine() tea.Cmd {
	if a.currentNote == nil || a.currentChapter == nil {
		return nil
	}

	// 1. Sync latest text from editor
	if a.screen == ScreenNoteEdit {
		a.currentNote.Body = a.editorModel.Content()
		a.currentNote.UpdatedAt = time.Now()
		_ = a.library.SaveChapter(a.currentBook.Slug, a.currentChapter)
	}

	// 2. Always reload env and rebuild AI client with latest model & key from .env
	config.LoadEnv()
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" && a.config != nil {
		apiKey = a.config.GeminiAPIKey
	}
	modelName := os.Getenv("GEMINI_MODEL")
	if modelName == "" && a.config != nil && a.config.GeminiModel != "" {
		modelName = a.config.GeminiModel
	}
	if modelName == "" {
		modelName = "gemini-3.8-flash"
	}
	if apiKey != "" {
		a.aiClient = ai.NewClient(apiKey, modelName)
	}

	if a.aiClient == nil {
		a.statusMsg = "Gemini API key not found. Set GEMINI_API_KEY in .env or environment."
		return nil
	}

	// 3. Split body into blocks
	blocks := model.SplitBlocks(a.currentNote.Body)
	if len(blocks) == 0 {
		a.statusMsg = "Note is empty. Type some content first!"
		return nil
	}

	if a.currentChapter.ProcessedHashes == nil {
		a.currentChapter.ProcessedHashes = make(map[string]bool)
	}

	var dirtyBlocks []ai.BlockRequest
	for _, b := range blocks {
		h := model.HashBlock(b.Text)
		if !a.currentChapter.ProcessedHashes[h] {
			dirtyBlocks = append(dirtyBlocks, ai.BlockRequest{
				ID:   b.ID,
				Text: b.Text,
			})
		}
	}

	// If all blocks were marked processed, but the user explicitly pressed Ctrl+G:
	// Refine all blocks in the note!
	if len(dirtyBlocks) == 0 {
		for _, b := range blocks {
			dirtyBlocks = append(dirtyBlocks, ai.BlockRequest{
				ID:   b.ID,
				Text: b.Text,
			})
		}
	}

	// 4. Create snapshot of chapter first
	_ = a.library.CreateSnapshot(a.currentBook.Slug, a.currentChapter.ID)

	// 5. Trigger asynchronous AI processing
	a.aiProcessing = true
	a.statusMsg = fmt.Sprintf("✦ Refining %d block(s) with Gemini AI...", len(dirtyBlocks))

	client := a.aiClient
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		resp, err := client.RefineBlocks(ctx, dirtyBlocks)
		if err != nil {
			return aiErrorMsg{err: err}
		}
		diffs := ai.PrepareDiff(dirtyBlocks, resp)
		return aiDiffResultMsg{diffs: diffs}
	}
}

func (a *App) View() string {
	if a.width < 50 || a.height < 15 {
		return "Terminal too small. Minimum size: 50x15. Please resize."
	}

	if a.screen == ScreenLoading {
		return a.renderLoadingScreen()
	}

	if a.locked {
		return a.passphrase.View(a.width, a.height)
	}

	if a.showHelp {
		return renderHelpOverlay(a.width, a.height)
	}

	header := renderHeader(a.getCurrentBreadcrumb(), a.width, a.locked)
	footer := renderFooter(a.getFooterHints(), a.statusMsg, a.width)

	headerHeight := lipgloss.Height(header)
	footerHeight := lipgloss.Height(footer)

	availHeight := a.height - headerHeight - footerHeight
	if availHeight < 3 {
		availHeight = 3
	}

	var content string
	switch a.screen {
	case ScreenLibrary:
		content = a.libraryList.View()
	case ScreenBook:
		content = a.bookList.View()
	case ScreenChapter:
		content = a.chapterList.View()
	case ScreenNoteView:
		content = a.noteView.View()
	case ScreenNoteEdit:
		content = a.editorModel.View()
	case ScreenSearch:
		content = a.searchModel.View()
	case ScreenDiffReview:
		content = a.diffModel.View()
	case ScreenSnapshots:
		content = a.snapshotModel.View()
	case ScreenTagFilter:
		content = a.tagModel.View()
	case ScreenNewItem, ScreenRename:
		content = lipgloss.Place(a.width-2, availHeight, lipgloss.Center, lipgloss.Center, a.inputOverlay.View())
	case ScreenConfirmDelete:
		content = lipgloss.Place(a.width-2, availHeight, lipgloss.Center, lipgloss.Center, a.confirmDialog.View())
	}

	panel := renderGlassPanel(content, a.width, availHeight)
	rendered := lipgloss.JoinVertical(lipgloss.Left, header, panel, footer)

	// Strict line clamp to prevent terminal scrolling and duplicate footers
	lines := strings.Split(rendered, "\n")
	if a.height > 0 && len(lines) > a.height {
		lines = lines[:a.height]
	}
	return strings.Join(lines, "\n")
}

func (a *App) pushScreen(s Screen) {
	a.prevScreens = append(a.prevScreens, a.screen)
	a.screen = s
}

func (a *App) popScreen() {
	if len(a.prevScreens) > 0 {
		a.screen = a.prevScreens[len(a.prevScreens)-1]
		a.prevScreens = a.prevScreens[:len(a.prevScreens)-1]
	} else {
		a.screen = ScreenLibrary
	}
	if a.screen == ScreenLibrary {
		a.currentBook = nil
		a.currentChapter = nil
		a.currentNote = nil
		a.libraryList.Refresh()
	} else if a.screen == ScreenBook {
		a.currentChapter = nil
		a.currentNote = nil
		a.bookList.Refresh()
	}
}

func (a *App) lock() {
	// If in editor, save current note first
	if a.screen == ScreenNoteEdit && a.currentNote != nil && a.currentChapter != nil {
		a.currentNote.Body = a.editorModel.Content()
		a.currentNote.UpdatedAt = time.Now()
		_ = a.library.SaveChapter(a.currentBook.Slug, a.currentChapter)
	}

	a.locked = true
	a.library.Lock()
	a.screen = ScreenPassphrase
	a.passphrase.Reset()
}

func (a *App) buildSearchIndex() {
	if a.library == nil || a.searchIndex == nil {
		return
	}

	allNotes := a.library.GetAllNotes()
	entries := make([]search.SearchEntry, len(allNotes))
	for i, ref := range allNotes {
		tags := model.ExtractTags(ref.Note.Body)
		entries[i] = search.SearchEntry{
			BookSlug:     ref.BookSlug,
			BookName:     ref.BookName,
			ChapterID:    ref.ChapterID,
			ChapterTitle: ref.ChapterTitle,
			NoteID:       ref.Note.ID,
			NoteTitle:    ref.Note.Title,
			Body:         markup.StripMarkup(ref.Note.Body),
			RawBody:      ref.Note.Body,
			Tags:         tags,
			Breadcrumb:   fmt.Sprintf("%s › %s › %s", ref.BookName, ref.ChapterTitle, ref.Note.Title),
		}
	}
	a.searchIndex.Build(entries)
}

func (a *App) getCurrentBreadcrumb() string {
	switch a.screen {
	case ScreenLibrary:
		return "Books (D:\\books)"
	case ScreenBook:
		if a.currentBook != nil {
			return fmt.Sprintf("Books (D:\\books) › %s", a.currentBook.DisplayName)
		}
		return "Books (D:\\books)"
	case ScreenNoteEdit, ScreenNoteView:
		bc := "Books (D:\\books)"
		if a.currentBook != nil {
			bc += " › " + a.currentBook.DisplayName
		}
		if a.currentChapter != nil {
			bc += " › " + a.currentChapter.Title + " (.vault)"
		}
		return bc
	default:
		bc := "Books (D:\\books)"
		if a.currentBook != nil {
			bc += " › " + a.currentBook.DisplayName
		}
		if a.currentChapter != nil {
			bc += " › " + a.currentChapter.Title + " (.vault)"
		}
		return bc
	}
}

func (a *App) getFooterHints() []KeyHint {
	switch a.screen {
	case ScreenLibrary:
		return []KeyHint{
			{Key: "←/→ or ↑/↓", Description: "Browse Books"},
			{Key: "Enter", Description: "Open Book"},
			{Key: "Ctrl+N / N", Description: "New Book"},
			{Key: "D", Description: "Delete"},
			{Key: "S", Description: "Sort"},
			{Key: "/", Description: "Search"},
			{Key: "?", Description: "Help"},
			{Key: "Q", Description: "Quit"},
		}
	case ScreenBook:
		return []KeyHint{
			{Key: "←/→ or ↑/↓", Description: "Browse Vaults"},
			{Key: "Enter", Description: "Read Mode"},
			{Key: "E", Description: "Edit Notes"},
			{Key: "Ctrl+N / N", Description: "New Vault"},
			{Key: "D", Description: "Delete"},
			{Key: "Esc", Description: "Back to Books"},
		}
	case ScreenNoteView:
		return []KeyHint{
			{Key: "E", Description: "Edit Notes"},
			{Key: "↑/↓ or J/K", Description: "Scroll"},
			{Key: "Ctrl+P", Description: "Export PDF"},
			{Key: "Ctrl+W", Description: "Export Word"},
			{Key: "Esc", Description: "Back to Vaults"},
		}
	case ScreenNewItem:
		return []KeyHint{
			{Key: "Enter", Description: "Confirm & Create"},
			{Key: "Esc", Description: "Cancel"},
		}
	case ScreenRename:
		return []KeyHint{
			{Key: "Enter", Description: "Confirm Rename"},
			{Key: "Esc", Description: "Cancel"},
		}
	case ScreenConfirmDelete:
		if strings.HasPrefix(a.confirmDialog.Action, "export") {
			return []KeyHint{
				{Key: "Enter", Description: "Confirm Export"},
				{Key: "Esc", Description: "Cancel"},
			}
		}
		return []KeyHint{
			{Key: "Enter", Description: "Confirm Delete"},
			{Key: "Esc", Description: "Cancel"},
		}
	case ScreenNoteEdit:
		return []KeyHint{
			{Key: "Ctrl+R", Description: "Read Mode"},
			{Key: "Ctrl+F", Description: "Format"},
			{Key: "Ctrl+G", Description: "AI Refine"},
			{Key: "Ctrl+I", Description: "Image"},
			{Key: "Ctrl+P", Description: "PDF"},
			{Key: "Ctrl+W", Description: "Word"},
			{Key: "Esc", Description: "Save & Exit"},
		}
	case ScreenSearch:
		return []KeyHint{
			{Key: "Type", Description: "Search"},
			{Key: "↑/↓", Description: "Select Result"},
			{Key: "Enter", Description: "Open Note"},
			{Key: "Esc", Description: "Back"},
		}
	case ScreenDiffReview:
		return []KeyHint{
			{Key: "Y", Description: "Accept"},
			{Key: "N", Description: "Reject"},
			{Key: "A", Description: "Accept All"},
			{Key: "K", Description: "Keep As-Is"},
			{Key: "Enter", Description: "Apply"},
			{Key: "Esc", Description: "Cancel"},
		}
	default:
		return []KeyHint{
			{Key: "↑/↓", Description: "Navigate"},
			{Key: "Esc", Description: "Back"},
			{Key: "?", Description: "Help"},
		}
	}
}

func (a *App) renderLoadingScreen() string {
	percent := (a.loadingStep * 100) / a.maxLoadingSteps
	if percent > 100 {
		percent = 100
	}

	totalBars := 26
	filled := (a.loadingStep * totalBars) / a.maxLoadingSteps
	if filled > totalBars {
		filled = totalBars
	}
	bar := strings.Repeat("█", filled) + strings.Repeat("░", totalBars-filled)

	msg := "✦ Opening Books & Encrypted Vaults..."
	sub := "Scanning D:\\books\\ directory..."
	if a.loadingStep > 20 {
		msg = "✦ Preparing Bookshelf..."
		sub = "Ready. Welcome to Kosha."
	} else if a.loadingStep > 10 {
		msg = "✦ Decrypting Vault Chapter Headers..."
		sub = "Authenticating XChaCha20-Poly1305 headers..."
	}

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(ColorSaffron)
	barStyle := lipgloss.NewStyle().Foreground(ColorSaffron)
	muted := lipgloss.NewStyle().Foreground(ColorMutedText)
	boxStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(ColorGlassBorder).
		Padding(1, 4).
		Align(lipgloss.Center)

	content := fmt.Sprintf("%s\n%s\n\n%s\n\n%s [%s] %d%%\n\n%s\n\n%s",
		titleStyle.Render("⚡  K O S H A   V A U L T  ⚡"),
		muted.Render("[ Sanskrit कोश: Sacred Treasury ]"),
		lipgloss.NewStyle().Bold(true).Foreground(ColorBodyText).Render(msg),
		barStyle.Render("⏳"),
		barStyle.Render(bar),
		percent,
		muted.Render(sub),
		muted.Render("Press [Space / Enter] to skip"),
	)

	box := boxStyle.Render(content)
	return lipgloss.Place(a.width, a.height, lipgloss.Center, lipgloss.Center, box)
}
