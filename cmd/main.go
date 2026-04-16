package main

import (
	"flag"
	"fmt"
	"os"

	"server-hardener/internal/steps"
	"server-hardener/internal/tui"
)

func main() {
	auditOnly := flag.Bool("audit", false, "Run audit check only (no changes)")
	skipStep := flag.Int("skip", 0, "Skip a specific step number")
	dryRun := flag.Bool("dry-run", false, "Show what would be done without making changes")
	flag.Parse()

	fmt.Print(tui.Banner())

	// Root check
	if !steps.IsRoot() {
		fmt.Println(tui.DangerBox.Render(
			tui.DangerStyle.Render("Not running as root!\n\n") +
				"This tool needs root to modify system config.\n" +
				"Run with: " + tui.InfoStyle.Render("sudo ./server-hardener"),
		))
		os.Exit(1)
	}

	if *dryRun {
		fmt.Println(tui.CalloutBox.Render(
			tui.WarnStyle.Render("DRY RUN MODE\n") +
				"No changes will be made. This is a preview only.",
		))
		fmt.Println()
		steps.Preflight()
		steps.AuditSummary()
		return
	}

	if *auditOnly {
		steps.AuditSummary()
		return
	}

	// ─── Preflight safety gate ──────────────────────────
	// Must pass before any changes. If it fails, dangerous
	// options are hard-blocked (not just warned about).
	if !steps.Preflight() {
		os.Exit(1)
	}

	// ─── Step pipeline ──────────────────────────────────
	type step struct {
		num int
		fn  func()
	}

	pipeline := []step{
		{1, steps.UpdateSystem},
		{2, steps.CreateUser},
		{3, steps.HardenSSH},
		{4, steps.SetupFirewall},
		{5, steps.SetupFail2Ban},
		{6, steps.HardenKernel},
		{7, steps.DisableServices},
	}

	for _, s := range pipeline {
		if *skipStep == s.num {
			fmt.Println(tui.StatusSkip(fmt.Sprintf("step %d (skipped via --skip flag)", s.num)))
			continue
		}
		s.fn()
	}

	// Always finish with audit
	steps.AuditSummary()
}
