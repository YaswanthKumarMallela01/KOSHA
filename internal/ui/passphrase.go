package ui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/YaswanthKumarMallela01/kosha/internal/crypto"
)

type PassphraseModel struct {
	app      *App
	input    textinput.Model
	attempts int
	errMsg   string
}

func NewPassphraseModel() PassphraseModel {
	ti := textinput.New()
	ti.Placeholder = "Enter master passphrase..."
	ti.EchoMode = textinput.EchoPassword
	ti.EchoCharacter = '•'
	ti.Focus()
	ti.CharLimit = 100
	ti.Width = 35

	return PassphraseModel{
		input: ti,
	}
}

func (m *PassphraseModel) Reset() {
	m.input.Reset()
	m.input.Focus()
	m.errMsg = ""
}

func (m PassphraseModel) Update(msg tea.Msg) (PassphraseModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			pass := m.input.Value()
			if pass == "" {
				return m, nil
			}

			if m.app == nil || m.app.config == nil {
				m.errMsg = "Configuration missing"
				return m, nil
			}

			// Validate passphrase
			masterKey := crypto.DeriveMasterKey([]byte(pass), []byte(m.app.config.MasterSalt), m.app.config.ArgonParams)
			if crypto.ValidateKeyCheck([]byte(m.app.config.KeyCheck), masterKey) {
				// Success!
				m.app.library.MasterKey = masterKey
				_ = m.app.library.LoadBooks()
				m.app.buildSearchIndex()
				m.app.locked = false
				m.app.screen = ScreenLibrary
				m.app.libraryList.Refresh()
				m.Reset()
				m.attempts = 0
				m.app.statusMsg = "Vault unlocked"
				return m, nil
			}

			// Failed
			crypto.ZeroBytes(masterKey)
			m.attempts++
			remaining := 5 - m.attempts
			if remaining <= 0 {
				m.errMsg = "Too many failed attempts. Vault locked."
			} else {
				m.errMsg = fmt.Sprintf("Incorrect passphrase. %d attempt(s) remaining.", remaining)
			}
			m.input.Reset()
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m PassphraseModel) View(width, height int) string {
	boxWidth := 46
	boxStyle := lipgloss.NewStyle().
		Background(ColorGlassFill).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(ColorGlassBorder).
		Padding(2, 4).
		Width(boxWidth)

	logo := SaffronStyle.Render("       कोश  K O S H A       \n   Encrypted Notes Vault    ")

	errLine := ""
	if m.errMsg != "" {
		errLine = ErrorStyle.Render(m.errMsg) + "\n\n"
	}

	content := fmt.Sprintf("%s\n\n%s%s\n%s\n\n%s",
		logo,
		errLine,
		MutedStyle.Render("Vault is locked. Enter your master passphrase:"),
		m.input.View(),
		MutedStyle.Render("[Enter] Unlock  [Ctrl+C] Exit"),
	)

	renderedBox := boxStyle.Render(content)
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, renderedBox)
}
