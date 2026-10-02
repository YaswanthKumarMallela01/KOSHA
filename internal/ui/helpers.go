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

func renderHeader(title, breadcrumb string, width int, locked bool) string {
	lockStr := ""
	if locked {
		lockStr = " 🔒"
	}
	left := HeaderStyle.Render("कोश Kosha - " + title + lockStr)
	right := BreadcrumbStyle.Render(breadcrumb)
	padWidth := width - lipgloss.Width(left) - lipgloss.Width(right)
	if padWidth < 0 {
		padWidth = 0
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, left, strings.Repeat(" ", padWidth), right)
}

func renderFooter(hints []KeyHint, status string, width int) string {
	var hintStrs []string
	for _, h := range hints {
		hintStrs = append(hintStrs, fmt.Sprintf("%s %s", SaffronStyle.Render(h.Key), MutedStyle.Render(h.Description)))
	}
	left := FooterStyle.Render(strings.Join(hintStrs, " | "))
	right := ""
	if status != "" {
		right = SaffronStyle.Render(status)
	}
	padWidth := width - lipgloss.Width(left) - lipgloss.Width(right)
	if padWidth < 0 {
		padWidth = 0
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, left, strings.Repeat(" ", padWidth), right)
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

// renderGlassPanel renders content inside a styled glass container with frost edge and shadow.
func renderGlassPanel(content string, width, height int) string {
	panelWidth := width - 4
	if panelWidth < 20 {
		panelWidth = 20
	}
	panelHeight := height - 3
	if panelHeight < 5 {
		panelHeight = 5
	}

	frostStyle := lipgloss.NewStyle().
		Background(ColorGlassFill).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(ColorFrostEdge).
		BorderTop(true).
		BorderBottom(true).
		BorderLeft(true).
		BorderRight(true).
		Width(panelWidth).
		Height(panelHeight)

	rendered := frostStyle.Render(content)
	shadow := ShadowStyle.Render(strings.Repeat("▀", panelWidth+2))

	return lipgloss.JoinVertical(lipgloss.Left, rendered, shadow)
}

func renderBackground(width, height int) string {
	return lipgloss.NewStyle().Background(ColorBackground).Width(width).Height(height).Render("")
}

type ListItem struct {
	ID          string
	Title       string
	Subtitle    string
	Description string
	Pinned      bool
	Badge       string
}

// ListModel provides robust list navigation state with scrolling.
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

func renderList(items []ListItem, selected, offset int, width, height int) string {
	if len(items) == 0 {
		return MutedStyle.Render("  (empty list)")
	}

	maxVisible := (height - 2) / 3
	if maxVisible < 1 {
		maxVisible = 5
	}

	// Adjust offset to keep selected visible
	if selected < offset {
		offset = selected
	} else if selected >= offset+maxVisible {
		offset = selected - maxVisible + 1
	}
	if offset < 0 {
		offset = 0
	}

	var b strings.Builder
	end := offset + maxVisible
	if end > len(items) {
		end = len(items)
	}

	itemWidth := width - 8
	if itemWidth < 10 {
		itemWidth = 10
	}

	for i := offset; i < end; i++ {
		item := items[i]
		isSel := (i == selected)

		pinStr := ""
		if item.Pinned {
			pinStr = PinMarker.Render()
		}

		badgeStr := ""
		if item.Badge != "" {
			badgeStr = " " + PinkStyle.Render("["+item.Badge+"]")
		}

		titleLine := pinStr + item.Title + badgeStr
		subLine := item.Subtitle
		if item.Description != "" {
			subLine += "  " + item.Description
		}

		if isSel {
			renderedTitle := SelectedItemStyle.Width(itemWidth).Render("▸ " + titleLine)
			renderedSub := lipgloss.NewStyle().
				Background(ColorGlassBorder).
				Foreground(ColorBodyText).
				Width(itemWidth).
				Render("  " + subLine)
			b.WriteString(renderedTitle + "\n" + renderedSub + "\n\n")
		} else {
			renderedTitle := ListItemStyle.Width(itemWidth).Render("  " + titleLine)
			renderedSub := MutedStyle.Width(itemWidth).Render("    " + subLine)
			b.WriteString(renderedTitle + "\n" + renderedSub + "\n\n")
		}
	}

	if len(items) > maxVisible {
		scrollInfo := fmt.Sprintf(" [%d-%d of %d] ", offset+1, end, len(items))
		b.WriteString(MutedStyle.Render(scrollInfo))
	}

	return b.String()
}

func renderHelpOverlay(width, height int) string {
	helpText := `
  KEYBOARD SHORTCUTS REFERENCE

  Global Navigation:
    ↑/↓ or j/k       Move selection
    Enter            Open / View selected item
    Esc / Backspace  Go back / cancel
    / or Ctrl+K      Global fuzzy search
    t                Filter notes by tag
    s                Toggle sort order (newest/oldest)
    Ctrl+L           Lock vault immediately
    ?                Toggle this help screen
    q                Quit application (from library)

  Management:
    n                Create new book / chapter / note
    r                Rename selected item
    d                Delete selected item (with confirmation)
    p                Pin / unpin selected note
    x                Export item to Markdown

  Editor:
    Arrow keys       Move cursor
    Shift+Arrows     Select text
    Ctrl+Left/Right  Word jump
    Home / End       Start / end of line
    PgUp / PgDn      Page up / page down
    Ctrl+Z / Ctrl+Y  Undo / Redo
    Ctrl+C / X / V   Copy / Cut / Paste
    Ctrl+R           Toggle Edit mode / Preview mode
    Ctrl+F           Toggle format toolbar
    Ctrl+G           Run Gemini AI grammar & highlight refine
    Ctrl+S           Manual snapshot of chapter
    Alt+B / Alt+I    Bold / Italic formatting
    Alt+U / Alt+S    Underline / Strikethrough
    Alt+L / E / R    Left / Center / Right alignment

  Diff Review (AI Refinement):
    y / n            Accept / Reject current block
    a / x            Accept all / Reject all blocks
    k                Keep block as-is (do not reprocess)
    Tab / Shift+Tab  Next / previous diff block
    Enter            Apply accepted changes
    Esc              Discard AI suggestions
`
	return renderGlassPanel(helpText, width, height)
}
