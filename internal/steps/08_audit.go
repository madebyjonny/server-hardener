package steps

import (
	"fmt"
	"strings"

	"server-hardener/internal/tui"
)

func AuditSummary() {
	fmt.Println(tui.StepHeader(8, "Security Audit"))
	fmt.Println()

	checks := []struct {
		name  string
		check func() bool
	}{
		{"Root SSH disabled", func() bool {
			return FileContains("/etc/ssh/sshd_config", "PermitRootLogin no")
		}},
		{"Password auth disabled", func() bool {
			return FileContains("/etc/ssh/sshd_config", "PasswordAuthentication no")
		}},
		{"SSH key auth enabled", func() bool {
			return FileContains("/etc/ssh/sshd_config", "PubkeyAuthentication yes")
		}},
		{"UFW active", func() bool {
			out, _ := Run("ufw", "status")
			return strings.Contains(out, "active")
		}},
		{"Fail2Ban running", func() bool {
			out, _ := Run("systemctl", "is-active", "fail2ban")
			return strings.TrimSpace(out) == "active"
		}},
		{"Unattended upgrades", func() bool {
			_, err := Run("dpkg", "-l", "unattended-upgrades")
			return err == nil
		}},
		{"Kernel hardening applied", func() bool {
			return FileContains("/etc/sysctl.d/99-hardening.conf", "tcp_syncookies")
		}},
	}

	passed := 0
	results := ""
	for _, c := range checks {
		ok := c.check()
		if ok {
			results += tui.StatusOK(c.name) + "\n"
			passed++
		} else {
			results += tui.StatusFail(c.name) + "\n"
		}
	}

	fmt.Print(results)
	fmt.Println()

	total := len(checks)
	pct := (passed * 100) / total
	score := fmt.Sprintf("%d / %d checks passed (%d%%)", passed, total, pct)

	switch {
	case pct == 100:
		fmt.Println(tui.ScoreBoxPass.Render(
			tui.SuccessStyle.Render("🎉 FULLY HARDENED\n") + score,
		))
	case pct >= 70:
		fmt.Println(tui.ScoreBoxWarn.Render(
			tui.WarnStyle.Render("⚡ GOOD — room to improve\n") + score,
		))
	default:
		fmt.Println(tui.ScoreBoxFail.Render(
			tui.DangerStyle.Render("🛑 NEEDS ATTENTION\n") + score,
		))
	}

	// Next steps
	fmt.Println()
	fmt.Println(tui.TitleStyle.Render(" Next Steps "))
	fmt.Println()
	tips := []string{
		"Set up SSH keys if not done → ssh-copy-id user@server",
		"Add 2FA to SSH → apt install libpam-google-authenticator",
		"Install log monitoring → logwatch or GoAccess",
		"Set up intrusion detection → AIDE or rkhunter",
		"Schedule automated backups",
		"Consider CrowdSec for community threat intel",
	}
	for _, tip := range tips {
		fmt.Println(tui.DimStyle.Render("  → ") + tip)
	}
	fmt.Println()
}
