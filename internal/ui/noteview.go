package ui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/YaswanthKumarMallela01/kosha/internal/markup"
	"github.com/YaswanthKumarMallela01/kosha/internal/model"
	"github.com/YaswanthKumarMallela01/kosha/internal/search"
)

type NoteViewModel struct {
	app        *App
	w, h       int
	links      []string
	activeLink int
	backlinks  []search.SearchEntry
	scrollOff  int
}

func NewNoteViewModel(app *App) NoteViewModel {
	return NoteViewModel{
		app: app,
	}
}

func (m *NoteViewModel) SetSize(w, h int) {
	m.w = w
	m.h = h
}

func (m *NoteViewModel) Refresh() {
	if m.app == nil || m.app.currentNote == nil {
		return
	}

	m.links = model.ExtractLinks(m.app.currentNote.Body)
	m.activeLink = 0
	m.scrollOff = 0

	if m.app.searchIndex != nil {
		m.backlinks = m.app.searchIndex.FindBacklinks(m.app.currentNote.Title)
	}
}

func (m NoteViewModel) Update(msg tea.Msg) (NoteViewModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "j", "down":
			m.scrollOff++
		case "k", "up":
			if m.scrollOff > 0 {
				m.scrollOff--
			}
		case "pgdown":
			m.scrollOff += 10
		case "pgup":
			m.scrollOff -= 10
			if m.scrollOff < 0 {
				m.scrollOff = 0
			}
		case "tab":
			if len(m.links) > 0 {
				m.activeLink = (m.activeLink + 1) % len(m.links)
				m.app.statusMsg = fmt.Sprintf("Selected link: [[%s]]", m.links[m.activeLink])
			}
		case "shift+tab":
			if len(m.links) > 0 {
				m.activeLink--
				if m.activeLink < 0 {
					m.activeLink = len(m.links) - 1
				}
				m.app.statusMsg = fmt.Sprintf("Selected link: [[%s]]", m.links[m.activeLink])
			}
		case "enter":
			if len(m.links) > 0 && m.activeLink < len(m.links) {
				target := m.links[m.activeLink]
				bookSlug := ""
				chapID := ""
				if m.app.currentBook != nil {
					bookSlug = m.app.currentBook.Slug
				}
				if m.app.currentChapter != nil {
					chapID = m.app.currentChapter.ID
				}
				ref, err := m.app.library.FindNoteByTitle(target, chapID, bookSlug)
				if err == nil && ref != nil {
					m.app.jumpToNote(ref)
					return m, nil
				}
				m.app.statusMsg = fmt.Sprintf("Note '[%s]' not found", target)
			}
		case "e", "E":
			if m.app.currentNote != nil {
				m.app.openEditor(m.app.currentNote)
			}
		case "ctrl+g":
			if m.app.currentNote != nil {
				m.app.triggerAIRefine()
			}
		case "p":
			if m.app.currentNote != nil {
				m.app.currentNote.Pinned = !m.app.currentNote.Pinned
				m.app.currentNote.UpdatedAt = time.Now()
				m.app.library.SaveChapter(m.app.currentBook.Slug, m.app.currentChapter)
				if m.app.currentNote.Pinned {
					m.app.statusMsg = "Note pinned"
				} else {
					m.app.statusMsg = "Note unpinned"
				}
			}
		case "x":
			if m.app.currentNote != nil {
				m.app.exportCurrentNote()
			}
		case "esc", "backspace":
			m.app.popScreen()
		}
	}
	return m, nil
}

func (m NoteViewModel) View() string {
	if m.app == nil || m.app.currentNote == nil {
		return MutedStyle.Render("No note selected")
	}

	note := m.app.currentNote
	viewWidth := m.w - 8
	if viewWidth < 20 {
		viewWidth = 20
	}

	// Title header
	pinStr := ""
	if note.Pinned {
		pinStr = PinMarker.Render()
	}
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorSaffron).
		Width(viewWidth)
	title := titleStyle.Render(pinStr + note.Title)

	// Meta line
	tags := model.ExtractTags(note.Body)
	tagStr := ""
	if len(tags) > 0 {
		tagStr = "  Tags: " + PinkStyle.Render(strings.Join(tags, " "))
	}
	metaStr := MutedStyle.Render(fmt.Sprintf("Created: %s | Updated: %s%s",
		formatTime(note.CreatedAt),
		formatTime(note.UpdatedAt),
		tagStr,
	))

	divider := MutedStyle.Render(strings.Repeat("─", viewWidth))

	// Rendered body
	renderedBody := markup.RenderToTerminal(note.Body, viewWidth, m.app.theme)

	var sb strings.Builder
	sb.WriteString(title + "\n")
	sb.WriteString(metaStr + "\n")
	sb.WriteString(divider + "\n\n")
	sb.WriteString(renderedBody)

	// Backlinks section
	if len(m.backlinks) > 0 {
		sb.WriteString("\n\n" + divider + "\n")
		sb.WriteString(SaffronStyle.Render("✦ Backlinks (notes mentioning this note):") + "\n")
		for _, bl := range m.backlinks {
			if bl.NoteID != note.ID {
				sb.WriteString(PinkStyle.Render(fmt.Sprintf("  • %s (%s › %s)\n", bl.NoteTitle, bl.BookName, bl.ChapterTitle)))
			}
		}
	}

	// Handle scrolling
	allLines := strings.Split(sb.String(), "\n")
	visibleHeight := m.h - 6
	if visibleHeight < 5 {
		visibleHeight = 5
	}

	if m.scrollOff > len(allLines)-visibleHeight {
		m.scrollOff = len(allLines) - visibleHeight
	}
	if m.scrollOff < 0 {
		m.scrollOff = 0
	}

	endLine := m.scrollOff + visibleHeight
	if endLine > len(allLines) {
		endLine = len(allLines)
	}

	return strings.Join(allLines[m.scrollOff:endLine], "\n")
}
