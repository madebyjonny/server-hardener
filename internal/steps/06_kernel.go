package steps

import (
	"fmt"

	"server-hardener/internal/tui"
)

func HardenKernel() {
	fmt.Println(tui.StepHeader(6, "Kernel & Network Hardening"))
	fmt.Println(tui.StatusInfo("Sysctl tweaks to prevent spoofing, SYN floods, and more."))

	if !tui.Confirm("Apply kernel hardening?") {
		fmt.Println(tui.StatusSkip("kernel hardening"))
		return
	}

	sysctlConf := `# ─── server-hardener sysctl rules ──────────────────────────

# Prevent IP spoofing
net.ipv4.conf.all.rp_filter = 1
net.ipv4.conf.default.rp_filter = 1

# Ignore ICMP redirects (prevent MITM)
net.ipv4.conf.all.accept_redirects = 0
net.ipv4.conf.default.accept_redirects = 0
net.ipv6.conf.all.accept_redirects = 0

# Don't send ICMP redirects (we're not a router)
net.ipv4.conf.all.send_redirects = 0
net.ipv4.conf.default.send_redirects = 0

# Ignore broadcast pings (smurf attack prevention)
net.ipv4.icmp_echo_ignore_broadcasts = 1

# SYN flood protection
net.ipv4.tcp_syncookies = 1
net.ipv4.tcp_max_syn_backlog = 2048
net.ipv4.tcp_synack_retries = 2

# Log suspicious packets
net.ipv4.conf.all.log_martians = 1
net.ipv4.conf.default.log_martians = 1

# Disable source routing
net.ipv4.conf.all.accept_source_route = 0
net.ipv6.conf.all.accept_source_route = 0

# Restrict unprivileged BPF and ptrace
kernel.unprivileged_bpf_disabled = 1
kernel.yama.ptrace_scope = 1

# Restrict kernel logs to root
kernel.dmesg_restrict = 1
`

	if err := WriteFile("/etc/sysctl.d/99-hardening.conf", sysctlConf, 0644); err != nil {
		fmt.Println(tui.StatusFail("Failed to write sysctl config"))
		return
	}
	fmt.Println(tui.StatusOK("Wrote /etc/sysctl.d/99-hardening.conf"))

	done := make(chan struct{})
	go tui.Spinner(done, "Applying kernel parameters")
	_, err := Run("sysctl", "--system")
	close(done)
	if err != nil {
		fmt.Println(tui.StatusFail("Failed to reload sysctl"))
	} else {
		fmt.Println(tui.StatusOK("Kernel parameters applied"))
	}
}
