package editor

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestSelectionPreservedOnShiftF(t *testing.T) {
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

	// Press Shift+F (represented as 'F' in standard terminals or "shift+f")
	m, _ := ed.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'F'}})
	edModel := m.(Model)

	if !edModel.showFormatPane {
		t.Error("Expected format pane to open on Shift+F ('F')")
	}
	if !edModel.buffer.HasSelection() {
		t.Error("Expected selection to remain intact after pressing Shift+F")
	}
	if edModel.buffer.SelectedText() != selected {
		t.Errorf("Selected text was corrupted by Shift+F: %q", edModel.buffer.SelectedText())
	}

	// Close format pane with Shift+F again
	mClose, _ := edModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	edModel = mClose.(Model)
	if edModel.showFormatPane {
		t.Error("Expected format pane to close when toggled with 'f'")
	}
	if !edModel.buffer.HasSelection() {
		t.Error("Expected selection to remain intact after closing format pane")
	}

	// Open format pane with "shift+f" string
	mOpen2, _ := edModel.Update(tea.KeyMsg(tea.Key{Type: tea.KeyRunes, Runes: []rune("shift+f")}))
	edModel = mOpen2.(Model)
	if !edModel.showFormatPane {
		t.Error("Expected format pane to open on 'shift+f'")
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

func TestCtrlFReservedAndPreservesSelection(t *testing.T) {
	orig := "Important code to preserve"
	ed := New(orig, nil)
	ed.Focus()

	ed.buffer.SelectAll()
	if !ed.buffer.HasSelection() {
		t.Fatal("Expected buffer to have selection")
	}

	// Press Ctrl+F - should NOT open format pane and MUST NOT delete selection
	m, _ := ed.Update(tea.KeyMsg{Type: tea.KeyCtrlF})
	edModel := m.(Model)

	if edModel.showFormatPane {
		t.Error("Expected format pane to NOT open on Ctrl+F (reserved for search)")
	}
	if !edModel.buffer.HasSelection() {
		t.Error("Expected selection to remain intact after pressing Ctrl+F")
	}
	if edModel.Content() != orig {
		t.Errorf("Content was modified by Ctrl+F: %q", edModel.Content())
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

func TestUndoAfterFormatting(t *testing.T) {
	orig := "Selected paragraph to format"
	ed := New(orig, nil)
	ed.Focus()

	ed.buffer.SelectAll()
	if !ed.buffer.HasSelection() {
		t.Fatal("Expected selection")
	}

	// Press Ctrl+B to bold
	m, _ := ed.Update(tea.KeyMsg{Type: tea.KeyCtrlB})
	edModel := m.(Model)

	if edModel.Content() != "**Selected paragraph to format**" {
		t.Fatalf("Expected bolded content, got %q", edModel.Content())
	}

	// Press Ctrl+Z to undo
	mUndo, _ := edModel.Update(tea.KeyMsg{Type: tea.KeyCtrlZ})
	edModelUndo := mUndo.(Model)

	if edModelUndo.Content() != orig {
		t.Fatalf("Expected content restored to %q on single Ctrl+Z, got %q", orig, edModelUndo.Content())
	}
}

func TestTabIndentsSelectionWithoutDeleting(t *testing.T) {
	orig := "Line 1\nLine 2"
	ed := New(orig, nil)
	ed.Focus()

	ed.buffer.SelectAll()
	if !ed.buffer.HasSelection() {
		t.Fatal("Expected selection")
	}

	// Press Tab
	m, _ := ed.Update(tea.KeyMsg{Type: tea.KeyTab})
	edModel := m.(Model)

	expected := "    Line 1\n    Line 2"
	if edModel.Content() != expected {
		t.Fatalf("Expected indented lines %q, got %q", expected, edModel.Content())
	}
	if !edModel.buffer.HasSelection() {
		t.Fatal("Expected selection to be preserved after Tab indentation")
	}

	// Press Shift+Tab to unindent
	mUnindent, _ := edModel.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	edModelUnindent := mUnindent.(Model)

	if edModelUnindent.Content() != orig {
		t.Fatalf("Expected unindented lines restored to %q, got %q", orig, edModelUnindent.Content())
	}
}

func TestWantsImageTrigger(t *testing.T) {
	ed := New("Some text", nil)
	ed.Focus()

	// Alt+I
	m, _ := ed.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}, Alt: true})
	edModel := m.(Model)

	// In bubbletea, alt+i may arrive as string "alt+i"
	m2, _ := edModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	_ = m2

	// Format pane 'p' triggers wantsImage
	ed.buffer.SelectAll()
	mPane, _ := ed.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'F'}})
	edPane := mPane.(Model)
	if !edPane.IsFormatPaneOpen() {
		t.Fatal("Expected format pane to be open")
	}

	mP, _ := edPane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	edP := mP.(Model)
	if !edP.WantsImage() {
		t.Fatal("Expected WantsImage() to be true after pressing 'p' in format pane")
	}
	edP.ClearWantsImage()
	if edP.WantsImage() {
		t.Fatal("Expected WantsImage() to be false after ClearWantsImage()")
	}
}
