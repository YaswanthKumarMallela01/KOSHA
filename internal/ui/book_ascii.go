package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// renderAsciiBookShape generates an ASCII art representation of a physical book cover
// with the book's title emblazoned directly on the book shape.
func renderAsciiBookShape(title string, width int, isSelected bool) string {
	displayTitle := strings.ToUpper(strings.TrimSpace(title))
	if len(displayTitle) == 0 {
		displayTitle = "UNTITLED BOOK"
	}
	if len(displayTitle) > 28 {
		displayTitle = displayTitle[:25] + "..."
	}

	// Pad title for 30 chars slot on front cover
	padLen := 30 - len(displayTitle)
	if padLen < 0 {
		padLen = 0
	}
	leftPad := padLen / 2
	rightPad := padLen - leftPad
	coverTitle := strings.Repeat(" ", leftPad) + displayTitle + strings.Repeat(" ", rightPad)

	// Pad title for 26 chars angled slot on top spine
	topTitle := displayTitle
	if len(topTitle) > 24 {
		topTitle = topTitle[:21] + "..."
	}
	topPadLen := 26 - len(topTitle)
	if topPadLen < 0 {
		topPadLen = 0
	}
	topTitleFormatted := strings.Repeat(" ", topPadLen/2) + topTitle + strings.Repeat(" ", topPadLen-topPadLen/2)

	borderColor := ColorGlassBorder
	titleColor := ColorBodyText
	accentColor := ColorMutedText

	if isSelected {
		borderColor = ColorSaffron
		titleColor = ColorSaffron
		accentColor = ColorLotusPink
	}

	bStyle := lipgloss.NewStyle().Foreground(borderColor)
	tStyle := lipgloss.NewStyle().Foreground(titleColor).Bold(true)
	aStyle := lipgloss.NewStyle().Foreground(accentColor)

	lines := []string{
		bStyle.Render("       .──────────────────────────────────────────."),
		bStyle.Render("      /   ______________________________________ /|"),
		bStyle.Render("     /   /                                     // |"),
		fmt.Sprintf("    /   /      %s       //  |", tStyle.Render(topTitleFormatted)),
		bStyle.Render("   /   /                                     //   |"),
		bStyle.Render("  /   /=====================================//    |"),
		bStyle.Render(" |   |   ╭───────────────────────────────╮ | |    |"),
		fmt.Sprintf(" |   |   │%s│ | |    |", tStyle.Render("  "+coverTitle+"  ")),
		bStyle.Render(" |   |   │                               │ | |    |"),
		fmt.Sprintf(" |   |   │   %s    │ | |    |", aStyle.Render(" [ K O S H A   B O O K ] ")),
		bStyle.Render(" |   |   │                               │ | |    |"),
		bStyle.Render(" |   |   ╰───────────────────────────────╯ | |    |"),
		bStyle.Render(" |   |   ░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░ | |   /"),
		bStyle.Render(" |   |                                   | |  /"),
		bStyle.Render(" \\   \\___________________________________/ / /"),
		bStyle.Render("  `───────────────────────────────────────' /"),
	}

	return strings.Join(lines, "\n")
}

// renderBookCard combines the ASCII book shape and the detailed information below it
func renderBookCard(title, path, details, date string, width int, isSelected bool) string {
	asciiBook := renderAsciiBookShape(title, width, isSelected)

	cardBorder := ColorGlassBorder
	if isSelected {
		cardBorder = ColorSaffron
	}

	boxWidth := 52
	if width-6 > boxWidth && width-6 < 90 {
		boxWidth = width - 6
	}

	infoBox := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(cardBorder).
		Padding(0, 1).
		Width(boxWidth).
		Render(fmt.Sprintf("%s\n  ├── 📁 Path:         %s\n  ├── 🔐 Vault Files:  %s\n  └── 🕒 Created:      %s",
			lipgloss.NewStyle().Bold(true).Foreground(cardBorder).Render("📚 BOOK INFORMATION: "+title),
			MutedStyle.Render(path),
			lipgloss.NewStyle().Foreground(ColorBodyText).Render(details),
			MutedStyle.Render(date),
		))

	return lipgloss.JoinVertical(lipgloss.Left, asciiBook, infoBox)
}
