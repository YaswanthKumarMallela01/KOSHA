package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/YaswanthKumarMallela01/kosha/internal/search"
)

type SearchModel struct {
	app      *App
	input    textinput.Model
	results  []search.SearchResult
	selected int
	offset   int
	w, h     int
}

func NewSearchModel(app *App) SearchModel {
	ti := textinput.New()
	ti.Placeholder = "Type to search notes, tags, content..."
	ti.Focus()
	ti.CharLimit = 100
	ti.Width = 50

	return SearchModel{
		app:   app,
		input: ti,
	}
}

func (m *SearchModel) SetSize(w, h int) {
	m.w = w
	m.h = h
}

func (m *SearchModel) Reset() {
	m.input.Reset()
	m.results = nil
	m.selected = 0
	m.offset = 0
	m.input.Focus()
}

func (m *SearchModel) SetQuery(query string) {
	m.input.SetValue(query)
	m.executeSearch()
}

func (m *SearchModel) executeSearch() {
	q := strings.TrimSpace(m.input.Value())
	if q == "" || m.app.searchIndex == nil {
		m.results = nil
		m.selected = 0
		return
	}

	m.results = m.app.searchIndex.Search(q, 50)
	if m.selected >= len(m.results) {
		m.selected = 0
	}
}

func (m SearchModel) Update(msg tea.Msg) (SearchModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.app.popScreen()
			return m, nil
		case "up", "ctrl+p":
			if m.selected > 0 {
				m.selected--
				if m.selected < m.offset {
					m.offset = m.selected
				}
			}
			return m, nil
		case "down", "ctrl+n":
			if m.selected < len(m.results)-1 {
				m.selected++
			}
			return m, nil
		case "enter":
			if len(m.results) > 0 && m.selected < len(m.results) {
				entry := m.results[m.selected].Entry
				m.app.jumpToNoteID(entry.BookSlug, entry.ChapterID, entry.NoteID)
				return m, nil
			}
		}
	}

	var cmd tea.Cmd
	prevVal := m.input.Value()
	m.input, cmd = m.input.Update(msg)
	if m.input.Value() != prevVal {
		m.executeSearch()
	}

	return m, cmd
}

func (m SearchModel) View() string {
	viewWidth := m.w - 8
	if viewWidth < 20 {
		viewWidth = 20
	}

	searchHeader := lipgloss.NewStyle().
		Background(ColorGlassFill).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(ColorSaffron).
		Padding(0, 1).
		Width(viewWidth).
		Render("🔍 Search: " + m.input.View())

	var resultsView string
	if len(m.results) == 0 {
		if strings.TrimSpace(m.input.Value()) == "" {
			resultsView = MutedStyle.Render("  Type to start searching across titles, tags, and note content...")
		} else {
			resultsView = MutedStyle.Render("  No matching notes found.")
		}
	} else {
		maxVisible := (m.h - 10) / 3
		if maxVisible < 2 {
			maxVisible = 2
		}

		if m.selected < m.offset {
			m.offset = m.selected
		} else if m.selected >= m.offset+maxVisible {
			m.offset = m.selected - maxVisible + 1
		}
		if m.offset < 0 {
			m.offset = 0
		}

		end := m.offset + maxVisible
		if end > len(m.results) {
			end = len(m.results)
		}

		var sb strings.Builder
		for i := m.offset; i < end; i++ {
			res := m.results[i]
			isSel := (i == m.selected)

			breadcrumb := res.Entry.Breadcrumb
			snippet := res.Snippet
			if snippet == "" {
				snippet = res.Entry.Body
			}
			if len(snippet) > 80 {
				snippet = snippet[:77] + "..."
			}
			snippet = strings.ReplaceAll(snippet, "\n", " ")

			titleLine := res.Entry.NoteTitle
			subLine := fmt.Sprintf("%s  •  %s", MutedStyle.Render(breadcrumb), snippet)

			if isSel {
				sb.WriteString(SelectedItemStyle.Width(viewWidth).Render("▸ "+titleLine) + "\n")
				sb.WriteString(lipgloss.NewStyle().Background(ColorGlassBorder).Foreground(ColorBodyText).Width(viewWidth).Render("  "+subLine) + "\n\n")
			} else {
				sb.WriteString(ListItemStyle.Width(viewWidth).Render("  "+titleLine) + "\n")
				sb.WriteString(MutedStyle.Width(viewWidth).Render("    "+subLine) + "\n\n")
			}
		}

		statusLine := fmt.Sprintf(" Showing %d of %d results [↑/↓ to navigate, Enter to open]", end-m.offset, len(m.results))
		sb.WriteString(MutedStyle.Render(statusLine))

		resultsView = sb.String()
	}

	return lipgloss.JoinVertical(lipgloss.Left, searchHeader, "\n", resultsView)
}
