package ui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/YaswanthKumarMallela01/kosha/internal/model"
)

type BookModel struct {
	app      *App
	chapters []*model.Chapter
	selected int
	w, h     int
}

func NewBookModel(app *App) BookModel {
	return BookModel{
		app:      app,
		selected: 0,
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
		m.chapters = nil
		return
	}

	model.SortChapters(chapters, m.app.ascending)
	m.chapters = chapters

	if m.selected >= len(m.chapters) {
		if len(m.chapters) > 0 {
			m.selected = len(m.chapters) - 1
		} else {
			m.selected = 0
		}
	}
}

func (m *BookModel) SelectedChapter() *model.Chapter {
	if len(m.chapters) == 0 || m.selected < 0 || m.selected >= len(m.chapters) {
		return nil
	}
	return m.chapters[m.selected]
}

func (m BookModel) Update(msg tea.Msg) (BookModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "left", "h", "up", "k":
			if m.selected > 0 {
				m.selected--
			} else if len(m.chapters) > 0 {
				m.selected = len(m.chapters) - 1
			}
		case "right", "l", "down", "j":
			if m.selected < len(m.chapters)-1 {
				m.selected++
			} else {
				m.selected = 0
			}
		case "home":
			m.selected = 0
		case "end":
			if len(m.chapters) > 0 {
				m.selected = len(m.chapters) - 1
			}
		case "enter":
			// Open Read Mode to read formatted text peacefully
			if sel := m.SelectedChapter(); sel != nil {
				m.app.currentChapter = sel
				var targetNote *model.Note
				if len(sel.Notes) > 0 {
					targetNote = sel.Notes[0]
				} else {
					targetNote = &model.Note{
						ID:        model.NewID(),
						Title:     sel.Title,
						Body:      "",
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					}
					_ = m.app.library.AddNote(m.app.currentBook.Slug, sel.ID, targetNote)
				}
				m.app.currentNote = targetNote
				m.app.noteView.Refresh()
				m.app.pushScreen(ScreenNoteView)
				return m, nil
			}
		case "e", "E":
			// Open Edit Mode directly
			if sel := m.SelectedChapter(); sel != nil {
				m.app.currentChapter = sel
				var targetNote *model.Note
				if len(sel.Notes) > 0 {
					targetNote = sel.Notes[0]
				} else {
					targetNote = &model.Note{
						ID:        model.NewID(),
						Title:     sel.Title,
						Body:      "",
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					}
					_ = m.app.library.AddNote(m.app.currentBook.Slug, sel.ID, targetNote)
				}
				m.app.currentNote = targetNote
				m.app.openEditor(targetNote)
				return m, nil
			}
		case "ctrl+n", "n", "N":
			m.app.inputOverlay.Open("Create New Vault Section in "+m.app.currentBook.DisplayName+" (e.g. Memory Management):", "", "new_vault", "")
			m.app.pushScreen(ScreenNewItem)
		case "r", "R":
			if sel := m.SelectedChapter(); sel != nil {
				m.app.inputOverlay.Open("Rename Vault Section:", sel.Title, "rename_chapter", sel.ID)
				m.app.pushScreen(ScreenRename)
			}
		case "d", "D":
			if sel := m.SelectedChapter(); sel != nil {
				m.app.confirmDialog.Open("Delete vault '"+sel.Title+"' and its encrypted notes?", "delete_chapter", sel.ID)
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
			if sel := m.SelectedChapter(); sel != nil {
				m.app.exportChapter(m.app.currentBook.Slug, sel.Title)
			}
		case "esc", "backspace":
			m.app.currentChapter = nil
			m.app.currentBook = nil
			m.app.libraryList.Refresh()
			m.app.popScreen()
			return m, nil
		}
	}
	return m, nil
}

func (m BookModel) View() string {
	if len(m.chapters) == 0 {
		emptyCard := renderSmallVaultCard("NO VAULTS", 0, false)
		info := lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(ColorGlassBorder).
			Padding(1, 2).
			Width(56).
			Render(fmt.Sprintf("No vault sections in '%s' yet!\n\nPress [Ctrl+N] or [N] to create a vault (e.g. \"Chapter 1: Overview\").\nYou can then write notes inside it immediately.", m.app.currentBook.DisplayName))
		return lipgloss.JoinVertical(lipgloss.Left, emptyCard, "\n", info)
	}

	selIdx := m.selected
	if selIdx < 0 || selIdx >= len(m.chapters) {
		selIdx = 0
	}

	// 1. Render all vaults side-wise using compact ASCII vault art
	cards := make([]string, len(m.chapters))
	for i, ch := range m.chapters {
		wordCount := 0
		for _, n := range ch.Notes {
			wordCount += len(strings.Fields(n.Body))
		}
		cards[i] = renderSmallVaultCard(ch.Title, wordCount, i == selIdx)
	}

	cardRow, navIndicator := renderHorizontalCards(cards, selIdx, m.w)

	// 2. Info box & note preview for selected vault
	selChap := m.chapters[selIdx]
	totalWords := 0
	var previewText string
	for _, n := range selChap.Notes {
		totalWords += len(strings.Fields(n.Body))
		if previewText == "" && strings.TrimSpace(n.Body) != "" {
			previewText = strings.TrimSpace(n.Body)
		}
	}

	if previewText == "" {
		previewText = "Vault is empty. Press [Enter] or [E] to start writing notes in this vault!"
	} else {
		// Truncate to first 3 lines
		pLines := strings.Split(previewText, "\n")
		if len(pLines) > 3 {
			pLines = append(pLines[:3], "...")
		}
		previewText = strings.Join(pLines, "\n")
	}

	boxWidth := 58
	if m.w-8 > boxWidth && m.w-8 < 95 {
		boxWidth = m.w - 8
	}

	infoBox := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(ColorSaffron).
		Padding(0, 1).
		Width(boxWidth).
		Render(fmt.Sprintf("%s\n  ├── 🔐 File:    %s.vault\n  ├── 📊 Stats:   %d word(s) written\n  ├── 🕒 Created: %s\n  └── 📝 Notes Preview:\n%s",
			lipgloss.NewStyle().Bold(true).Foreground(ColorSaffron).Render("🔐 VAULT CHAPTER: "+selChap.Title),
			selChap.ID,
			totalWords,
			MutedStyle.Render(formatTime(selChap.CreatedAt)),
			lipgloss.NewStyle().Foreground(ColorBodyText).PaddingLeft(4).Render(previewText),
		))

	return lipgloss.JoinVertical(lipgloss.Left,
		cardRow,
		"\n",
		MutedStyle.Render(navIndicator),
		"\n",
		infoBox,
	)
}
