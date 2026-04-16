package tui

import (
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

// CharmTheme returns a custom huh theme matching our palette.
func CharmTheme() *huh.Theme {
	t := huh.ThemeBase()

	t.Focused.Title = t.Focused.Title.Foreground(ColorSecondary).Bold(true)
	t.Focused.Description = t.Focused.Description.Foreground(ColorDim)
	t.Focused.Base = t.Focused.Base.BorderForeground(ColorPrimary)
	t.Focused.SelectedOption = t.Focused.SelectedOption.Foreground(ColorPrimary)
	t.Focused.SelectSelector = t.Focused.SelectSelector.Foreground(ColorPrimary)
	t.Focused.FocusedButton = t.Focused.FocusedButton.
		Background(ColorPrimary).
		Foreground(lipgloss.Color("#FFFFFF"))
	t.Focused.BlurredButton = t.Focused.BlurredButton.
		Background(ColorDim).
		Foreground(lipgloss.Color("#FFFFFF"))

	return t
}

// Confirm shows a styled yes/no prompt.
func Confirm(title string) bool {
	var result bool
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title(title).
				Affirmative("Yes").
				Negative("No").
				Value(&result),
		),
	).WithTheme(CharmTheme())

	if err := form.Run(); err != nil {
		return false
	}
	return result
}

// ConfirmWithDescription shows a confirm with extra context.
func ConfirmWithDescription(title, desc string) bool {
	var result bool
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title(title).
				Description(desc).
				Affirmative("Yes").
				Negative("No").
				Value(&result),
		),
	).WithTheme(CharmTheme())

	if err := form.Run(); err != nil {
		return false
	}
	return result
}

// InputString shows a styled text input.
func InputString(title, placeholder string) string {
	var result string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title(title).
				Placeholder(placeholder).
				Value(&result),
		),
	).WithTheme(CharmTheme())

	if err := form.Run(); err != nil {
		return ""
	}
	return result
}

// InputWithDefault shows a text input with a default value pre-filled.
func InputWithDefault(title, defaultVal string) string {
	result := defaultVal
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title(title).
				Value(&result),
		),
	).WithTheme(CharmTheme())

	if err := form.Run(); err != nil {
		return defaultVal
	}
	if result == "" {
		return defaultVal
	}
	return result
}

// SelectOne shows a styled single-select menu.
func SelectOne(title string, options []string) string {
	var result string
	opts := make([]huh.Option[string], len(options))
	for i, o := range options {
		opts[i] = huh.NewOption(o, o)
	}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title(title).
				Options(opts...).
				Value(&result),
		),
	).WithTheme(CharmTheme())

	if err := form.Run(); err != nil {
		return ""
	}
	return result
}

// MultiSelect shows a styled multi-select menu.
func MultiSelect(title string, options []string) []string {
	var result []string
	opts := make([]huh.Option[string], len(options))
	for i, o := range options {
		opts[i] = huh.NewOption(o, o)
	}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title(title).
				Options(opts...).
				Value(&result),
		),
	).WithTheme(CharmTheme())

	if err := form.Run(); err != nil {
		return nil
	}
	return result
}
