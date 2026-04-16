package tui

import "github.com/charmbracelet/lipgloss"

// ─── Color Palette ──────────────────────────────────────────
// A moody cyberpunk-ish palette that feels like a hacker tool.
var (
	ColorPrimary   = lipgloss.Color("#7C3AED") // purple
	ColorSecondary = lipgloss.Color("#06B6D4") // cyan
	ColorSuccess   = lipgloss.Color("#10B981") // emerald
	ColorWarning   = lipgloss.Color("#F59E0B") // amber
	ColorDanger    = lipgloss.Color("#EF4444") // red
	ColorDim       = lipgloss.Color("#6B7280") // gray
	ColorText      = lipgloss.Color("#E5E7EB") // light gray
	ColorBg        = lipgloss.Color("#1F2937") // dark bg
)

// ─── Reusable Styles ────────────────────────────────────────

var (
	// Banner / title bar
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(ColorPrimary).
			Padding(0, 2).
			MarginBottom(1)

	SubtitleStyle = lipgloss.NewStyle().
			Foreground(ColorSecondary).
			Italic(true)

	// Section headers for each step
	StepHeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary).
			Border(lipgloss.NormalBorder(), false, false, true, false).
			BorderForeground(ColorPrimary).
			MarginTop(1).
			MarginBottom(1).
			PaddingBottom(0)

	// Status messages
	SuccessStyle = lipgloss.NewStyle().
			Foreground(ColorSuccess).
			Bold(true)

	WarnStyle = lipgloss.NewStyle().
			Foreground(ColorWarning).
			Bold(true)

	DangerStyle = lipgloss.NewStyle().
			Foreground(ColorDanger).
			Bold(true)

	DimStyle = lipgloss.NewStyle().
			Foreground(ColorDim)

	InfoStyle = lipgloss.NewStyle().
			Foreground(ColorSecondary)

	// The box that wraps warning/danger callouts
	CalloutBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorWarning).
			Padding(0, 1).
			MarginLeft(2)

	DangerBox = lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(ColorDanger).
			Padding(0, 1).
			MarginLeft(2)

	// Audit scorecard
	ScoreBoxPass = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorSuccess).
			Padding(1, 3).
			MarginTop(1)

	ScoreBoxWarn = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorWarning).
			Padding(1, 3).
			MarginTop(1)

	ScoreBoxFail = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorDanger).
			Padding(1, 3).
			MarginTop(1)
)

// ─── Helper Formatters ──────────────────────────────────────

func StatusOK(msg string) string {
	return SuccessStyle.Render("  ✔ ") + msg
}

func StatusFail(msg string) string {
	return DangerStyle.Render("  ✘ ") + msg
}

func StatusSkip(msg string) string {
	return DimStyle.Render("  ⏭ ") + DimStyle.Render(msg)
}

func StatusWarn(msg string) string {
	return WarnStyle.Render("  ⚠ ") + msg
}

func StatusInfo(msg string) string {
	return InfoStyle.Render("  ℹ ") + msg
}

func StepHeader(num int, title string) string {
	icon := []string{"", "📦", "👤", "🔑", "🧱", "🚨", "🧬", "🧹", "📋"}
	emoji := ""
	if num < len(icon) {
		emoji = icon[num] + " "
	}
	return StepHeaderStyle.Render(
		lipgloss.JoinHorizontal(lipgloss.Center,
			DimStyle.Render("step "+string(rune('0'+num))+" ── "),
			emoji+title,
		),
	)
}
