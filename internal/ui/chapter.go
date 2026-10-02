package ui

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/YaswanthKumarMallela01/kosha/internal/model"
)

type ChapterModel struct {
	app  *App
	list ListModel
	w, h int
}

func NewChapterModel(app *App) ChapterModel {
	return ChapterModel{
		app:  app,
		list: ListModel{},
	}
}

func (m *ChapterModel) SetSize(w, h int) {
	m.w = w
	m.h = h
}

func (m *ChapterModel) Refresh() {
	if m.app == nil || m.app.currentChapter == nil {
		return
	}

	notes := make([]*model.Note, len(m.app.currentChapter.Notes))
	copy(notes, m.app.currentChapter.Notes)
	model.SortNotes(notes, m.app.ascending)

	items := make([]ListItem, len(notes))
	for i, n := range notes {
		tags := model.ExtractTags(n.Body)
		tagStr := ""
		if len(tags) > 0 {
			tagStr = strings.Join(tags, " ")
		}

		items[i] = ListItem{
			ID:          n.ID,
			Title:       n.Title,
			Subtitle:    tagStr,
			Description: "Updated: " + formatTime(n.UpdatedAt),
			Pinned:      n.Pinned,
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

func (m ChapterModel) Update(msg tea.Msg) (ChapterModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "j", "down":
			m.list.Down()
		case "k", "up":
			m.list.Up()
		case "pgdown":
			m.list.PageDown(5)
		case "pgup":
			m.list.PageUp(5)
		case "home":
			m.list.Home()
		case "end":
			m.list.End()
		case "enter":
			if sel := m.list.SelectedItem(); sel != nil {
				for _, n := range m.app.currentChapter.Notes {
					if n.ID == sel.ID {
						m.app.currentNote = n
						m.app.noteView.Refresh()
						m.app.pushScreen(ScreenNoteView)
						return m, nil
					}
				}
			}
		case "e":
			if sel := m.list.SelectedItem(); sel != nil {
				for _, n := range m.app.currentChapter.Notes {
					if n.ID == sel.ID {
						m.app.currentNote = n
						m.app.openEditor(n)
						return m, nil
					}
				}
			}
		case "n":
			m.app.inputOverlay.Open("Enter new note title:", "", "new_note", "")
			m.app.pushScreen(ScreenNewItem)
		case "p":
			if sel := m.list.SelectedItem(); sel != nil {
				for _, n := range m.app.currentChapter.Notes {
					if n.ID == sel.ID {
						n.Pinned = !n.Pinned
						n.UpdatedAt = time.Now()
						m.app.library.SaveChapter(m.app.currentBook.Slug, m.app.currentChapter)
						m.Refresh()
						if n.Pinned {
							m.app.statusMsg = "Note pinned"
						} else {
							m.app.statusMsg = "Note unpinned"
						}
						return m, nil
					}
				}
			}
		case "r":
			if sel := m.list.SelectedItem(); sel != nil {
				m.app.inputOverlay.Open("Rename note:", sel.Title, "rename_note", sel.ID)
				m.app.pushScreen(ScreenRename)
			}
		case "d":
			if sel := m.list.SelectedItem(); sel != nil {
				m.app.confirmDialog.Open("Delete note '"+sel.Title+"'?", "delete_note", sel.ID)
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
			if m.app.currentBook != nil && m.app.currentChapter != nil {
				m.app.exportChapter(m.app.currentBook.Slug, m.app.currentChapter.Title)
			}
		case "esc", "backspace":
			m.app.popScreen()
		}
	}
	return m, nil
}

func (m ChapterModel) View() string {
	listHeight := m.h - 6
	if listHeight < 5 {
		listHeight = 5
	}
	return renderList(m.list.Items, m.list.Selected, m.list.Offset, m.w, listHeight)
}
