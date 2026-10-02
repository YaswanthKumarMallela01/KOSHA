package ui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

type TagFilterModel struct {
	app        *App
	tagsList   ListModel
	notesList  ListModel
	showingNotes bool
	activeTag  string
	w, h       int
}

func NewTagFilterModel(app *App) TagFilterModel {
	return TagFilterModel{
		app:      app,
		tagsList: ListModel{},
		notesList: ListModel{},
	}
}

func (m *TagFilterModel) SetSize(w, h int) {
	m.w = w
	m.h = h
}

func (m *TagFilterModel) Refresh() {
	if m.app == nil || m.app.searchIndex == nil {
		return
	}

	tags := m.app.searchIndex.GetAllTags()
	items := make([]ListItem, len(tags))
	for i, t := range tags {
		items[i] = ListItem{
			ID:       t.Name,
			Title:    PinkStyle.Render(t.Name),
			Subtitle: fmt.Sprintf("%d note(s)", t.Count),
		}
	}

	m.tagsList.Items = items
	m.tagsList.Selected = 0
	m.tagsList.Offset = 0
	m.showingNotes = false
	m.activeTag = ""
}

func (m TagFilterModel) Update(msg tea.Msg) (TagFilterModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "backspace":
			if m.showingNotes {
				m.showingNotes = false
				return m, nil
			}
			m.app.popScreen()
			return m, nil
		case "j", "down":
			if m.showingNotes {
				m.notesList.Down()
			} else {
				m.tagsList.Down()
			}
		case "k", "up":
			if m.showingNotes {
				m.notesList.Up()
			} else {
				m.tagsList.Up()
			}
		case "enter":
			if !m.showingNotes {
				if sel := m.tagsList.SelectedItem(); sel != nil {
					m.activeTag = sel.ID
					entries := m.app.searchIndex.GetNotesByTag(m.activeTag)
					items := make([]ListItem, len(entries))
					for i, e := range entries {
						items[i] = ListItem{
							ID:          e.NoteID,
							Title:       e.NoteTitle,
							Subtitle:    e.Breadcrumb,
							Description: e.BookSlug + "/" + e.ChapterID,
						}
					}
					m.notesList.Items = items
					m.notesList.Selected = 0
					m.notesList.Offset = 0
					m.showingNotes = true
				}
			} else {
				if sel := m.notesList.SelectedItem(); sel != nil {
					entries := m.app.searchIndex.GetNotesByTag(m.activeTag)
					for _, e := range entries {
						if e.NoteID == sel.ID {
							m.app.jumpToNoteID(e.BookSlug, e.ChapterID, e.NoteID)
							return m, nil
						}
					}
				}
			}
		}
	}
	return m, nil
}

func (m TagFilterModel) View() string {
	listHeight := m.h - 6
	if listHeight < 5 {
		listHeight = 5
	}

	if m.showingNotes {
		header := SaffronStyle.Render(fmt.Sprintf("Notes tagged with %s:", PinkStyle.Render(m.activeTag)))
		list := renderList(m.notesList.Items, m.notesList.Selected, m.notesList.Offset, m.w, listHeight)
		footer := MutedStyle.Render("[Enter] Open Note  [Esc] Back to Tags")
		return header + "\n\n" + list + "\n" + footer
	}

	header := SaffronStyle.Render("All Tags:")
	list := renderList(m.tagsList.Items, m.tagsList.Selected, m.tagsList.Offset, m.w, listHeight)
	footer := MutedStyle.Render("[Enter] View Notes  [Esc] Go Back")
	return header + "\n\n" + list + "\n" + footer
}
