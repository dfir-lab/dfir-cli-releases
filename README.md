# dfir-cli

**Digital Forensics & Incident Response CLI**

[![GitHub Release](https://img.shields.io/github/v/release/dfir-lab/dfir-cli-releases)](https://github.com/dfir-lab/dfir-cli-releases/releases/latest)
[![License: Proprietary](https://img.shields.io/badge/License-Proprietary-red.svg)](LICENSE)

A powerful command-line toolkit for SOC analysts and incident responders, powered by the [DFIR Lab](https://platform.dfir-lab.ch) API. Analyze phishing emails, enrich indicators of compromise, and scan for external exposure — all from your terminal.

---

## Installation

### macOS (Homebrew)

```bash
brew install dfir-lab/tap/dfir-cli
```

### Linux

```bash
curl -fsSL https://raw.githubusercontent.com/dfir-lab/dfir-cli-releases/main/install.sh | sh
```

### Linux (APT / deb)

Download the `.deb` package from the [latest release](https://github.com/dfir-lab/dfir-cli-releases/releases/latest) and install:

```bash
sudo dpkg -i dfir-cli_*.deb
```

### Windows (Scoop)

```powershell
scoop bucket add dfir-lab https://github.com/dfir-lab/scoop-bucket.git
scoop install dfir-cli
```

### Windows (PowerShell)

```powershell
iwr https://raw.githubusercontent.com/dfir-lab/dfir-cli-releases/main/install.ps1 | iex
```

### Verify installation

```bash
dfir-cli --version
```

---

## Getting Started

### Step 1: Configure your API key

Get your API key from the [DFIR Lab Platform](https://platform.dfir-lab.ch) dashboard, then run:

```bash
dfir-cli config init
```

You will be prompted to enter your API key (input is hidden for security):

```
Welcome to dfir-cli configuration!
----------------------------------

Setting up profile: default

Enter your API key: ************************************

Configuration saved successfully! (profile: default)

Next steps:
  dfir-cli config list                    Show current configuration
  dfir-cli config set KEY VALUE           Change a setting
  dfir-cli enrichment lookup --ip 8.8.8.8 Run your first enrichment
```

Alternatively, set it via environment variable:

```bash
export DFIR_LAB_API_KEY=sk-dfir-...
```

### Step 2: Run your first analysis

```bash
dfir-cli enrichment lookup --ip 8.8.8.8
```

---

## Features

### Phishing Email Analysis

Analyze suspicious emails for spoofing indicators, malicious links, and social engineering tactics.

**Standard analysis:**

```bash
dfir-cli phishing analyze --file suspicious.eml
```

Example output:

```
Phishing Analysis

  Verdict:    MALICIOUS
  Score:      [████████████████████░░░░] 87/100
  Summary:    Spoofed sender with credential harvesting link

  Authentication Results:
    SPF:       FAIL
    DKIM:      FAIL
    DMARC:     FAIL

  Key Findings:
    - SPF alignment failure: envelope sender does not match header from
    - DKIM signature missing or invalid
    - Link text mismatch: displayed URL differs from actual destination
    - Urgency language detected in email body

  Extracted IOCs:
    URL     https://login-secure.example.xyz/verify
    IP      185.220.101.34
    Domain  login-secure.example.xyz
```

**AI-enhanced analysis** (uses 10 credits for deeper inspection with Claude):

```bash
dfir-cli phishing analyze --file suspicious.eml --ai
```

**Analyze raw email headers from stdin:**

```bash
cat headers.txt | dfir-cli phishing analyze --type headers
```

### IOC Enrichment

Enrich indicators of compromise across multiple threat intelligence providers including VirusTotal, AbuseIPDB, Shodan, URLhaus, and more.

**Look up a single indicator:**

```bash
# IP address
dfir-cli enrichment lookup --ip 185.220.101.34

# Domain
dfir-cli enrichment lookup --domain evil.example.com

# URL
dfir-cli enrichment lookup --url "https://phishing.example.com/login"

# File hash (MD5, SHA-1, or SHA-256)
dfir-cli enrichment lookup --hash 44d88612fea8a8f36de82e1278abb02f

# Email address
dfir-cli enrichment lookup --email attacker@example.com
```

Example output:

```
IOC Enrichment: 185.220.101.34 (IP)

  Verdict:    MALICIOUS
  Score:      [██████████████████░░░░░░] 76/100
  Consensus:  3/4 providers flagged

  Provider       Verdict      Score   Details
  VirusTotal     Malicious     89     14/90 engines flagged
  AbuseIPDB      Malicious     95     847 reports, ISP: Tor Exit
  Shodan         Suspicious    65     8 open ports, 3 CVEs
  GreyNoise      Benign         5     Known scanner

  Credits: 3 used, 97 remaining
```

**Auto-detect IOC type:**

```bash
dfir-cli enrichment lookup --ioc 185.220.101.34
dfir-cli enrichment lookup --ioc evil.example.com
```

**Batch enrichment from a file** (one indicator per line):

```bash
dfir-cli enrichment lookup --batch iocs.txt
```

**Filter results by provider or minimum score:**

```bash
dfir-cli enrichment lookup --ip 1.2.3.4 --providers VirusTotal,AbuseIPDB
dfir-cli enrichment lookup --ip 1.2.3.4 --min-score 50
```

### Exposure Scanning

Scan domains for external attack surface exposure. The scanner queries 11 intelligence providers in parallel to enumerate subdomains, discover open ports, grade SSL/TLS configurations, and match services against known vulnerabilities.

```bash
dfir-cli exposure scan --domain example.com
```

Example output:

```
Exposure Scan: example.com

  Risk Level:  LOW
  Risk Score:  [████░░░░░░░░░░░░░░░░░░░] 18/100
  SSL Grade:   A+
  Duration:    12.4s

  Credits: 10 used, 87 remaining
```

**Batch scanning:**

```bash
dfir-cli exposure scan --batch domains.txt
```

### Credit Balance

Check your remaining API credits without consuming any:

```bash
dfir-cli credits
```

```
Credit Balance (as of last API call)

  Credits Remaining:  87
  Last Used:          10
  Last Request:       2026-03-29T10:53:24Z

  Note: Credit balance is updated after each API operation.
```

---

## Output Formats

dfir-cli supports multiple output formats for different use cases:

| Format | Flag | Use Case |
|--------|------|----------|
| Table | *(default)* | Human-readable terminal output with colors |
| JSON | `--output json` | Structured output for scripting and `jq` |
| JSONL | `--output jsonl` | One JSON object per line for streaming |
| Quiet | `--quiet` | Verdict and score only, for shell scripts |

**JSON output for scripting:**

```bash
dfir-cli enrichment lookup --ip 1.2.3.4 --output json | jq '.results[0].verdict'
```

**Pipe indicators and get JSON results:**

```bash
cat iocs.txt | dfir-cli enrichment lookup --output json > results.json
```

**Quiet mode for shell scripts:**

```bash
dfir-cli enrichment lookup --ip 1.2.3.4 --quiet
# Output: MALICIOUS 76 185.220.101.34
```

**Non-TTY auto-detection:** When piped (not connected to a terminal), dfir-cli automatically disables colors and switches to JSON output.

---

## Exit Codes

dfir-cli uses meaningful exit codes for CI/CD integration and automated pipelines:

| Code | Meaning |
|------|---------|
| 0 | Success / Clean verdict |
| 1 | Error (invalid input, network failure, etc.) |
| 2 | Malicious / High risk detected |
| 3 | Suspicious / Medium risk detected |
| 4 | Insufficient credits |

**Example: alert on malicious results in CI:**

```bash
dfir-cli enrichment lookup --ip "$SUSPICIOUS_IP" --quiet
if [ $? -eq 2 ]; then
  echo "ALERT: Malicious indicator detected!"
  # trigger incident response workflow
fi
```

---

## Configuration

### Profiles

Manage multiple configurations for different environments:

```bash
# Create a staging profile
dfir-cli config init --profile staging

# Use a specific profile
dfir-cli enrichment lookup --ip 1.2.3.4 --profile staging

# List all settings for a profile
dfir-cli config list --profile staging
```

### Config commands

```bash
dfir-cli config list                          # Show all settings
dfir-cli config get api-key                   # Show masked API key
dfir-cli config get api-key --unmask          # Show full API key
dfir-cli config set output-format json        # Change default output
dfir-cli config set timeout 120s              # Set request timeout
dfir-cli config set no-color true             # Disable colors
```

### API key precedence

The API key is resolved in this order (highest priority first):

1. `--api-key` flag
2. `DFIR_LAB_API_KEY` environment variable
3. Config file (`~/.config/dfir-cli/config.yaml`)

---

## Updating

Check for new versions:

```bash
dfir-cli update --check
```

Update to the latest version:

```bash
dfir-cli update
```

dfir-cli also checks for updates automatically in the background (once every 24 hours) and prints a notice when a new version is available.

---

## Shell Completions

Shell completions let you press **Tab** to auto-complete commands, subcommands, and flags — for example, typing `dfir-cli enr` + Tab completes to `dfir-cli enrichment`.

### Homebrew (automatic)

If you installed via Homebrew, completions are **already installed**. If Tab completion isn't working, clear the cache and restart your terminal:

```bash
rm -f ~/.zcompdump*
exec $SHELL -l
```

### Manual installation

If you installed without Homebrew, set up completions for your shell:

**Bash:**

```bash
dfir-cli completion bash > /usr/local/etc/bash_completion.d/dfir-cli
source ~/.bashrc
```

**Zsh (Oh My Zsh):**

```bash
dfir-cli completion zsh > ~/.oh-my-zsh/completions/_dfir-cli
rm -f ~/.zcompdump*
exec $SHELL -l
```

If `~/.oh-my-zsh/completions` doesn't exist, create it first: `mkdir -p ~/.oh-my-zsh/completions`

**Zsh (without Oh My Zsh):**

```bash
mkdir -p ~/.zsh/completions
dfir-cli completion zsh > ~/.zsh/completions/_dfir-cli
```

Then add this to your `~/.zshrc` (before `compinit`):

```bash
fpath=(~/.zsh/completions $fpath)
autoload -Uz compinit && compinit
```

**Fish:**

```bash
dfir-cli completion fish > ~/.config/fish/completions/dfir-cli.fish
```

---

## Global Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--api-key` | | Override API key for this invocation |
| `--api-url` | | Override API base URL |
| `--output` | `-o` | Output format: table, json, jsonl, csv |
| `--no-color` | | Disable colored output |
| `--verbose` | `-v` | Show HTTP request/response debug info |
| `--quiet` | `-q` | Minimal output (verdict only) |
| `--timeout` | | HTTP request timeout (default 60s) |
| `--profile` | `-p` | Named config profile (default "default") |

---

## Credit Costs

| Operation | Credits |
|-----------|---------|
| IOC Enrichment (single lookup) | 5 |
| IOC Enrichment (batch, per IOC) | 3 |
| Phishing Analysis | 1 |
| Phishing Analysis (AI-enhanced) | 10 |
| Exposure Scan | 10 (0 if cached) |
| DNS Lookup | 1 |
| Blacklist Check | 1 |
| GeoIP Lookup | 1 |
| URL Expand | 1 |
| Safe Browsing | 2 |
| CheckPhish | 2 |
| Phishing Enrich | 2 |
| URLScan | 3 |

Credit costs are subject to change. Check the [DFIR Lab Platform](https://platform.dfir-lab.ch) for the latest pricing.

---

## Feedback and Support

We'd love to hear from you. If you encounter a bug, have a feature request, or want to suggest an improvement:

- **Report a bug** -- Open an issue at [dfir-lab/dfir-cli-releases/issues](https://github.com/dfir-lab/dfir-cli-releases/issues) with steps to reproduce, expected vs. actual behavior, and your dfir-cli version (`dfir-cli --version`).
- **Request a feature** -- Open an issue describing the use case and how you'd like it to work. We prioritize based on community demand.
- **Ask a question** -- If something isn't clear in the documentation, open an issue and we'll improve it.
- **Contact support** -- For account or billing issues, reach out at [platform.dfir-lab.ch/contact](https://platform.dfir-lab.ch/contact).

Your feedback helps us build a better tool for the DFIR community.

---

## Links

- [DFIR Lab Platform](https://platform.dfir-lab.ch)
- [API Documentation](https://platform.dfir-lab.ch/docs)
- [Release Downloads](https://github.com/dfir-lab/dfir-cli-releases/releases)
- [Homebrew Tap](https://github.com/dfir-lab/homebrew-tap)
- [Report an Issue](https://github.com/dfir-lab/dfir-cli-releases/issues)

---

## License

Copyright (c) 2026 DFIR Lab. All rights reserved.

This software is proprietary. The compiled binaries are provided for use under the terms described at [platform.dfir-lab.ch/terms](https://platform.dfir-lab.ch/terms). Redistribution, reverse engineering, or modification of the binaries is prohibited without prior written consent.
