package tui

import (
	"fmt"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// Banner prints the full-width startup banner.
func Banner() string {
	logo := `
  ██████  ██░ ██  ▄▄▄       ██▀███  ▓█████▄ ▓█████  ███▄    █ 
▒██    ▒ ▓██░ ██▒▒████▄    ▓██ ▒ ██▒▒██▀ ██▌▓█   ▀  ██ ▀█   █ 
░ ▓██▄   ▒██▀▀██░▒██  ▀█▄  ▓██ ░▄█ ▒░██   █▌▒███   ▓██  ▀█ ██▒
  ▒   ██▒░▓█ ░██ ░██▄▄▄▄██ ▒██▀▀█▄  ░▓█▄   ▌▒▓█  ▄ ▓██▒  ▐▌██▒
▒██████▒▒░▓█▒░██▓ ▓█   ▓██▒░██▓ ▒██▒░▒████▓ ░▒████▒▒██░   ▓██░
▒ ▒▓▒ ▒ ░ ▒ ░░▒░▒ ▒▒   ▓▒█░░ ▒▓ ░▒▓░ ▒▒▓  ▒ ░░ ▒░ ░░ ▒░   ▒ ▒ 
`

	logoStyle := lipgloss.NewStyle().
		Foreground(ColorPrimary).
		Bold(true)

	tagline := SubtitleStyle.Render("  Interactive server hardening — the fun way.")
	version := DimStyle.Render("  v1.0.0 • requires root • ubuntu/debian")

	return logoStyle.Render(logo) + "\n" + tagline + "\n" + version + "\n"
}

// Spinner runs an inline spinner with elapsed time so long ops don't look hung.
// Call in a goroutine: go tui.Spinner(done, "Updating packages")
func Spinner(done <-chan struct{}, label string) {
	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	start := time.Now()
	i := 0
	for {
		select {
		case <-done:
			// Clear the spinner line
			fmt.Printf("\r%s\r", "                                                            ")
			return
		default:
			elapsed := time.Since(start).Truncate(time.Second)
			fmt.Printf("\r  %s %s %s",
				InfoStyle.Render(frames[i%len(frames)]),
				DimStyle.Render(label+"..."),
				DimStyle.Render(fmt.Sprintf("(%s)", elapsed)),
			)
			time.Sleep(80 * time.Millisecond)
			i++
		}
	}
}
