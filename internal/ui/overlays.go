package ui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type InputOverlayModel struct {
	Prompt   string
	Action   string // "new_book", "new_chapter", "new_note", "rename"
	TargetID string
	Input    textinput.Model
	Active   bool
	w, h     int
}

func NewInputOverlayModel() InputOverlayModel {
	ti := textinput.New()
	ti.Focus()
	ti.CharLimit = 100
	ti.Width = 40
	return InputOverlayModel{
		Input: ti,
	}
}

func (m *InputOverlayModel) Open(prompt, defaultValue, action, targetID string) {
	m.Prompt = prompt
	m.Action = action
	m.TargetID = targetID
	m.Input.SetValue(defaultValue)
	m.Input.Focus()
	m.Active = true
}

func (m *InputOverlayModel) Close() {
	m.Active = false
	m.Input.Reset()
}

func (m InputOverlayModel) Update(msg tea.Msg) (InputOverlayModel, tea.Cmd) {
	var cmd tea.Cmd
	m.Input, cmd = m.Input.Update(msg)
	return m, cmd
}

func (m InputOverlayModel) View() string {
	boxStyle := lipgloss.NewStyle().
		Background(ColorGlassFill).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(ColorSaffron).
		Padding(1, 3).
		Width(50)

	content := fmt.Sprintf("%s\n\n%s\n\n%s",
		SaffronStyle.Render(m.Prompt),
		m.Input.View(),
		MutedStyle.Render("[Enter] Confirm  [Esc] Cancel"),
	)

	return boxStyle.Render(content)
}

type ConfirmDialogModel struct {
	Message  string
	Action   string // "delete_book", "delete_chapter", "delete_note", "quit"
	TargetID string
	Active   bool
}

func NewConfirmDialogModel() ConfirmDialogModel {
	return ConfirmDialogModel{}
}

func (m *ConfirmDialogModel) Open(message, action, targetID string) {
	m.Message = message
	m.Action = action
	m.TargetID = targetID
	m.Active = true
}

func (m *ConfirmDialogModel) Close() {
	m.Active = false
}

func (m ConfirmDialogModel) Update(msg tea.Msg) (ConfirmDialogModel, tea.Cmd) {
	return m, nil
}

func (m ConfirmDialogModel) View() string {
	boxStyle := lipgloss.NewStyle().
		Background(ColorGlassFill).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(ColorError).
		Padding(1, 3).
		Width(50)

	content := fmt.Sprintf("%s\n\n%s\n\n%s",
		ErrorStyle.Render("CONFIRM ACTION"),
		m.Message,
		MutedStyle.Render("[y] Yes  [n / Esc] No"),
	)

	return boxStyle.Render(content)
}
