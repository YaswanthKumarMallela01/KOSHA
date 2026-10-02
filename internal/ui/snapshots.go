package ui

import (
	"fmt"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/YaswanthKumarMallela01/kosha/internal/config"
	"github.com/YaswanthKumarMallela01/kosha/internal/vault"
)

type SnapshotModel struct {
	app      *App
	list     ListModel
	preview  string
	inPreview bool
	w, h     int
}

func NewSnapshotModel(app *App) SnapshotModel {
	return SnapshotModel{
		app:  app,
		list: ListModel{},
	}
}

func (m *SnapshotModel) SetSize(w, h int) {
	m.w = w
	m.h = h
}

func (m *SnapshotModel) Refresh() {
	if m.app == nil || m.app.currentBook == nil || m.app.currentChapter == nil {
		return
	}

	bookDir := config.BookDir(m.app.library.DataDir, m.app.currentBook.Slug)
	snapshots, err := vault.ListSnapshots(bookDir, m.app.currentChapter.ID)
	if err != nil || len(snapshots) == 0 {
		m.list.Items = nil
		return
	}

	items := make([]ListItem, len(snapshots))
	// Newest first
	for i, s := range snapshots {
		filename := filepath.Base(s.Path)
		items[i] = ListItem{
			ID:          s.Path,
			Title:       "Snapshot: " + formatTime(s.Timestamp),
			Subtitle:    filename,
			Description: "",
		}
	}

	m.list.Items = items
	m.inPreview = false
	m.preview = ""
}

func (m SnapshotModel) Update(msg tea.Msg) (SnapshotModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "backspace":
			if m.inPreview {
				m.inPreview = false
				return m, nil
			}
			m.app.popScreen()
			return m, nil
		case "j", "down":
			if !m.inPreview {
				m.list.Down()
			}
		case "k", "up":
			if !m.inPreview {
				m.list.Up()
			}
		case "enter":
			// Preview selected snapshot
			if sel := m.list.SelectedItem(); sel != nil {
				ch, err := vault.LoadSnapshot(sel.ID, m.app.library.MasterKey)
				if err == nil && ch != nil {
					m.preview = fmt.Sprintf("Snapshot: %s\nNotes count: %d\nCreated: %s\n\n",
						ch.Title, len(ch.Notes), formatTime(ch.CreatedAt))
					for _, n := range ch.Notes {
						m.preview += fmt.Sprintf("• %s\n", n.Title)
					}
					m.inPreview = true
				} else {
					m.app.statusMsg = "Failed to load snapshot"
				}
			}
		case "r":
			// Restore selected snapshot
			if sel := m.list.SelectedItem(); sel != nil {
				// 1. Create a snapshot of current chapter first
				_ = m.app.library.CreateSnapshot(m.app.currentBook.Slug, m.app.currentChapter.ID)

				// 2. Load snapshot
				restoredChapter, err := vault.LoadSnapshot(sel.ID, m.app.library.MasterKey)
				if err == nil && restoredChapter != nil {
					// 3. Save restored chapter to disk
					m.app.currentChapter = restoredChapter
					_ = m.app.library.SaveChapter(m.app.currentBook.Slug, restoredChapter)
					m.app.chapterList.Refresh()
					m.app.statusMsg = "Chapter restored from snapshot"
					m.app.popScreen()
					return m, nil
				}
				m.app.statusMsg = "Failed to restore snapshot"
			}
		}
	}
	return m, nil
}

func (m SnapshotModel) View() string {
	if m.inPreview {
		previewStyle := lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(ColorSaffron).
			Padding(1, 2).
			Width(m.w - 8)

		content := fmt.Sprintf("%s\n\n%s\n\n%s",
			SaffronStyle.Render("SNAPSHOT PREVIEW"),
			m.preview,
			MutedStyle.Render("[r] Restore this snapshot  [Esc] Back to snapshots list"),
		)
		return previewStyle.Render(content)
	}

	listHeight := m.h - 6
	if listHeight < 5 {
		listHeight = 5
	}

	header := SaffronStyle.Render("Snapshots (Version History - Last 20 versions):")
	list := renderList(m.list.Items, m.list.Selected, m.list.Offset, m.w, listHeight)
	footer := MutedStyle.Render("[Enter] Preview  [r] Restore  [Esc] Go Back")

	return lipgloss.JoinVertical(lipgloss.Left, header, "\n", list, "\n", footer)
}
