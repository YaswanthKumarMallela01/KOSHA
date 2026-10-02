package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

// fitDisplayString truncates or pads a string to fit exact target display width
func fitDisplayString(s string, targetWidth int) string {
	s = strings.TrimSpace(s)
	currWidth := runewidth.StringWidth(s)
	if currWidth == targetWidth {
		return s
	}
	if currWidth < targetWidth {
		return s + strings.Repeat(" ", targetWidth-currWidth)
	}

	// Needs truncation
	var result strings.Builder
	w := 0
	target := targetWidth - 3
	if target < 1 {
		target = 1
	}
	for _, r := range s {
		rw := runewidth.RuneWidth(r)
		if w+rw > target {
			break
		}
		result.WriteRune(r)
		w += rw
	}
	result.WriteString("...")
	w += 3
	if w < targetWidth {
		result.WriteString(strings.Repeat(" ", targetWidth-w))
	}
	return result.String()
}

// renderSmallBookCard renders a compact 24x7 3D ASCII book shape.
func renderSmallBookCard(title string, vaultCount int, isSelected bool) string {
	bColor := ColorGlassBorder
	tColor := ColorBodyText
	subColor := ColorMutedText
	badgeColor := ColorMutedText

	if isSelected {
		bColor = ColorSaffron
		tColor = ColorSaffron
		subColor = ColorBodyText
		badgeColor = ColorLotusPink
	}

	bStyle := lipgloss.NewStyle().Foreground(bColor)
	tStyle := lipgloss.NewStyle().Foreground(tColor).Bold(true)
	subStyle := lipgloss.NewStyle().Foreground(subColor)
	badgeStyle := lipgloss.NewStyle().Foreground(badgeColor).Bold(true)

	tStr := fitDisplayString(strings.ToUpper(title), 16)
	subText := fmt.Sprintf("%d Vaults", vaultCount)
	if vaultCount == 1 {
		subText = "1 Vault"
	} else if vaultCount == 0 {
		subText = "Empty Vault"
	}
	sStr := fitDisplayString(subText, 16)

	lines := []string{
		bStyle.Render("   .──────────────────. "),
		fmt.Sprintf("  %s %s          %s", bStyle.Render("/"), badgeStyle.Render("📚 BOOK"), bStyle.Render("/│")),
		bStyle.Render(" ┌──────────────────┐  │"),
		fmt.Sprintf(" %s %s %s", bStyle.Render("│"), tStyle.Render(tStr), bStyle.Render("│  │")),
		fmt.Sprintf(" %s %s %s", bStyle.Render("│"), subStyle.Render(sStr), bStyle.Render("│  │")),
		bStyle.Render(" └──────────────────┘ / "),
		bStyle.Render("  '──────────────────'  "),
	}

	return strings.Join(lines, "\n")
}

// renderSmallVaultCard renders a compact 24x7 3D ASCII vault shape (matching book size).
func renderSmallVaultCard(title string, wordCount int, isSelected bool) string {
	bColor := ColorGlassBorder
	tColor := ColorBodyText
	subColor := ColorMutedText
	badgeColor := ColorMutedText

	if isSelected {
		bColor = ColorSaffron
		tColor = ColorSaffron
		subColor = ColorBodyText
		badgeColor = ColorLotusPink
	}

	bStyle := lipgloss.NewStyle().Foreground(bColor)
	tStyle := lipgloss.NewStyle().Foreground(tColor).Bold(true)
	subStyle := lipgloss.NewStyle().Foreground(subColor)
	badgeStyle := lipgloss.NewStyle().Foreground(badgeColor).Bold(true)

	tStr := fitDisplayString(strings.ToUpper(title), 16)
	var subText string
	if wordCount > 0 {
		subText = fmt.Sprintf("%d words", wordCount)
	} else {
		subText = "Encrypted • New"
	}
	sStr := fitDisplayString(subText, 16)

	lines := []string{
		bStyle.Render("   .──────────────────. "),
		fmt.Sprintf("  %s %s         %s", bStyle.Render("/"), badgeStyle.Render("🔐 VAULT"), bStyle.Render("/│")),
		bStyle.Render(" ┌──────────────────┐  │"),
		fmt.Sprintf(" %s %s %s", bStyle.Render("│"), tStyle.Render(tStr), bStyle.Render("│  │")),
		fmt.Sprintf(" %s %s %s", bStyle.Render("│"), subStyle.Render(sStr), bStyle.Render("│  │")),
		bStyle.Render(" └──────────────────┘ / "),
		bStyle.Render("  '──────────────────'  "),
	}

	return strings.Join(lines, "\n")
}

// renderHorizontalCards arranges ASCII cards side-wise with sliding window for screen width.
func renderHorizontalCards(cards []string, selectedIdx int, termWidth int) (string, string) {
	if len(cards) == 0 {
		return "", ""
	}

	cardWidth := 25
	gap := 2
	maxFit := (termWidth - 6) / (cardWidth + gap)
	if maxFit < 1 {
		maxFit = 1
	}
	if maxFit > 4 {
		maxFit = 4
	}

	total := len(cards)
	if selectedIdx < 0 {
		selectedIdx = 0
	}
	if selectedIdx >= total {
		selectedIdx = total - 1
	}

	startIdx := 0
	if selectedIdx >= maxFit {
		startIdx = selectedIdx - maxFit + 1
	}
	endIdx := startIdx + maxFit
	if endIdx > total {
		endIdx = total
		startIdx = endIdx - maxFit
		if startIdx < 0 {
			startIdx = 0
		}
	}

	visibleCards := cards[startIdx:endIdx]
	var cardRow string
	for i, c := range visibleCards {
		if i == 0 {
			cardRow = c
		} else {
			cardRow = lipgloss.JoinHorizontal(lipgloss.Top, cardRow, "  ", c)
		}
	}

	// Nav pagination indicator
	hasLeft := startIdx > 0
	hasRight := endIdx < total

	leftArrow := "   "
	if hasLeft {
		leftArrow = "◀◀ "
	}
	rightArrow := "   "
	if hasRight {
		rightArrow = " ▶▶"
	}

	navIndicator := fmt.Sprintf("%s[Item %d of %d]%s  •  Use [←/→] or [↑/↓] to browse", leftArrow, selectedIdx+1, total, rightArrow)

	return cardRow, navIndicator
}
