package steps

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"server-hardener/internal/tui"
)

const sshdConfig = "/etc/ssh/sshd_config"

func HardenSSH() {
	fmt.Println(tui.StepHeader(3, "SSH Hardening"))
	fmt.Println(tui.StatusInfo("The #1 thing attackers probe. Let's lock it down."))

	if !tui.Confirm("Harden SSH configuration?") {
		fmt.Println(tui.StatusSkip("SSH hardening"))
		return
	}

	// ─── Backup ──────────────────────────────────────────
	BackupFile(sshdConfig)
	fmt.Println(tui.StatusOK("Backed up sshd_config → sshd_config.bak"))

	data, err := os.ReadFile(sshdConfig)
	if err != nil {
		fmt.Println(tui.StatusFail("Cannot read sshd_config"))
		return
	}
	config := string(data)
	originalConfig := config

	// ─── Situational awareness ───────────────────────────
	nonRootKeyUsers := []string{}
	for _, u := range Safety.KeyUsers {
		if u != "root" {
			nonRootKeyUsers = append(nonRootKeyUsers, u)
		}
	}

	fmt.Println()
	if len(Safety.KeyUsers) > 0 {
		fmt.Println(tui.StatusInfo("Users with SSH keys: " + strings.Join(Safety.KeyUsers, ", ")))
	}
	if len(nonRootKeyUsers) > 0 {
		fmt.Println(tui.StatusInfo("Non-root users with keys: " + strings.Join(nonRootKeyUsers, ", ")))
	} else {
		fmt.Println(tui.StatusWarn("No non-root users have SSH keys"))
	}
	fmt.Println()

	// ─── Collect choices ─────────────────────────────────
	wantsDisableRoot := false
	wantsDisablePassword := false
	newPort := Safety.CurrentSSHPort

	// Root login
	if tui.Confirm("Disable root login via SSH?") {
		if len(nonRootKeyUsers) == 0 && Safety.SSHConnected {
			fmt.Println(tui.DangerBox.Render(
				tui.DangerStyle.Render("⛔  BLOCKED\n\n") +
					"No non-root user has SSH keys.\n" +
					"Disabling root = nobody can log in.\n\n" +
					"Set up a non-root user with keys first (step 2).",
			))
			fmt.Println()
		} else {
			wantsDisableRoot = true
		}
	}

	// Password auth
	fmt.Println()
	if !CanDisablePasswordAuth() {
		fmt.Println(tui.DangerBox.Render(
			tui.DangerStyle.Render("🔐  PASSWORD AUTH — BLOCKED\n\n") +
				"No SSH keys found. This option is locked.",
		))
		fmt.Println()
	} else {
		if tui.ConfirmWithDescription(
			"Disable password authentication (key-only)?",
			"Only if you've confirmed key login works.",
		) {
			wantsDisablePassword = true
		}
	}

	// Port
	portChoice := tui.SelectOne("SSH Port", []string{
		"Keep default (22)",
		"Change to 2222",
		"Custom port",
	})
	switch portChoice {
	case "Change to 2222":
		newPort = "2222"
	case "Custom port":
		newPort = tui.InputWithDefault("Enter port number", "22")
	}

	// ─── Build config ────────────────────────────────────
	if wantsDisableRoot {
		config = setSshdOption(config, "PermitRootLogin", "no")
		Safety.RootLoginDisabled = true
	}
	if wantsDisablePassword {
		config = setSshdOption(config, "PasswordAuthentication", "no")
		config = setSshdOption(config, "PubkeyAuthentication", "yes")
	}
	if newPort != Safety.CurrentSSHPort {
		config = setSshdOption(config, "Port", newPort)
	}

	// Safe hardening
	config = setSshdOption(config, "X11Forwarding", "no")
	config = setSshdOption(config, "MaxAuthTries", "3")
	config = setSshdOption(config, "ClientAliveInterval", "300")
	config = setSshdOption(config, "ClientAliveCountMax", "2")

	// ─── Show diff ───────────────────────────────────────
	fmt.Println()
	fmt.Println(tui.StepHeaderStyle.Render("Changes to apply"))
	showConfigDiff(originalConfig, config)
	fmt.Println()

	if config == originalConfig {
		fmt.Println(tui.StatusInfo("No changes to apply."))
		return
	}

	// ─── Open firewall port BEFORE touching SSH ──────────
	if newPort != Safety.CurrentSSHPort {
		ufwStatus, _ := Run("ufw", "status")
		if strings.Contains(ufwStatus, "active") {
			Run("ufw", "allow", newPort+"/tcp")
			fmt.Println(tui.StatusOK(fmt.Sprintf("Opened port %s in firewall first", newPort)))
		}
	}

	// ─── Write config ────────────────────────────────────
	if err := WriteFile(sshdConfig, config, 0644); err != nil {
		fmt.Println(tui.StatusFail("Failed to write sshd_config"))
		return
	}

	// ─── Validate syntax ─────────────────────────────────
	if out, err := Run("sshd", "-t"); err != nil {
		fmt.Println(tui.StatusFail("Config validation failed: " + out))
		WriteFile(sshdConfig, originalConfig, 0644)
		fmt.Println(tui.StatusOK("Restored original config"))
		return
	}
	fmt.Println(tui.StatusOK("Config syntax valid"))

	// ─── Schedule automatic rollback ─────────────────────
	// Use `at` to schedule a rollback in 2 minutes. This is a system
	// service that runs independently — it doesn't care if our process
	// dies, the terminal hangs, or the connection drops.

	// Install at if missing
	if !CommandExists("at") {
		fmt.Println(tui.StatusInfo("Installing 'at' for scheduled rollback..."))
		RunApt(KeepExisting, "install", "-y", "at")
		Run("systemctl", "enable", "atd")
		Run("systemctl", "start", "atd")
	}

	rollbackCmd := fmt.Sprintf(
		`cp %s.bak %s && systemctl restart ssh 2>/dev/null; systemctl restart sshd 2>/dev/null`,
		sshdConfig, sshdConfig,
	)
	out, err := RunShell(fmt.Sprintf(`echo '%s' | at now + 2 minutes 2>&1`, rollbackCmd))
	if err != nil {
		fmt.Println(tui.StatusWarn("Could not schedule auto-rollback via 'at'"))
		fmt.Println(tui.DimStyle.Render("    " + out))
		fmt.Println(tui.StatusInfo("If something goes wrong, manually run:"))
		fmt.Println(tui.InfoStyle.Render(fmt.Sprintf("    cp %s.bak %s && systemctl restart ssh", sshdConfig, sshdConfig)))
		fmt.Println()
	} else {
		fmt.Println(tui.StatusOK("Auto-rollback scheduled in 2 minutes"))
	}

	// ─── Apply: restart sshd ─────────────────────────────
	// Use restart, not reload. Reload hangs on many distros.
	// Our current session survives because SSH doesn't kill existing
	// connections on restart — only new connections use the new config.

	user := getSafeUser(nonRootKeyUsers)
	serverIP := getServerIP()
	portStr := portFlag(newPort)

	fmt.Println()
	fmt.Println(tui.DangerBox.Render(
		tui.DangerStyle.Render("🛡️  APPLYING SSH CHANGES\n\n") +
			"SSH will restart. Your current session stays alive.\n" +
			"A rollback is scheduled in 2 minutes.\n\n" +
			tui.WarnStyle.Render("BEFORE confirming, open a NEW terminal and run:\n\n") +
			tui.InfoStyle.Render(fmt.Sprintf("  ssh %s@%s%s\n\n", user, serverIP, portStr)) +
			"If that works → confirm here to keep changes.\n" +
			"If it fails  → do nothing, rollback happens in 2 min.",
	))
	fmt.Println()

	if !tui.Confirm("Restart SSH now?") {
		fmt.Println(tui.StatusInfo("Aborting — restoring original config."))
		WriteFile(sshdConfig, originalConfig, 0644)
		cancelAtJobs()
		return
	}

	fmt.Println(tui.StatusInfo("Restarting SSH..."))
	if _, err := Run("systemctl", "restart", "ssh"); err != nil {
		if _, err := Run("systemctl", "restart", "sshd"); err != nil {
			fmt.Println(tui.StatusFail("Restart failed — restoring backup"))
			WriteFile(sshdConfig, originalConfig, 0644)
			Run("systemctl", "restart", "ssh")
			Run("systemctl", "restart", "sshd")
			cancelAtJobs()
			return
		}
	}
	fmt.Println(tui.StatusOK("SSH restarted"))

	// ─── User tests, then confirms ───────────────────────
	fmt.Println()
	fmt.Println(tui.CalloutBox.Render(
		tui.WarnStyle.Render("⏱️  2 MINUTES until auto-rollback\n\n") +
			tui.InfoStyle.Render("Open a NEW terminal RIGHT NOW and test:\n\n") +
			fmt.Sprintf("  ssh %s@%s%s\n\n", user, serverIP, portStr) +
			"DO NOT close this terminal.\n" +
			"If the test works, come back and confirm below.\n" +
			"If it doesn't, just wait — it'll roll back automatically.",
	))
	fmt.Println()

	if tui.Confirm("Did SSH work in the other terminal? Keep these changes?") {
		// Cancel the rollback
		cancelAtJobs()
		fmt.Println(tui.StatusOK("Changes committed! Rollback cancelled."))

		if wantsDisablePassword {
			Safety.PasswordAuthOff = true
		}
		if newPort != Safety.CurrentSSHPort {
			Safety.SSHPortChanged = true
			Safety.NewSSHPort = newPort
		}
	} else {
		// Roll back immediately, don't wait
		fmt.Println(tui.StatusInfo("Rolling back now..."))
		WriteFile(sshdConfig, originalConfig, 0644)
		Run("systemctl", "restart", "ssh")
		Run("systemctl", "restart", "sshd")
		cancelAtJobs()
		Safety.RootLoginDisabled = false
		Safety.PasswordAuthOff = false
		Safety.SSHPortChanged = false
		fmt.Println(tui.StatusOK("Original config restored"))
	}
}

