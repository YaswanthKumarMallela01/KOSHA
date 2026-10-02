package ui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/YaswanthKumarMallela01/kosha/internal/model"
)

type BookModel struct {
	app  *App
	list ListModel
	w, h int
}

func NewBookModel(app *App) BookModel {
	return BookModel{
		app:  app,
		list: ListModel{},
	}
}

func (m *BookModel) SetSize(w, h int) {
	m.w = w
	m.h = h
}

func (m *BookModel) Refresh() {
	if m.app == nil || m.app.currentBook == nil {
		return
	}

	chapters, err := m.app.library.LoadChapters(m.app.currentBook.Slug)
	if err != nil {
		m.list.Items = nil
		return
	}

	model.SortChapters(chapters, m.app.ascending)

	items := make([]ListItem, len(chapters))
	for i, ch := range chapters {
		noteCount := len(ch.Notes)
		noteStr := fmt.Sprintf("%d encrypted note(s) inside", noteCount)
		if noteCount == 1 {
			noteStr = "1 encrypted note inside"
		}

		items[i] = ListItem{
			ID:          ch.ID,
			Type:        "VAULT",
			Title:       ch.Title,
			Path:        ch.ID + ".vault",
			Subtitle:    noteStr,
			Description: formatTime(ch.CreatedAt),
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

func (m BookModel) Update(msg tea.Msg) (BookModel, tea.Cmd) {
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
				ch, err := m.app.library.GetChapter(m.app.currentBook.Slug, sel.ID)
				if err == nil && ch != nil {
					m.app.currentChapter = ch
					m.app.chapterList.Refresh()
					m.app.pushScreen(ScreenChapter)
					return m, nil
				}
			}
		case "n":
			m.app.inputOverlay.Open("Create New Chapter / Vault File in this book:", "", "new_chapter", "")
			m.app.pushScreen(ScreenNewItem)
		case "r":
			if sel := m.list.SelectedItem(); sel != nil {
				m.app.inputOverlay.Open("Rename Chapter:", sel.Title, "rename_chapter", sel.ID)
				m.app.pushScreen(ScreenRename)
			}
		case "d":
			if sel := m.list.SelectedItem(); sel != nil {
				m.app.confirmDialog.Open("Delete vault file '"+sel.Title+"' (.vault)?", "delete_chapter", sel.ID)
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
		case "x":
			if m.app.currentBook != nil {
				m.app.exportBook(m.app.currentBook.Slug)
			}
		case "esc", "backspace":
			m.app.popScreen()
		}
	}
	return m, nil
}

func (m BookModel) View() string {
	listHeight := m.h - 8
	if listHeight < 5 {
		listHeight = 5
	}
	return renderList(m.list.Items, m.list.Selected, m.list.Offset, m.w, listHeight)
}
