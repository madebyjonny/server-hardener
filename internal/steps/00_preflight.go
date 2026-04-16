package steps

import (
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"server-hardener/internal/tui"
)

// SafetyState holds the results of preflight checks. Steps reference this
// to decide what they're allowed to do — no step should trust its own
// local checks alone.
type SafetyState struct {
	// SSH key detection
	HasAnySSHKey   bool
	KeyUsers       []string // users who have authorized_keys
	CurrentSSHPort string

	// Connection info
	MyIP         string // the IP we're connected from (SSH_CLIENT)
	SSHConnected bool   // are we in an SSH session right now

	// Flags that steps set to coordinate
	SSHPortChanged    bool
	NewSSHPort        string
	PasswordAuthOff   bool
	FirewallEnabled   bool
	RootLoginDisabled bool
	CreatedUser       string // the user created in step 2
}

// Global safety state — steps read and write this.
var Safety SafetyState

// Preflight runs all safety checks before any changes are made.
// Returns false if it's too dangerous to proceed.
func Preflight() bool {
	fmt.Println(tui.StepHeader(0, "Preflight Safety Checks"))
	fmt.Println(tui.StatusInfo("Checking your access before we touch anything."))
	fmt.Println()

	allClear := true

	// ─── 1. Detect SSH session ──────────────────────────
	Safety.SSHConnected = os.Getenv("SSH_CLIENT") != "" || os.Getenv("SSH_TTY") != ""
	if Safety.SSHConnected {
		client := os.Getenv("SSH_CLIENT")
		parts := strings.Fields(client)
		if len(parts) >= 1 {
			Safety.MyIP = parts[0]
		}
		fmt.Println(tui.StatusOK(fmt.Sprintf("SSH session detected (from %s)", Safety.MyIP)))
	} else {
		fmt.Println(tui.StatusInfo("Not an SSH session — running locally or via console"))
	}

	// ─── 2. Find SSH keys ───────────────────────────────
	Safety.CurrentSSHPort = CurrentSSHPort()
	Safety.KeyUsers = findUsersWithKeys()
	Safety.HasAnySSHKey = len(Safety.KeyUsers) > 0

	if Safety.HasAnySSHKey {
		fmt.Println(tui.StatusOK(fmt.Sprintf(
			"SSH keys found for: %s", strings.Join(Safety.KeyUsers, ", "),
		)))
	} else {
		fmt.Println(tui.DangerStyle.Render("  ✘ No SSH keys found on this server"))
		fmt.Println()
		fmt.Println(tui.DangerBox.Render(
			tui.DangerStyle.Render("🔐  NO SSH KEYS DETECTED\n\n") +
				"Without SSH keys, disabling password auth or\n" +
				"misconfiguring the firewall WILL lock you out.\n\n" +
				tui.InfoStyle.Render("From your LOCAL machine, run:\n\n") +
				"  1.  ssh-keygen -t ed25519\n" +
				"  2.  ssh-copy-id <user>@<this-server>\n" +
				"  3.  ssh <user>@<this-server>  ← verify no password needed\n" +
				"  4.  Re-run this tool\n",
		))
		fmt.Println()
		allClear = false
	}

	// ─── 3. Check current SSH config isn't already broken ─
	if out, err := Run("sshd", "-t"); err != nil {
		fmt.Println(tui.StatusFail("Current sshd_config has errors: " + out))
		fmt.Println(tui.StatusWarn("Fix this before running hardening."))
		allClear = false
	} else {
		fmt.Println(tui.StatusOK("Current sshd_config is valid"))
	}

	// ─── 4. Verify SSH connectivity while we still can ──
	if Safety.SSHConnected && Safety.HasAnySSHKey {
		fmt.Println(tui.StatusOK(fmt.Sprintf("SSH port: %s", Safety.CurrentSSHPort)))

		// Try to verify we can actually reach sshd on the current port
		conn, err := net.DialTimeout("tcp",
			fmt.Sprintf("127.0.0.1:%s", Safety.CurrentSSHPort),
			2*time.Second,
		)
		if err != nil {
			fmt.Println(tui.StatusFail("Cannot connect to sshd on port " + Safety.CurrentSSHPort))
			allClear = false
		} else {
			conn.Close()
			fmt.Println(tui.StatusOK("sshd is listening and reachable"))
		}
	}

	fmt.Println()

	// ─── Decision gate ──────────────────────────────────
	if !allClear {
		fmt.Println(tui.DangerBox.Render(
			tui.DangerStyle.Render("⛔  PREFLIGHT FAILED\n\n") +
				"One or more safety checks did not pass.\n" +
				"Proceeding risks locking you out of this server.\n\n" +
				"The tool will still run, but dangerous options\n" +
				"(disable password auth, enable firewall) will be\n" +
				tui.DangerStyle.Render("BLOCKED") + " until you fix the issues above.",
		))
		fmt.Println()

		if !tui.Confirm("Continue in safe mode? (dangerous options blocked)") {
			fmt.Println()
			fmt.Println(tui.StatusInfo("Exiting. Fix the issues above and re-run."))
			return false
		}
	} else {
		fmt.Println(tui.ScoreBoxPass.Render(
			tui.SuccessStyle.Render("✔  ALL PREFLIGHT CHECKS PASSED\n") +
				"SSH keys found, sshd valid, connection verified.\n" +
				"Safe to proceed with hardening.",
		))
		fmt.Println()
	}

	return true
}

// findUsersWithKeys returns a list of usernames that have a non-empty authorized_keys.
func findUsersWithKeys() []string {
	var users []string

	// Check root
	if hasAuthorizedKeys("/root/.ssh/authorized_keys") {
		users = append(users, "root")
	}

	// Check all home directories
	entries, err := os.ReadDir("/home")
	if err != nil {
		return users
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		path := fmt.Sprintf("/home/%s/.ssh/authorized_keys", e.Name())
		if hasAuthorizedKeys(path) {
			users = append(users, e.Name())
		}
	}

	return users
}

func hasAuthorizedKeys(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	if info.Size() == 0 {
		return false
	}
	// Read and check it's not just empty lines or comments
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			return true
		}
	}
	return false
}

// CanDisablePasswordAuth returns true only if preflight found SSH keys.
func CanDisablePasswordAuth() bool {
	return Safety.HasAnySSHKey
}

// CanEnableFirewall returns true only if we're confident SSH won't be blocked.
func CanEnableFirewall() bool {
	return Safety.HasAnySSHKey || !Safety.SSHConnected
}
