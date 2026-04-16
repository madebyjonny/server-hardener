package steps

import (
	"fmt"
	"os"
	"strings"

	"server-hardener/internal/tui"
)

func CreateUser() {
	fmt.Println(tui.StepHeader(2, "Create Admin User"))
	fmt.Println(tui.StatusInfo("Best practice: never work as root directly."))

	if !tui.Confirm("Create a new sudo-capable user?") {
		fmt.Println(tui.StatusSkip("user creation"))
		return
	}

	username := tui.InputString("Username", "deploy")
	if username == "" {
		fmt.Println(tui.StatusSkip("no username provided"))
		return
	}

	// ─── Create user (non-interactive) ───────────────────
	if _, err := Run("id", username); err == nil {
		fmt.Println(tui.StatusInfo(fmt.Sprintf("User '%s' already exists", username)))
	} else {
		// --disabled-password avoids the password prompt that hangs
		// the TUI. The user logs in via SSH key, not password.
		fmt.Println(tui.StatusInfo("Creating user (no password — key-only login)..."))
		if _, err := Run("adduser",
			"--disabled-password",
			"--gecos", "",
			username,
		); err != nil {
			fmt.Println(tui.StatusFail("Failed to create user: " + err.Error()))
			return
		}
		fmt.Println(tui.StatusOK(fmt.Sprintf("User '%s' created", username)))
	}

	// ─── Add to sudo ─────────────────────────────────────
	if _, err := Run("usermod", "-aG", "sudo", username); err != nil {
		fmt.Println(tui.StatusFail("Failed to add to sudo group"))
	} else {
		fmt.Println(tui.StatusOK(fmt.Sprintf("'%s' added to sudo group", username)))
	}

	// Track this user so later steps show the right username
	Safety.CreatedUser = username

	// ─── SSH key setup ───────────────────────────────────
	// This is the critical part. The new user MUST have a working
	// SSH key before we let anything else proceed.

	if SSHKeyExists(username) {
		fmt.Println(tui.StatusOK("SSH key already present for " + username))
		updateSafetyWithUser(username)
		verifySSHLogin(username)
		return
	}

	// No key for this user — figure out how to get one there.
	fmt.Println()
	fmt.Println(tui.StatusWarn(fmt.Sprintf("No SSH key found for '%s'", username)))

	// Check if root has keys we can copy
	rootHasKeys := hasAuthorizedKeys("/root/.ssh/authorized_keys")

	if rootHasKeys {
		fmt.Println()
		fmt.Println(tui.CalloutBox.Render(
			tui.InfoStyle.Render("💡  Root has SSH keys set up.\n\n") +
				"Since you're logged in as root with a key, the same key\n" +
				"on your local machine can authenticate as '" + username + "'\n" +
				"if we copy root's authorized_keys to the new user.",
		))
		fmt.Println()

		if tui.ConfirmWithDescription(
			fmt.Sprintf("Copy root's authorized_keys to '%s'?", username),
			"This is the easiest and safest option — same key, new user.",
		) {
			if err := copyRootKeysToUser(username); err != nil {
				fmt.Println(tui.StatusFail("Failed to copy keys: " + err.Error()))
			} else {
				fmt.Println(tui.StatusOK(fmt.Sprintf("Root's SSH keys copied to '%s'", username)))
				updateSafetyWithUser(username)
				verifySSHLogin(username)
				return
			}
		}
	}

	// Option: paste a key
	fmt.Println()
	keyMethod := tui.SelectOne("How do you want to set up the SSH key?", []string{
		"Paste my public key now",
		"I'll run ssh-copy-id from my machine (pause here)",
		"Skip — I'll handle it later",
	})

	switch keyMethod {
	case "Paste my public key now":
		fmt.Println()
		fmt.Println(tui.StatusInfo("Paste the FULL line from your local ~/.ssh/id_ed25519.pub"))
		fmt.Println(tui.StatusInfo("(or id_rsa.pub). It starts with ssh-ed25519 or ssh-rsa."))
		fmt.Println()

		pubkey := tui.InputString("Public key", "ssh-ed25519 AAAA...")
		pubkey = strings.TrimSpace(pubkey)

		if !isValidPubKey(pubkey) {
			fmt.Println(tui.StatusFail("That doesn't look like a valid SSH public key."))
			fmt.Println(tui.StatusInfo("It should start with ssh-ed25519, ssh-rsa, ecdsa-sha2, etc."))
			fmt.Println(tui.StatusWarn("Key not installed. SSH hardening will be limited."))
			return
		}

		if err := installKeyForUser(username, pubkey); err != nil {
			fmt.Println(tui.StatusFail("Failed to install key: " + err.Error()))
			return
		}
		fmt.Println(tui.StatusOK("SSH key installed for " + username))
		updateSafetyWithUser(username)
		verifySSHLogin(username)

	case "I'll run ssh-copy-id from my machine (pause here)":
		fmt.Println()
		fmt.Println(tui.CalloutBox.Render(
			tui.InfoStyle.Render("On your LOCAL machine, run:\n\n") +
				fmt.Sprintf("  ssh-copy-id %s@%s\n\n", username, getServerIP()) +
				"Then come back here and press Enter.",
		))
		fmt.Println()

		// Wait for them
		tui.InputString("Press Enter when done", "")

		// Re-check
		if SSHKeyExists(username) {
			fmt.Println(tui.StatusOK("Key detected for " + username))
			updateSafetyWithUser(username)
			verifySSHLogin(username)
		} else {
			fmt.Println(tui.StatusFail("Still no key found for " + username))
			fmt.Println(tui.StatusWarn("SSH hardening will be limited."))

			// One more chance — ssh-copy-id needs password auth,
			// which won't work with --disabled-password
			fmt.Println()
			fmt.Println(tui.CalloutBox.Render(
				tui.WarnStyle.Render("⚠  ssh-copy-id needs password auth to work.\n\n") +
					"Since we created this user without a password,\n" +
					"ssh-copy-id can't authenticate.\n\n" +
					tui.InfoStyle.Render("Try pasting the key instead, or set a temporary password:\n") +
					fmt.Sprintf("  sudo passwd %s\n", username) +
					"  Then retry ssh-copy-id, then re-run this tool.",
			))
			fmt.Println()
		}

	default:
		fmt.Println(tui.StatusSkip("SSH key setup"))
		fmt.Println(tui.StatusWarn("SSH hardening will be limited until a key is set up."))
	}
}

