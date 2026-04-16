package steps

import (
	"fmt"
	"strings"

	"server-hardener/internal/tui"
)

func SetupFirewall() {
	fmt.Println(tui.StepHeader(4, "Firewall (UFW)"))
	fmt.Println(tui.StatusInfo("Default deny + allow only what you need."))

	if !tui.Confirm("Configure UFW firewall?") {
		fmt.Println(tui.StatusSkip("firewall setup"))
		return
	}

	// Install if missing
	if !CommandExists("ufw") {
		done := make(chan struct{})
		go tui.Spinner(done, "Installing UFW")
		_, err := RunApt(KeepExisting, "install", "-y", "ufw")
		close(done)
		if err != nil {
			fmt.Println(tui.StatusFail("Failed to install UFW"))
			return
		}
		fmt.Println(tui.StatusOK("UFW installed"))
	}

	// Default policies
	Run("ufw", "default", "deny", "incoming")
	Run("ufw", "default", "allow", "outgoing")
	fmt.Println(tui.StatusOK("Policy: deny incoming, allow outgoing"))

	// ─── SSH: always open, auto-detect port ──────────────
	sshPort := Safety.CurrentSSHPort
	if Safety.SSHPortChanged {
		sshPort = Safety.NewSSHPort
	}
	Run("ufw", "allow", sshPort+"/tcp")
	fmt.Println(tui.StatusOK(fmt.Sprintf("SSH auto-allowed on port %s", sshPort)))

	// Safety: also keep port 22 and old port open
	if sshPort != "22" {
		Run("ufw", "allow", "22/tcp")
		fmt.Println(tui.StatusOK("Port 22 also allowed (safety net)"))
	}
	if Safety.SSHPortChanged && Safety.CurrentSSHPort != Safety.NewSSHPort {
		Run("ufw", "allow", Safety.CurrentSSHPort+"/tcp")
		fmt.Println(tui.StatusOK("Old port " + Safety.CurrentSSHPort + " also allowed (safety net)"))
	}

	// Service multi-select
	services := tui.MultiSelect("Which services should be accessible?", []string{
		"HTTP (80/tcp)",
		"HTTPS (443/tcp)",
		"HTTP + HTTPS (80 & 443)",
		"PostgreSQL (5432/tcp)",
		"MySQL (3306/tcp)",
		"Redis (6379/tcp)",
		"None of these",
	})

	portMap := map[string]string{
		"HTTP (80/tcp)":         "80/tcp",
		"HTTPS (443/tcp)":      "443/tcp",
		"PostgreSQL (5432/tcp)": "5432/tcp",
		"MySQL (3306/tcp)":     "3306/tcp",
		"Redis (6379/tcp)":     "6379/tcp",
	}

	for _, s := range services {
		if s == "HTTP + HTTPS (80 & 443)" {
			Run("ufw", "allow", "80/tcp")
			Run("ufw", "allow", "443/tcp")
			fmt.Println(tui.StatusOK("Allowed HTTP + HTTPS"))
			continue
		}
		if port, ok := portMap[s]; ok {
			Run("ufw", "allow", port)
			fmt.Println(tui.StatusOK("Allowed " + s))
		}
	}

	// Custom ports
	for {
		custom := tui.InputString("Add a custom port? (e.g. 8080/tcp — blank to skip)", "")
		if custom == "" {
			break
		}
		if !strings.Contains(custom, "/") {
			custom += "/tcp"
		}
		Run("ufw", "allow", custom)
		fmt.Println(tui.StatusOK("Allowed " + custom))
	}

	// Show rules
	rules, _ := Run("ufw", "status", "numbered")
	fmt.Println()
	fmt.Println(tui.DimStyle.Render("  Pending rules:"))
	fmt.Println(tui.DimStyle.Render(rules))
	fmt.Println()

	// Verify SSH rule
	if !strings.Contains(rules, sshPort) {
		Run("ufw", "allow", sshPort+"/tcp")
		fmt.Println(tui.StatusWarn("SSH rule was missing — force added"))
	}

	if !Safety.SSHConnected {
		if tui.Confirm("Enable UFW now?") {
			enableUFW()
			Safety.FirewallEnabled = true
		}
		return
	}

	// ─── SSH session: enable with scheduled rollback ─────
	user := getSafeUser(nil)
	serverIP := getServerIP()
	portStr := portFlag(sshPort)

	// Schedule auto-disable via at
	if CommandExists("at") {
		RunShell(`echo 'ufw disable' | at now + 2 minutes 2>&1`)
		fmt.Println(tui.StatusOK("Auto-disable scheduled in 2 minutes"))
	}

	fmt.Println()
	fmt.Println(tui.DangerBox.Render(
		tui.DangerStyle.Render("🛡️  ENABLING FIREWALL\n\n") +
			"A rollback is scheduled in 2 minutes.\n\n" +
			tui.WarnStyle.Render("BEFORE confirming, test in a NEW terminal:\n\n") +
			tui.InfoStyle.Render(fmt.Sprintf("  ssh %s@%s%s\n\n", user, serverIP, portStr)) +
			"If it works → confirm here.\n" +
			"If not → wait 2 min, firewall disables itself.",
	))
	fmt.Println()

	if !tui.Confirm("Enable UFW now?") {
		cancelAtJobs()
		fmt.Println(tui.StatusSkip("firewall not enabled"))
		return
	}

	enableUFW()

	fmt.Println()
	fmt.Println(tui.CalloutBox.Render(
		tui.WarnStyle.Render("⏱️  TEST NOW!\n\n") +
			fmt.Sprintf("  ssh %s@%s%s\n\n", user, serverIP, portStr) +
			"Come back and confirm when it works.",
	))
	fmt.Println()

	if tui.Confirm("Did SSH work? Keep the firewall enabled?") {
		cancelAtJobs()
		Safety.FirewallEnabled = true
		fmt.Println(tui.StatusOK("Firewall committed! Rollback cancelled."))
	} else {
		Run("ufw", "disable")
		cancelAtJobs()
		fmt.Println(tui.StatusOK("Firewall disabled — access restored"))
	}
}

func enableUFW() {
	out, err := RunShell("echo 'y' | ufw enable")
	if err != nil {
		fmt.Println(tui.StatusFail("Failed to enable UFW: " + out))
	} else {
		fmt.Println(tui.StatusOK("UFW enabled"))
	}
}
