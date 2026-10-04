package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// KeyHint represents a keybinding hint for the footer.
type KeyHint struct {
	Key         string
	Description string
}

func renderHeader(breadcrumb string, width int, locked bool) string {
	lockStr := ""
	if locked {
		lockStr = " 🔒 LOCKED"
	}

	left := lipgloss.NewStyle().
		Background(ColorBackground).
		Foreground(ColorSaffron).
		Bold(true).
		Render("⚡ KOSHA VAULT")

	bcStyle := lipgloss.NewStyle().
		Background(ColorBackground).
		Foreground(ColorBodyText).
		Render(" " + breadcrumb + lockStr)

	topBar := lipgloss.JoinHorizontal(lipgloss.Top, left, bcStyle)
	divider := lipgloss.NewStyle().
		Foreground(ColorGlassBorder).
		Render(strings.Repeat("━", width))

	return topBar + "\n" + divider
}

func renderFooter(hints []KeyHint, status string, width int) string {
	var parts []string
	for _, h := range hints {
		part := fmt.Sprintf("[%s] %s", SaffronStyle.Render(h.Key), lipgloss.NewStyle().Foreground(ColorBodyText).Render(h.Description))
		parts = append(parts, part)
	}

	left := " " + strings.Join(parts, "  •  ")
	right := ""
	if status != "" {
		right = SaffronStyle.Render("✦ " + status + " ")
	}

	padWidth := width - lipgloss.Width(left) - lipgloss.Width(right)
	if padWidth < 0 {
		padWidth = 0
	}

	content := left + strings.Repeat(" ", padWidth) + right
	divider := lipgloss.NewStyle().
		Foreground(ColorGlassBorder).
		Render(strings.Repeat("─", width))

	return divider + "\n" + content
}

func truncateString(s string, max int) string {
	if len(s) > max && max > 3 {
		return s[:max-3] + "..."
	}
	return s
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("2006-01-02 15:04")
}

// renderGlassPanel renders content in a clean, pitch-black container bordered neatly.
func renderGlassPanel(content string, width, height int) string {
	panelWidth := width - 2
	if panelWidth < 20 {
		panelWidth = 20
	}
	panelHeight := height
	if panelHeight < 3 {
		panelHeight = 3
	}

	lines := strings.Split(content, "\n")
	bgStyle := lipgloss.NewStyle().Background(ColorBackground)
	var paddedLines []string
	for i := 0; i < panelHeight; i++ {
		line := ""
		if i < len(lines) {
			line = lines[i]
		}
		w := lipgloss.Width(line)
		trailing := ""
		if w < panelWidth {
			trailing = bgStyle.Render(strings.Repeat(" ", panelWidth-w))
		}
		paddedLines = append(paddedLines, line+trailing)
	}
	return strings.Join(paddedLines, "\n")
}

func renderBackground(width, height int) string {
	return lipgloss.NewStyle().Background(ColorBackground).Width(width).Height(height).Render("")
}

type ListItem struct {
	ID          string
	Type        string // "BOOK", "VAULT", "NOTE", "TAG", "SNAPSHOT"
	Title       string
	Path        string
	Subtitle    string
	Description string
	Pinned      bool
	Badge       string
}

// ListModel provides list navigation state with scrolling.
type ListModel struct {
	Items    []ListItem
	Selected int
	Offset   int
}

func (m *ListModel) Up() {
	if m.Selected > 0 {
		m.Selected--
		if m.Selected < m.Offset {
			m.Offset = m.Selected
		}
	}
}

func (m *ListModel) Down() {
	if m.Selected < len(m.Items)-1 {
		m.Selected++
	}
}

func (m *ListModel) Home() {
	m.Selected = 0
	m.Offset = 0
}

func (m *ListModel) End() {
	if len(m.Items) > 0 {
		m.Selected = len(m.Items) - 1
	}
}

func (m *ListModel) PageUp(pageSize int) {
	m.Selected -= pageSize
	if m.Selected < 0 {
		m.Selected = 0
	}
	if m.Selected < m.Offset {
		m.Offset = m.Selected
	}
}

func (m *ListModel) PageDown(pageSize int) {
	m.Selected += pageSize
	if m.Selected >= len(m.Items) {
		m.Selected = len(m.Items) - 1
	}
}

func (m *ListModel) SelectedItem() *ListItem {
	if m.Selected >= 0 && m.Selected < len(m.Items) {
		return &m.Items[m.Selected]
	}
	return nil
}

