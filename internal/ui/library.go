package ui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/YaswanthKumarMallela01/kosha/internal/model"
)

type LibraryModel struct {
	app      *App
	books    []*model.Book
	selected int
	w, h     int
}

func NewLibraryModel(app *App) LibraryModel {
	return LibraryModel{
		app:      app,
		selected: 0,
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
	m.books = books

	if m.selected >= len(m.books) {
		if len(m.books) > 0 {
			m.selected = len(m.books) - 1
		} else {
			m.selected = 0
		}
	}
}

func (m *LibraryModel) SelectedBook() *model.Book {
	if len(m.books) == 0 || m.selected < 0 || m.selected >= len(m.books) {
		return nil
	}
	return m.books[m.selected]
}

func (m LibraryModel) Update(msg tea.Msg) (LibraryModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "left", "h", "up", "k":
			if m.selected > 0 {
				m.selected--
			} else if len(m.books) > 0 {
				m.selected = len(m.books) - 1
			}
		case "right", "l", "down", "j":
			if m.selected < len(m.books)-1 {
				m.selected++
			} else {
				m.selected = 0
			}
		case "home":
			m.selected = 0
		case "end":
			if len(m.books) > 0 {
				m.selected = len(m.books) - 1
			}
		case "enter":
			if sel := m.SelectedBook(); sel != nil {
				m.app.currentBook = sel
				m.app.bookList.Refresh()
				m.app.pushScreen(ScreenBook)
				return m, nil
			}
		case "ctrl+n", "n", "N":
			m.app.inputOverlay.Open("Create New Book in D:\\books (e.g. Operating Systems):", "", "new_book", "")
			m.app.pushScreen(ScreenNewItem)
		case "r", "R":
			if sel := m.SelectedBook(); sel != nil {
				m.app.inputOverlay.Open("Rename Book:", sel.DisplayName, "rename_book", sel.Slug)
				m.app.pushScreen(ScreenRename)
			}
		case "d", "D":
			if sel := m.SelectedBook(); sel != nil {
				m.app.confirmDialog.Open("Delete book '"+sel.DisplayName+"' and all its vault files from disk?", "delete_book", sel.Slug)
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
			if sel := m.SelectedBook(); sel != nil {
				m.app.exportBook(sel.Slug)
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
	if len(m.books) == 0 {
		emptyCard := renderSmallBookCard("NO BOOKS", 0, false)
		info := lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(ColorGlassBorder).
			Padding(1, 2).
			Width(50).
			Render("No books found in D:\\books\\\n\nPress [Ctrl+N] or [N] to create your first book!\nExample: \"Operating Systems\" or \"Algorithms\"")
		return lipgloss.JoinVertical(lipgloss.Left, emptyCard, "\n", info)
	}

	selIdx := m.selected
	if selIdx < 0 || selIdx >= len(m.books) {
		selIdx = 0
	}

	// 1. Render all books side-wise using compact ASCII book art
	cards := make([]string, len(m.books))
	for i, b := range m.books {
		chaps, _ := m.app.library.LoadChapters(b.Slug)
		cards[i] = renderSmallBookCard(b.DisplayName, len(chaps), i == selIdx)
	}

	cardRow, navIndicator := renderHorizontalCards(cards, selIdx, m.w)

	// 2. Info box for currently selected book
	selBook := m.books[selIdx]
	chaps, _ := m.app.library.LoadChapters(selBook.Slug)
	chapCount := len(chaps)

	boxWidth := 56
	if m.w-8 > boxWidth && m.w-8 < 90 {
		boxWidth = m.w - 8
	}

	infoBox := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(ColorSaffron).
		Padding(0, 1).
		Width(boxWidth).
		Render(fmt.Sprintf("%s\n  ├── 📁 Path:     %s\n  ├── 🔐 Vaults:   %d encrypted vault chapter(s)\n  └── 🕒 Created:  %s",
			lipgloss.NewStyle().Bold(true).Foreground(ColorSaffron).Render("📚 SELECTED BOOK: "+selBook.DisplayName),
			MutedStyle.Render(selBook.Path),
			chapCount,
			MutedStyle.Render(formatTime(selBook.CreatedAt)),
		))

	guide := MutedStyle.Render("Press [Enter] to open this book & view its vaults  •  [Ctrl+N] Create New Book")

	return lipgloss.JoinVertical(lipgloss.Left,
		cardRow,
		"\n",
		MutedStyle.Render(navIndicator),
		"\n",
		infoBox,
		"\n",
		guide,
	)
}
