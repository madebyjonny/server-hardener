# 🔒 Server Hardener

<img width="1580" height="1300" alt="image" src="https://github.com/user-attachments/assets/c755e4ec-d2e2-43e7-b1e6-cd9ad61a274b" />


An interactive TUI for securing fresh Linux servers — built with [Charm](https://charm.sh).

Uses **Bubble Tea**, **Lip Gloss**, and **Huh** for a beautiful terminal experience
with styled prompts, spinners, multi-selects, and a color-coded audit scorecard.

![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go)
![Charm](https://img.shields.io/badge/Charm-TUI-ff69b4?logo=data:image/svg+xml;base64,...)

## Features

| Step | What it does |
|------|-------------|
| 📦 1 | **System Updates** — `apt update && upgrade`, optional unattended-upgrades |
| 👤 2 | **Create Admin User** — non-root sudo user, detects missing SSH keys |
| 🔑 3 | **SSH Hardening** — root login, password auth, port, timeouts + **lockout protection** |
| 🧱 4 | **Firewall (UFW)** — deny-by-default, multi-select for services |
| 🚨 5 | **Fail2Ban** — preset profiles (lenient/moderate/strict) or custom |
| 🧬 6 | **Kernel Hardening** — sysctl rules for SYN flood, spoofing, MITM prevention |
| 🧹 7 | **Disable Services** — finds and masks unnecessary daemons |
| 📋 8 | **Audit Summary** — color-coded scorecard with next-step suggestions |

## Lockout Protection

The SSH hardening step **will not let you disable password auth** if it can't find
any SSH keys on the server. It shows a big red warning with exact instructions to
set up keys first. Even when keys are detected, it requires a double confirmation.

The sshd config is also **validated with `sshd -t`** before restarting — if the
config is broken, it auto-restores from backup.

## Quick Start

```bash
# Build
go build -o server-hardener ./cmd/

# Deploy and run
scp server-hardener user@your-server:~/
ssh user@your-server "sudo ~/server-hardener"
```

## Flags

```
-audit      Audit only — shows your current security score, changes nothing
-dry-run    Preview mode (runs audit without modifications)
-skip N     Skip step number N (1–7)
```

## Cross-Compile

```bash
GOOS=linux GOARCH=amd64 go build -o server-hardener ./cmd/
```

## Project Structure

```
server-hardener/
├── cmd/
│   └── main.go                  # Entry point, flags, pipeline
├── internal/
│   ├── tui/
│   │   ├── styles.go            # Lip Gloss theme & formatters
│   │   ├── banner.go            # ASCII banner & spinner
│   │   └── prompts.go           # Huh-based confirm/input/select wrappers
│   └── steps/
│       ├── runner.go            # Shell exec, file helpers, safety checks
│       ├── 01_updates.go        # System updates
│       ├── 02_user.go           # User creation + SSH key detection
│       ├── 03_ssh.go            # SSH hardening with lockout protection
│       ├── 04_firewall.go       # UFW setup with multi-select
│       ├── 05_fail2ban.go       # Fail2Ban with preset profiles
│       ├── 06_kernel.go         # Sysctl hardening
│       ├── 07_services.go       # Disable unnecessary services
│       └── 08_audit.go          # Scorecard & recommendations
├── go.mod
└── README.md
```

## Dependencies

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) — TUI framework
- [Lip Gloss](https://github.com/charmbracelet/lipgloss) — Styling
- [Huh](https://github.com/charmbracelet/huh) — Interactive forms
- [Charm Log](https://github.com/charmbracelet/log) — Structured logging

## License

MIT
