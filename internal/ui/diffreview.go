package ui

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/YaswanthKumarMallela01/kosha/internal/ai"
	"github.com/YaswanthKumarMallela01/kosha/internal/model"
)

type DiffReviewModel struct {
	app       *App
	blocks    []ai.DiffBlock
	activeIdx int
	w, h      int
}

func NewDiffReviewModel(app *App) DiffReviewModel {
	return DiffReviewModel{
		app: app,
	}
}

func (m *DiffReviewModel) SetBlocks(blocks []ai.DiffBlock) {
	m.blocks = blocks
	m.activeIdx = 0
}

func (m DiffReviewModel) Update(msg tea.Msg) (DiffReviewModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.app.popScreen()
			return m, nil
		case "tab", "j", "down":
			if len(m.blocks) > 0 {
				m.activeIdx = (m.activeIdx + 1) % len(m.blocks)
			}
		case "shift+tab", "up":
			if len(m.blocks) > 0 {
				m.activeIdx--
				if m.activeIdx < 0 {
					m.activeIdx = len(m.blocks) - 1
				}
			}
		case "y":
			if len(m.blocks) > 0 {
				m.blocks[m.activeIdx].Accepted = true
				m.blocks[m.activeIdx].Rejected = false
				m.blocks[m.activeIdx].KeepAsIs = false
				if m.activeIdx < len(m.blocks)-1 {
					m.activeIdx++
				}
			}
		case "n":
			if len(m.blocks) > 0 {
				m.blocks[m.activeIdx].Accepted = false
				m.blocks[m.activeIdx].Rejected = true
				m.blocks[m.activeIdx].KeepAsIs = false
				if m.activeIdx < len(m.blocks)-1 {
					m.activeIdx++
				}
			}
		case "k": // Keep as is (mark processed with original text hash)
			if len(m.blocks) > 0 {
				m.blocks[m.activeIdx].Accepted = false
				m.blocks[m.activeIdx].Rejected = false
				m.blocks[m.activeIdx].KeepAsIs = true
				if m.activeIdx < len(m.blocks)-1 {
					m.activeIdx++
				}
			}
		case "a": // Accept all
			for i := range m.blocks {
				m.blocks[i].Accepted = true
				m.blocks[i].Rejected = false
				m.blocks[i].KeepAsIs = false
			}
			m.app.statusMsg = "All blocks accepted"
		case "x": // Reject all
			for i := range m.blocks {
				m.blocks[i].Accepted = false
				m.blocks[i].Rejected = true
				m.blocks[i].KeepAsIs = false
			}
			m.app.statusMsg = "All blocks rejected"
		case "enter":
			m.applyChanges()
			m.app.popScreen()
			return m, nil
		}
	}
	return m, nil
}

func (m *DiffReviewModel) applyChanges() {
	if m.app == nil || m.app.currentNote == nil || m.app.currentChapter == nil {
		return
	}

	note := m.app.currentNote
	origBlocks := model.SplitBlocks(note.Body)

	if m.app.currentChapter.ProcessedHashes == nil {
		m.app.currentChapter.ProcessedHashes = make(map[string]bool)
	}

	appliedCount := 0
	// Build map of accepted or kept changes
	for _, db := range m.blocks {
		if db.Accepted {
			// Find block and replace text
			for i := range origBlocks {
				if origBlocks[i].ID == db.ID {
					origBlocks[i].Text = db.Refined
					newHash := model.HashBlock(db.Refined)
					m.app.currentChapter.ProcessedHashes[newHash] = true
					appliedCount++
					break
				}
			}
		} else if db.KeepAsIs {
			origHash := model.HashBlock(db.Original)
			m.app.currentChapter.ProcessedHashes[origHash] = true
		}
	}

	// Merge blocks back into note body
	note.Body = model.MergeBlocks(origBlocks)
	note.UpdatedAt = time.Now()

	// Update editor content
	m.app.editorModel.SetContent(note.Body)

	// Save to encrypted vault
	if err := m.app.library.SaveChapter(m.app.currentBook.Slug, m.app.currentChapter); err != nil {
		m.app.statusMsg = fmt.Sprintf("Error saving: %v", err)
	} else {
		m.app.statusMsg = fmt.Sprintf("Applied %d AI refinement(s) and saved", appliedCount)
	}

	m.app.buildSearchIndex()
}

func (m DiffReviewModel) View() string {
	if len(m.blocks) == 0 {
		return MutedStyle.Render("No changed blocks to review.")
	}

	block := m.blocks[m.activeIdx]
	viewWidth := m.w - 8
	if viewWidth < 30 {
		viewWidth = 30
	}

	// Header
	statusStr := "Pending"
	if block.Accepted {
		statusStr = SaffronStyle.Render("✓ Accepted")
	} else if block.Rejected {
		statusStr = ErrorStyle.Render("✗ Rejected")
	} else if block.KeepAsIs {
		statusStr = PinkStyle.Render("Keep As-Is")
	}

	heavyWarn := ""
	if block.HeavilyChanged {
		heavyWarn = "  " + ErrorStyle.Render("⚠ Heavily Changed (>35% words)")
	}

	headerText := fmt.Sprintf("Block %d of %d  [%s]%s",
		m.activeIdx+1, len(m.blocks), statusStr, heavyWarn)

	// Original Box (coral/muted rose)
	origStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(ColorError).
		Padding(0, 1).
		Width(viewWidth).
		Foreground(ColorBodyText)

	origBox := origStyle.Render(
		ErrorStyle.Render("ORIGINAL (Removed):") + "\n\n" + block.Original,
	)

	// Refined Box (saffron)
	refStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(ColorSaffron).
		Padding(0, 1).
		Width(viewWidth).
		Foreground(ColorBodyText)

	refBox := refStyle.Render(
		SaffronStyle.Render("REFINED (Added):") + "\n\n" + block.Refined,
	)

	// Key hints
	controls := MutedStyle.Render(
		"[y] Accept  [n] Reject  [a] Accept All  [x] Reject All  [k] Keep As-Is\n[Tab] Next  [Enter] Apply Changes  [Esc] Cancel",
	)

	return lipgloss.JoinVertical(lipgloss.Left,
		SaffronStyle.Render(headerText),
		"\n",
		origBox,
		"\n",
		refBox,
		"\n",
		controls,
	)
}
