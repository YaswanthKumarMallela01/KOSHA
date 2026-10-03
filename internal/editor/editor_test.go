package editor

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestSelectionPreservedOnCtrlF(t *testing.T) {
	ed := New("First line of notes\nSecond line of notes\nThird line of notes", nil)
	ed.Focus()

	// Select all
	ed.buffer.SelectAll()
	if !ed.buffer.HasSelection() {
		t.Fatal("Expected buffer to have selection")
	}
	selected := ed.buffer.SelectedText()
	if selected != "First line of notes\nSecond line of notes\nThird line of notes" {
		t.Fatalf("Unexpected selected text: %q", selected)
	}

	// Press Ctrl+F
	m, _ := ed.Update(tea.KeyMsg{Type: tea.KeyCtrlF})
	edModel := m.(Model)

	if !edModel.showFormatPane {
		t.Error("Expected format pane to open on Ctrl+F")
	}
	if !edModel.buffer.HasSelection() {
		t.Error("Expected selection to remain intact after pressing Ctrl+F")
	}
	if edModel.buffer.SelectedText() != selected {
		t.Errorf("Selected text was corrupted by Ctrl+F: %q", edModel.buffer.SelectedText())
	}

	// Press 'b' to bold the entire selection
	m2, _ := edModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})
	edModel2 := m2.(Model)

	if edModel2.showFormatPane {
		t.Error("Expected format pane to close after applying format")
	}
	content := edModel2.Content()
	expected := "**First line of notes**\n**Second line of notes**\n**Third line of notes**"
	if content != expected {
		t.Errorf("Expected formatted content:\n%q\nGot:\n%q", expected, content)
	}
}

func TestSelectionNotDeletedByCtrl(t *testing.T) {
	ed := New("Hello World", nil)
	ed.Focus()

	ed.buffer.SelectAll()
	if !ed.buffer.HasSelection() {
		t.Fatal("Expected selection")
	}

	// Send an unhandled Ctrl key (e.g. string "ctrl+k")
	m, _ := ed.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{11}})
	edModel := m.(Model)

	if !edModel.buffer.HasSelection() {
		t.Error("Expected selection to be preserved on unhandled Ctrl key")
	}
	if edModel.Content() != "Hello World" {
		t.Errorf("Content was modified on Ctrl key: %q", edModel.Content())
	}
}