// ─── Helpers ─────────────────────────────────────────────────

func copyRootKeysToUser(username string) error {
	rootKeys, err := os.ReadFile("/root/.ssh/authorized_keys")
	if err != nil {
		return fmt.Errorf("cannot read root's authorized_keys: %w", err)
	}

	sshDir := fmt.Sprintf("/home/%s/.ssh", username)
	Run("mkdir", "-p", sshDir)

	authFile := sshDir + "/authorized_keys"
	if err := WriteFile(authFile, string(rootKeys), 0600); err != nil {
		return fmt.Errorf("cannot write authorized_keys: %w", err)
	}

	// Fix ownership — this is critical, sshd rejects keys with wrong ownership
	Run("chown", "-R", username+":"+username, sshDir)
	Run("chmod", "700", sshDir)
	Run("chmod", "600", authFile)

	return nil
}

func installKeyForUser(username, pubkey string) error {
	sshDir := fmt.Sprintf("/home/%s/.ssh", username)
	Run("mkdir", "-p", sshDir)

	authFile := sshDir + "/authorized_keys"

	// Append rather than overwrite, in case there are existing keys
	existing := ""
	if data, err := os.ReadFile(authFile); err == nil {
		existing = string(data)
	}

	content := existing
	if content != "" && !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	content += pubkey + "\n"

	if err := WriteFile(authFile, content, 0600); err != nil {
		return err
	}

	Run("chown", "-R", username+":"+username, sshDir)
	Run("chmod", "700", sshDir)
	Run("chmod", "600", authFile)

	return nil
}

func isValidPubKey(key string) bool {
	prefixes := []string{"ssh-ed25519", "ssh-rsa", "ecdsa-sha2", "ssh-dss", "sk-ssh-ed25519", "sk-ecdsa-sha2"}
	for _, p := range prefixes {
		if strings.HasPrefix(key, p+" ") {
			// Basic sanity: should have at least 2 space-separated parts
			parts := strings.Fields(key)
			return len(parts) >= 2
		}
	}
	return false
}

func verifySSHLogin(username string) {
	fmt.Println()
	fmt.Println(tui.StatusInfo("Verifying SSH key login works..."))

	// Try to SSH to localhost as the user — this proves sshd accepts the key
	port := Safety.CurrentSSHPort
	out, err := Run("ssh",
		"-o", "StrictHostKeyChecking=no",
		"-o", "BatchMode=yes",              // fail immediately if key doesn't work
		"-o", "ConnectTimeout=5",
		"-p", port,
		username+"@127.0.0.1",
		"echo", "hardener-test-ok",
	)

	if err != nil || !strings.Contains(out, "hardener-test-ok") {
		fmt.Println(tui.StatusWarn("Could not verify SSH login for " + username))
		fmt.Println(tui.DimStyle.Render("    (This might be fine — local SSH test can fail for valid reasons)"))
		fmt.Println(tui.DimStyle.Render("    Test manually: ssh " + username + "@" + getServerIP()))
	} else {
		fmt.Println(tui.StatusOK(fmt.Sprintf("Verified: SSH login works for '%s'", username)))
	}
}

func updateSafetyWithUser(username string) {
	Safety.HasAnySSHKey = true
	if !containsString(Safety.KeyUsers, username) {
		Safety.KeyUsers = append(Safety.KeyUsers, username)
	}
}

func getServerIP() string {
	if Safety.MyIP != "" {
		// We know the client IP, but we need the server IP
		// Best effort: check hostname -I
		out, err := Run("hostname", "-I")
		if err == nil {
			parts := strings.Fields(out)
			if len(parts) > 0 {
				return parts[0]
			}
		}
	}
	return "<this-server-ip>"
}

func containsString(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}
