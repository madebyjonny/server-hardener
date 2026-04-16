package steps

import (
	"fmt"

	"server-hardener/internal/tui"
)

func UpdateSystem() {
	fmt.Println(tui.StepHeader(1, "System Updates"))
	fmt.Println(tui.StatusInfo("Patches known vulnerabilities via apt."))

	if !tui.Confirm("Run system updates?") {
		fmt.Println(tui.StatusSkip("system updates"))
		return
	}

	// Ask about config conflict policy BEFORE starting so there are
	// no surprise dpkg TUI panels mid-install.
	policyChoice := tui.SelectOne(
		"When a package ships a new config file that you've modified:",
		[]string{
			"Keep my existing configs (safe default)",
			"Use the maintainer's new version",
		},
	)
	policy := KeepExisting
	if policyChoice == "Use the maintainer's new version" {
		policy = UseMaintainer
	}
	fmt.Println(tui.StatusInfo("Config policy set — no interactive prompts will appear."))

	// Choose output mode
	verbose := tui.Confirm("Show live apt output? (recommended for long upgrades)")

	// apt update
	fmt.Println(tui.StatusInfo("Updating package lists..."))
	if verbose {
		if err := RunAptLive(policy, "update", "-y"); err != nil {
			fmt.Println(tui.StatusFail("apt update failed"))
			return
		}
	} else {
		done := make(chan struct{})
		go tui.Spinner(done, "Updating package lists")
		_, err := RunApt(policy, "update", "-y")
		close(done)
		if err != nil {
			fmt.Println(tui.StatusFail("apt update failed"))
			return
		}
	}
	fmt.Println(tui.StatusOK("Package lists updated"))

	// apt upgrade
	fmt.Println(tui.StatusInfo("Upgrading packages (this can take several minutes)..."))
	if verbose {
		if err := RunAptLive(policy, "upgrade", "-y"); err != nil {
			fmt.Println(tui.StatusFail("apt upgrade failed"))
			return
		}
	} else {
		done := make(chan struct{})
		go tui.Spinner(done, "Upgrading packages")
		_, err := RunApt(policy, "upgrade", "-y")
		close(done)
		if err != nil {
			fmt.Println(tui.StatusFail("apt upgrade failed"))
			return
		}
	}
	fmt.Println(tui.StatusOK("All packages upgraded"))

	// Unattended upgrades
	if tui.ConfirmWithDescription(
		"Enable unattended-upgrades?",
		"Automatically installs security patches overnight.",
	) {
		fmt.Println(tui.StatusInfo("Installing unattended-upgrades package..."))
		done := make(chan struct{})
		go tui.Spinner(done, "Installing unattended-upgrades")
		_, err := RunApt(policy, "install", "-y", "unattended-upgrades")
		close(done)
		if err != nil {
			fmt.Println(tui.StatusFail("Failed to install unattended-upgrades"))
		} else {
			fmt.Println(tui.StatusOK("Package installed"))
			fmt.Println(tui.StatusInfo("Configuring automatic updates (this can take a moment)..."))
			done = make(chan struct{})
			go tui.Spinner(done, "Running dpkg-reconfigure")
			RunShell("DEBIAN_FRONTEND=noninteractive dpkg-reconfigure -plow unattended-upgrades")
			close(done)
			fmt.Println(tui.StatusOK("Unattended upgrades enabled"))
		}
	}

	fmt.Println()
	fmt.Println(tui.StatusOK("Step 1 complete."))
}