// ─── Helpers ─────────────────────────────────────────────────

func cancelAtJobs() {
	// Remove all pending at jobs (ours is the only recent one)
	out, _ := Run("atq")
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Fields(line)
		if len(fields) > 0 {
			Run("atrm", fields[0])
		}
	}
}

func showConfigDiff(oldConfig, newConfig string) {
	WriteFile("/tmp/.hardener-old.conf", oldConfig, 0644)
	WriteFile("/tmp/.hardener-new.conf", newConfig, 0644)

	out, _ := Run("diff", "--color=never", "-u",
		"/tmp/.hardener-old.conf", "/tmp/.hardener-new.conf")

	os.Remove("/tmp/.hardener-old.conf")
	os.Remove("/tmp/.hardener-new.conf")

	if out == "" {
		fmt.Println(tui.DimStyle.Render("  (no changes)"))
		return
	}

	for _, line := range strings.Split(out, "\n") {
		switch {
		case strings.HasPrefix(line, "---"), strings.HasPrefix(line, "+++"):
			continue
		case strings.HasPrefix(line, "@@"):
			fmt.Println(tui.DimStyle.Render("  " + line))
		case strings.HasPrefix(line, "-"):
			fmt.Println(tui.DangerStyle.Render("  " + line))
		case strings.HasPrefix(line, "+"):
			fmt.Println(tui.SuccessStyle.Render("  " + line))
		default:
			fmt.Println(tui.DimStyle.Render("  " + line))
		}
	}
}

func getSafeUser(nonRootUsers []string) string {
	if Safety.CreatedUser != "" {
		return Safety.CreatedUser
	}
	if len(nonRootUsers) > 0 {
		return nonRootUsers[0]
	}
	for _, u := range Safety.KeyUsers {
		if u != "root" {
			return u
		}
	}
	if !Safety.RootLoginDisabled && containsString(Safety.KeyUsers, "root") {
		return "root"
	}
	return "<your-user>"
}

func portFlag(port string) string {
	if port != "22" {
		return fmt.Sprintf(" -p %s", port)
	}
	return ""
}

func setSshdOption(config, key, value string) string {
	re := regexp.MustCompile(`(?m)^#?\s*` + key + `\s+.*$`)
	replacement := fmt.Sprintf("%s %s", key, value)
	if re.MatchString(config) {
		return re.ReplaceAllString(config, replacement)
	}
	return strings.TrimRight(config, "\n") + "\n" + replacement + "\n"
}
