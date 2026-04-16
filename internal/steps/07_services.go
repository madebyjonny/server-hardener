package steps

import (
	"fmt"
	"strings"

	"server-hardener/internal/tui"
)

func DisableServices() {
	fmt.Println(tui.StepHeader(7, "Disable Unused Services"))
	fmt.Println(tui.StatusInfo("Less running = less attack surface."))

	if !tui.Confirm("Scan for and disable unused services?") {
		fmt.Println(tui.StatusSkip("service review"))
		return
	}

	type svc struct {
		name string
		desc string
	}

	candidates := []svc{
		{"avahi-daemon", "mDNS/Bonjour discovery"},
		{"cups", "Printing"},
		{"cups-browsed", "Printer browsing"},
		{"bluetooth", "Bluetooth"},
		{"ModemManager", "Modem/mobile broadband"},
		{"whoopsie", "Ubuntu error reporting"},
		{"apport", "Crash reports"},
		{"snapd", "Snap package manager"},
	}

	// Find which are actually enabled
	var active []string
	activeMap := map[string]svc{}
	for _, s := range candidates {
		out, _ := Run("systemctl", "is-enabled", s.name)
		out = strings.TrimSpace(out)
		if out == "enabled" || out == "static" {
			label := fmt.Sprintf("%s — %s", s.name, s.desc)
			active = append(active, label)
			activeMap[label] = s
		}
	}

	if len(active) == 0 {
		fmt.Println(tui.StatusOK("No unnecessary services found running. Clean!"))
		return
	}

	toDisable := tui.MultiSelect("Select services to disable:", active)

	if len(toDisable) == 0 {
		fmt.Println(tui.StatusSkip("no services selected"))
		return
	}

	for _, label := range toDisable {
		s := activeMap[label]
		Run("systemctl", "stop", s.name)
		Run("systemctl", "disable", s.name)
		Run("systemctl", "mask", s.name)
		fmt.Println(tui.StatusOK(fmt.Sprintf("Stopped, disabled, and masked %s", s.name)))
	}
}
