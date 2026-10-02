package ui

import "github.com/charmbracelet/lipgloss"

// Colors - Pitch Black Developer & Saffron Theme. Strictly NO GREEN.
const (
	ColorBackground  = lipgloss.Color("#000000") // Pitch black
	ColorGlassFill   = lipgloss.Color("#000000") // Deep pitch black
	ColorGlassBorder = lipgloss.Color("#262933") // Clean dark border
	ColorFrostEdge   = lipgloss.Color("#3A4050") // Crisp border highlight
	ColorBodyText    = lipgloss.Color("#E2E4E9") // Crisp readable parchment/white
	ColorSaffron     = lipgloss.Color("#F5A623") // Golden saffron highlight
	ColorLotusPink   = lipgloss.Color("#E58BB0") // Lotus pink selection/cursor
	ColorMutedText   = lipgloss.Color("#6C7385") // Muted metadata
	ColorError       = lipgloss.Color("#E5646E") // Soft coral error
	ColorShadow      = lipgloss.Color("#000000") // Pitch black
	ColorGradientTop = lipgloss.Color("#000000")
	ColorGradientBot = lipgloss.Color("#000000")
)

// Styles
var (
	GlassPanel = lipgloss.NewStyle().
			Background(ColorBackground).
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(ColorGlassBorder).
			Foreground(ColorBodyText).
			Padding(0, 1)

	HeaderStyle = lipgloss.NewStyle().
			Background(ColorBackground).
			Foreground(ColorSaffron).
			Bold(true).
			Padding(0, 1)

	FooterStyle = lipgloss.NewStyle().
			Background(ColorBackground).
			Foreground(ColorMutedText).
			Padding(0, 1)

	ListItemStyle = lipgloss.NewStyle().
			Foreground(ColorBodyText).
			Padding(0, 1)

	SelectedItemStyle = lipgloss.NewStyle().
			Foreground(ColorSaffron).
			Bold(true).
			Padding(0, 1)

	MutedStyle = lipgloss.NewStyle().
			Foreground(ColorMutedText)

	ErrorStyle = lipgloss.NewStyle().
			Foreground(ColorError).
			Bold(true)

	SaffronStyle = lipgloss.NewStyle().
			Foreground(ColorSaffron).
			Bold(true)

	PinkStyle = lipgloss.NewStyle().
			Foreground(ColorLotusPink)

	ShadowStyle = lipgloss.NewStyle().
			Foreground(ColorShadow)

	BreadcrumbStyle = lipgloss.NewStyle().
			Foreground(ColorMutedText)

	PinMarker = lipgloss.NewStyle().
			Foreground(ColorSaffron).
			SetString("📌 ")

	TagStyle = lipgloss.NewStyle().
			Foreground(ColorLotusPink)
)
