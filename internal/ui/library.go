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
		chapStr := fmt.Sprintf("%d encrypted vault file(s) (.vault)", chapCount)
		if chapCount == 1 {
			chapStr = "1 encrypted vault file (.vault)"
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
		case "up", "k":
			m.list.Up()
		case "down", "j":
			m.list.Down()
		case "pgdown":
			m.list.PageDown(3)
		case "pgup":
			m.list.PageUp(3)
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
		case "n":
			m.app.inputOverlay.Open("Create New Book in D:\\books:", "", "new_book", "")
			m.app.pushScreen(ScreenNewItem)
		case "r":
			if sel := m.list.SelectedItem(); sel != nil {
				m.app.inputOverlay.Open("Rename Book:", sel.Title, "rename_book", sel.ID)
				m.app.pushScreen(ScreenRename)
			}
		case "d":
			if sel := m.list.SelectedItem(); sel != nil {
				m.app.confirmDialog.Open("Delete book '"+sel.Title+"' and its vault files from disk?", "delete_book", sel.ID)
				m.app.pushScreen(ScreenConfirmDelete)
			}
		case "s":
			m.app.ascending = !m.app.ascending
			m.Refresh()
			if m.app.ascending {
				m.app.statusMsg = "Sorted: oldest first"
			} else {
				m.app.statusMsg = "Sorted: newest first"
			}
		case "R": // Resurface jump
			if m.app.resurfaceNote != nil {
				m.app.jumpToNote(m.app.resurfaceNote)
			}
		case "x":
			if sel := m.list.SelectedItem(); sel != nil {
				m.app.exportBook(sel.ID)
			}
		case "t":
			m.app.tagModel.Refresh()
			m.app.pushScreen(ScreenTagFilter)
		case "/", "ctrl+k":
			m.app.searchModel.Reset()
			m.app.pushScreen(ScreenSearch)
		case "q":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m LibraryModel) View() string {
	listHeight := m.h - 8
	if listHeight < 5 {
		listHeight = 5
	}

	content := renderList(m.list.Items, m.list.Selected, m.list.Offset, m.w, listHeight)

	// Resurface card at bottom if available
	if m.app.resurfaceNote != nil && m.h > 18 {
		rn := m.app.resurfaceNote
		snippet := rn.Note.Body
		if len(snippet) > 60 {
			snippet = snippet[:57] + "..."
		}
		snippet = strings.ReplaceAll(snippet, "\n", " ")

		resurfaceCard := lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(ColorFrostEdge).
			Padding(0, 1).
			Width(m.w - 6).
			Render(fmt.Sprintf("%s %s › %s › %s  |  %s",
				SaffronStyle.Render("✦ RESURFACE [Shift+R]:"),
				rn.BookName, rn.ChapterTitle, rn.Note.Title,
				lipgloss.NewStyle().Foreground(ColorMutedText).Render(snippet),
			))

		content = lipgloss.JoinVertical(lipgloss.Left, content, resurfaceCard)
	}

	return content
}
