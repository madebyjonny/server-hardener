package steps

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// dpkgEnv returns environment variables that suppress all interactive
// dpkg/debconf prompts (the "install maintainer's version?" TUI panels).
func dpkgEnv() []string {
	env := os.Environ()
	env = append(env, "DEBIAN_FRONTEND=noninteractive")
	env = append(env, "DEBCONF_NONINTERACTIVE_SEEN=true")
	return env
}

// dpkgOpts are the apt options that auto-resolve config file conflicts.
// -o Dpkg::Options::="--force-confdef"  → keep current config if untouched, use maintainer's if new
// -o Dpkg::Options::="--force-confold"  → if in doubt, keep the existing config (safe default)
var dpkgOpts = []string{
	"-o", `Dpkg::Options::=--force-confdef`,
	"-o", `Dpkg::Options::=--force-confold`,
}

// ConfigPolicy controls how apt handles config file conflicts.
type ConfigPolicy int

const (
	// KeepExisting keeps your current config files (safe default).
	KeepExisting ConfigPolicy = iota
	// UseMaintainer replaces with the package maintainer's version.
	UseMaintainer
)

// dpkgOptsForPolicy returns the right dpkg flags for the chosen policy.
func dpkgOptsForPolicy(policy ConfigPolicy) []string {
	switch policy {
	case UseMaintainer:
		return []string{
			"-o", `Dpkg::Options::=--force-confnew`,
		}
	default: // KeepExisting
		return []string{
			"-o", `Dpkg::Options::=--force-confdef`,
			"-o", `Dpkg::Options::=--force-confold`,
		}
	}
}

// Run executes a command and returns combined output.
func Run(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

// RunApt executes an apt command in non-interactive mode with the given
// config conflict policy. No dpkg TUI prompts will appear.
func RunApt(policy ConfigPolicy, args ...string) (string, error) {
	fullArgs := append(args, dpkgOptsForPolicy(policy)...)
	cmd := exec.Command("apt", fullArgs...)
	cmd.Env = dpkgEnv()
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

// RunAptLive is like RunApt but streams output to the terminal.
func RunAptLive(policy ConfigPolicy, args ...string) error {
	fullArgs := append(args, dpkgOptsForPolicy(policy)...)
	cmd := exec.Command("apt", fullArgs...)
	cmd.Env = dpkgEnv()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// RunLive executes a command with stdout/stderr streaming to the terminal.
func RunLive(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// RunShell runs a command through /bin/sh.
func RunShell(command string) (string, error) {
	return Run("sh", "-c", command)
}

// FileContains checks if a file contains a substring.
func FileContains(path, substr string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	return strings.Contains(string(data), substr)
}

// WriteFile writes content to a file.
func WriteFile(path, content string, perm os.FileMode) error {
	return os.WriteFile(path, []byte(content), perm)
}

// BackupFile creates a .bak copy (won't overwrite existing backup).
func BackupFile(path string) error {
	_, err := Run("cp", "-n", path, fmt.Sprintf("%s.bak", path))
	return err
}

// IsRoot checks effective UID.
func IsRoot() bool {
	return os.Geteuid() == 0
}

// CommandExists checks if a binary is on PATH.
func CommandExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// SSHKeyExists checks if the given user has authorized_keys set up.
func SSHKeyExists(username string) bool {
	var path string
	if username == "root" {
		path = "/root/.ssh/authorized_keys"
	} else {
		path = fmt.Sprintf("/home/%s/.ssh/authorized_keys", username)
	}
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	if info.Size() == 0 {
		return false
	}
	// Check it's not just comments/empty lines
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

// CurrentSSHPort reads the configured SSH port from sshd_config.
func CurrentSSHPort() string {
	data, err := os.ReadFile("/etc/ssh/sshd_config")
	if err != nil {
		return "22"
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Port ") {
			parts := strings.Fields(line)
			if len(parts) == 2 {
				return parts[1]
			}
		}
	}
	return "22"
}
