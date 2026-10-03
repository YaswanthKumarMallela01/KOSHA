package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

func TestPassphraseModelViewPitchBlackBackground(t *testing.T) {
	lipgloss.SetColorProfile(termenv.TrueColor)
	p := NewPassphraseModel()
	width := 100
	height := 25

	rendered := p.View(width, height)
	lines := strings.Split(rendered, "\n")

	if len(lines) != height {
		t.Fatalf("Expected exactly %d lines, got %d", height, len(lines))
	}

	for i, l := range lines {
		// Each line must contain pitch black background #000000 (48;2;0;0;0m)
		if !strings.Contains(l, "48;2;0;0;0m") {
			t.Errorf("Line %d lacks pitch black background: %q", i, l)
		}
	}
}
