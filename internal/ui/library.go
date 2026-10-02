package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/YaswanthKumarMallela01/kosha/internal/model"
)

type LibraryModel struct {
	app  *App
	list ListModel
	w, h int
}

func NewLibraryModel(app *App) LibraryModel {
	return LibraryModel{
		app:  app,
		list: ListModel{},
	}
}

func (m *LibraryModel) SetSize(w, h int) {
	m.w = w
	m.h = h
}

func (m *LibraryModel) Refresh() {
	if m.app == nil || m.app.library == nil {
		return
	}

	books := make([]*model.Book, len(m.app.library.Books))
	copy(books, m.app.library.Books)
	model.SortBooks(books, m.app.ascending)

	items := make([]ListItem, len(books))
	for i, b := range books {
		chaps, _ := m.app.library.LoadChapters(b.Slug)
		chapCount := len(chaps)
		totalNotes := 0
		for _, ch := range chaps {
			totalNotes += len(ch.Notes)
		}

		chapStr := fmt.Sprintf("%d encrypted vault section(s) (.vault) | %d note(s)", chapCount, totalNotes)
		if chapCount == 1 {
			chapStr = fmt.Sprintf("1 encrypted vault section (.vault) | %d note(s)", totalNotes)
		}

		items[i] = ListItem{
			ID:          b.Slug,
			Type:        "BOOK",
			Title:       b.DisplayName,
			Path:        b.Path,
			Subtitle:    chapStr,
			Description: formatTime(b.CreatedAt),
		}
	}

	m.list.Items = items
	if m.list.Selected >= len(items) {
		if len(items) > 0 {
			m.list.Selected = len(items) - 1
		} else {
			m.list.Selected = 0
		}
	}
}

func (m LibraryModel) Update(msg tea.Msg) (LibraryModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k", "left", "h":
			m.list.Up()
		case "down", "j", "right", "l":
			m.list.Down()
		case "pgdown":
			m.list.PageDown(1)
		case "pgup":
			m.list.PageUp(1)
		case "home":
			m.list.Home()
		case "end":
			m.list.End()
		case "enter":
			if sel := m.list.SelectedItem(); sel != nil {
				for _, b := range m.app.library.Books {
					if b.Slug == sel.ID {
						m.app.currentBook = b
						m.app.bookList.Refresh()
						m.app.pushScreen(ScreenBook)
						return m, nil
					}
				}
			}
		case "ctrl+n", "n", "N":
			m.app.inputOverlay.Open("Create New Book in D:\\books (e.g. Operating Systems):", "", "new_book", "")
			m.app.pushScreen(ScreenNewItem)
		case "r", "R":
			if sel := m.list.SelectedItem(); sel != nil {
				m.app.inputOverlay.Open("Rename Book:", sel.Title, "rename_book", sel.ID)
				m.app.pushScreen(ScreenRename)
			}
		case "d", "D":
			if sel := m.list.SelectedItem(); sel != nil {
				m.app.confirmDialog.Open("Delete book '"+sel.Title+"' and all its vault files from disk?", "delete_book", sel.ID)
				m.app.pushScreen(ScreenConfirmDelete)
			}
		case "s", "S":
			m.app.ascending = !m.app.ascending
			m.Refresh()
			if m.app.ascending {
				m.app.statusMsg = "Sorted: oldest first"
			} else {
				m.app.statusMsg = "Sorted: newest first"
			}
		case "x", "X":
			if sel := m.list.SelectedItem(); sel != nil {
				m.app.exportBook(sel.ID)
			}
		case "t", "T":
			m.app.tagModel.Refresh()
			m.app.pushScreen(ScreenTagFilter)
		case "/", "ctrl+k":
			m.app.searchModel.Reset()
			m.app.pushScreen(ScreenSearch)
		case "q", "Q":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m LibraryModel) View() string {
	if len(m.list.Items) == 0 {
		asciiEmpty := renderAsciiBookShape("NO BOOKS YET", m.w, false)
		info := lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(ColorGlassBorder).
			Padding(1, 2).
			Width(52).
			Render("No books found in D:\\books\\\n\nPress [Ctrl+N] or [N] to create a new book!\nExample: \"Operating Systems\" or \"Algorithms\"")
		return lipgloss.JoinVertical(lipgloss.Left, asciiEmpty, info)
	}

	selIdx := m.list.Selected
	if selIdx < 0 || selIdx >= len(m.list.Items) {
		selIdx = 0
	}
	selItem := m.list.Items[selIdx]

	// 1. Bookshelf Tab Bar showing all available books
	var shelfTabs []string
	for i, item := range m.list.Items {
		tabText := fmt.Sprintf("📖 %s", item.Title)
		if i == selIdx {
			tabStyle := lipgloss.NewStyle().
				Background(ColorSaffron).
				Foreground(lipgloss.Color("#000000")).
				Bold(true).
				Padding(0, 1)
			shelfTabs = append(shelfTabs, tabStyle.Render(tabText))
		} else {
			tabStyle := lipgloss.NewStyle().
				Background(ColorGlassBorder).
				Foreground(ColorBodyText).
				Padding(0, 1)
			shelfTabs = append(shelfTabs, tabStyle.Render(tabText))
		}
	}
	shelfBar := strings.Join(shelfTabs, "  ")

	// 2. The physical ASCII Book Shape with this book's title emblazoned on it,
	// and the book information box down side
	bookCard := renderBookCard(selItem.Title, selItem.Path, selItem.Subtitle, selItem.Description, m.w, true)

	// 3. Navigation guide
	navGuide := MutedStyle.Render(fmt.Sprintf("── [Book %d of %d] ── Use [↑/↓] or [←/→] to browse books ──", selIdx+1, len(m.list.Items)))

	return lipgloss.JoinVertical(lipgloss.Left, shelfBar, "\n", bookCard, "\n", navGuide)
}
