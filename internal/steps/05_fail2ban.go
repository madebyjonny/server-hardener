package steps

import (
	"fmt"

	"server-hardener/internal/tui"
)

func SetupFail2Ban() {
	fmt.Println(tui.StepHeader(5, "Fail2Ban"))
	fmt.Println(tui.StatusInfo("Watches logs and auto-bans brute-force attackers."))

	if !tui.Confirm("Install and configure Fail2Ban?") {
		fmt.Println(tui.StatusSkip("Fail2Ban"))
		return
	}

	// Install
	done := make(chan struct{})
	go tui.Spinner(done, "Installing Fail2Ban")
	_, err := RunApt(KeepExisting, "install", "-y", "fail2ban")
	close(done)
	if err != nil {
		fmt.Println(tui.StatusFail("Failed to install fail2ban"))
		return
	}
	fmt.Println(tui.StatusOK("Fail2Ban installed"))

	// Configure via interactive prompts
	preset := tui.SelectOne("Ban aggressiveness", []string{
		"Lenient  — ban 10min after 5 failures",
		"Moderate — ban 1hr after 3 failures",
		"Strict   — ban 24hr after 3 failures",
		"Custom",
	})

	var bantime, maxretry, findtime string
	switch {
	case preset == "Lenient  — ban 10min after 5 failures":
		bantime, maxretry, findtime = "600", "5", "600"
	case preset == "Strict   — ban 24hr after 3 failures":
		bantime, maxretry, findtime = "86400", "3", "3600"
	case preset == "Custom":
		bantime = tui.InputWithDefault("Ban duration (seconds)", "3600")
		maxretry = tui.InputWithDefault("Max retries before ban", "5")
		findtime = tui.InputWithDefault("Time window for retries (seconds)", "600")
	default: // Moderate
		bantime, maxretry, findtime = "3600", "3", "600"
	}

	jail := fmt.Sprintf(`[DEFAULT]
bantime  = %s
findtime = %s
maxretry = %s
banaction = ufw

[sshd]
enabled  = true
port     = ssh
filter   = sshd
logpath  = /var/log/auth.log
`, bantime, findtime, maxretry)

	if err := WriteFile("/etc/fail2ban/jail.local", jail, 0644); err != nil {
		fmt.Println(tui.StatusFail("Failed to write jail.local"))
		return
	}
	fmt.Println(tui.StatusOK(fmt.Sprintf("jail.local: ban=%ss, retries=%s, window=%ss", bantime, maxretry, findtime)))

	fmt.Println(tui.StatusInfo("Starting Fail2Ban service..."))
	Run("systemctl", "enable", "fail2ban")
	if _, err := Run("systemctl", "restart", "fail2ban"); err != nil {
		fmt.Println(tui.StatusFail("Failed to restart fail2ban"))
	} else {
		fmt.Println(tui.StatusOK("Fail2Ban running and enabled on boot"))
	}
}