// renderList renders items as coding cards with symbols for books, vaults, and notes.
func renderList(items []ListItem, selected, offset int, width, height int) string {
	if len(items) == 0 {
		emptyBox := lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(ColorGlassBorder).
			Padding(1, 2).
			Width(width - 6).
			Render("No items found.\nPress [N] to create a new one, or [?] for help.")
		return "\n" + emptyBox
	}

	cardHeight := 5 // each coding card is ~4 lines + 1 margin
	maxVisible := height / cardHeight
	if maxVisible < 1 {
		maxVisible = 1
	}

	// Adjust offset
	if selected < offset {
		offset = selected
	} else if selected >= offset+maxVisible {
		offset = selected - maxVisible + 1
	}
	if offset < 0 {
		offset = 0
	}

	end := offset + maxVisible
	if end > len(items) {
		end = len(items)
	}

	cardWidth := width - 6
	if cardWidth < 30 {
		cardWidth = 30
	}

	var b strings.Builder

	for i := offset; i < end; i++ {
		item := items[i]
		isSel := (i == selected)

		// Border color: Golden Saffron for selected, dark steel for unselected
		borderColor := ColorGlassBorder
		selectorPrefix := "  "
		if isSel {
			borderColor = ColorSaffron
			selectorPrefix = "▸ "
		}

		// Header symbol and type
		typeLabel := "ITEM"
		switch item.Type {
		case "BOOK":
			typeLabel = "📚 BOOK"
		case "VAULT":
			typeLabel = "🔐 VAULT FILE (.vault)"
		case "NOTE":
			typeLabel = "📝 NOTE"
		case "TAG":
			typeLabel = "🏷️  TAG"
		case "SNAPSHOT":
			typeLabel = "🕒 SNAPSHOT"
		}

		pinMarker := ""
		if item.Pinned {
			pinMarker = " 📌 [PINNED]"
		}

		// Build card inner lines
		line1 := fmt.Sprintf("%sTitle:     %s%s", selectorPrefix, SaffronStyle.Render(item.Title), PinkStyle.Render(pinMarker))

		line2 := ""
		if item.Path != "" {
			line2 = fmt.Sprintf("   ├── 📁 Path:     %s\n", MutedStyle.Render(item.Path))
		}

		line3 := ""
		if item.Subtitle != "" {
			line3 = fmt.Sprintf("   ├── ℹ️  Details:  %s\n", lipgloss.NewStyle().Foreground(ColorBodyText).Render(item.Subtitle))
		}

		line4 := ""
		if item.Description != "" {
			line4 = fmt.Sprintf("   └── 🕒 Date:     %s", MutedStyle.Render(item.Description))
		} else {
			line4 = "   └──"
		}

		cardContent := fmt.Sprintf("── %s ──\n%s\n%s%s%s",
			lipgloss.NewStyle().Bold(true).Foreground(borderColor).Render(typeLabel),
			line1,
			line2,
			line3,
			line4,
		)

		cardStyle := lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(borderColor).
			Padding(0, 1).
			Width(cardWidth)

		b.WriteString(cardStyle.Render(cardContent) + "\n")
	}

	if len(items) > maxVisible {
		scrollInfo := fmt.Sprintf("── [Page: Item %d of %d] ── Use [↑/↓] or [J/K] to scroll ──", selected+1, len(items))
		b.WriteString(MutedStyle.Render(scrollInfo))
	}

	return b.String()
}

func renderHelpOverlay(width, height int) string {
	helpText := `
  ╔════════════════════════════════════════════════════════════════════╗
  ║                 KOSHA KEYBOARD REFERENCE GUIDE                     ║
  ╚════════════════════════════════════════════════════════════════════╝

  NAVIGATION:
    ↑ / ↓ or k / j   Navigate items smoothly
    Enter            Open / View selected item
    Esc / Backspace  Go back to previous screen
    / or Ctrl+K      Global fuzzy search across all vaults
    t                Filter notes by tag
    s                Toggle sort order (newest / oldest)
    Ctrl+L           Lock vault immediately
    ?                Toggle this help screen
    q                Quit application (from library)

  MANAGEMENT:
    n                Create new Book / Chapter Vault / Note
    r                Rename selected item
    d                Delete selected item (with confirmation)
    p                Pin / Unpin note
    x                Export notes to standard Markdown

  EDITOR MODE:
    Arrow keys       Move cursor
    Shift+Arrows     Select text
    Ctrl+Left/Right  Word jump
    Home / End       Start / end of line
    Ctrl+Z / Ctrl+Y  Undo / Redo
    Ctrl+C / X / V   Copy / Cut / Paste
    Ctrl+R           Toggle Edit mode / Preview mode
    Shift+F          Toggle format panel
    Ctrl+F           Find / Search (reserved)
    Ctrl+G           Run Gemini AI grammar & highlight refine
    Ctrl+S           Save snapshot of current vault
    Esc              Save note and exit editor

  AI DIFF REVIEW:
    y / n            Accept / Reject current block
    a / x            Accept all / Reject all blocks
    k                Keep block as-is
    Tab / Shift+Tab  Next / previous block
    Enter            Apply accepted changes
    Esc              Discard AI changes
`
	boxWidth := width - 6
	if boxWidth < 50 {
		boxWidth = 50
	}
	return lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(ColorSaffron).
		Padding(1, 2).
		Width(boxWidth).
		Render(helpText)
}
