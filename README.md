# dfir-cli

**Digital Forensics & Incident Response CLI**

[![Go Report Card](https://goreportcard.com/badge/github.com/dfir-lab/dfir-cli)](https://goreportcard.com/report/github.com/dfir-lab/dfir-cli)
[![GitHub Release](https://img.shields.io/github/v/release/dfir-lab/dfir-cli-releases)](https://github.com/dfir-lab/dfir-cli-releases/releases/latest)
[![License: Proprietary](https://img.shields.io/badge/License-Proprietary-red.svg)](LICENSE)
[![CI](https://github.com/dfir-lab/dfir-cli/actions/workflows/ci.yml/badge.svg)](https://github.com/dfir-lab/dfir-cli/actions/workflows/ci.yml)

A powerful command-line toolkit for SOC analysts and incident responders, powered by the [DFIR Platform](https://platform.dfir-lab.ch/api-keys) API.

---

## Features

- **Phishing email analysis** -- standard and AI-enhanced detection
- **Phishing toolkit** -- DNS analysis, blacklist checks, GeoIP, Safe Browsing, CheckPhish, URLScan, URL expansion, and URL enrichment
- **IOC enrichment** -- IP, domain, URL, hash, and email lookups with multi-provider results
- **External exposure scanning** for domains with concurrent batch support
- **API usage tracking** -- view request counts and credit consumption by service
- **Multiple output formats** -- table, JSON, JSONL, and `--json`/`-j` shorthand
- **Batch processing** from files or stdin with configurable `--concurrency`
- **Shell completions** for bash, zsh, and fish
- **Cross-platform** -- macOS, Linux, Windows
- **Configurable profiles** with secure API key storage via system keychain

---

## Quick Start

```bash
# Install (macOS)
brew install dfir-lab/tap/dfir-cli

# Configure
dfir-cli config init

# Analyze a phishing email
dfir-cli phishing analyze --file suspicious.eml

# Enrich an IOC
dfir-cli enrichment lookup --ip 1.2.3.4

# Scan for exposure
dfir-cli exposure scan --domain example.com
```

---

## Installation

### macOS (Homebrew)

```bash
brew install dfir-lab/tap/dfir-cli
```

### Linux (curl)

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

### Go install

```bash
go install github.com/dfir-lab/dfir-cli/cmd/dfir-cli@latest
```

### From source

```bash
git clone https://github.com/dfir-lab/dfir-cli.git
cd dfir-cli
make build
```

---

## Configuration

Run the guided setup to get started:

```bash
dfir-cli config init
```

Set your API key directly:

```bash
dfir-cli config set api_key sk-dfir-...
```

Or use an environment variable:

```bash
export DFIR_LAB_API_KEY=sk-dfir-...
```

### Profiles

Switch between configurations using profiles:

```bash
dfir-cli config init --profile staging
dfir-cli enrichment lookup --ip 1.2.3.4 --profile staging
```

### Secure key storage

When available, API keys are stored in the system keychain (macOS Keychain, Windows Credential Manager, Linux Secret Service) rather than in plaintext. Falls back to the config file on headless systems.

### Config location

Configuration is stored at `~/.config/dfir-cli/config.yaml`.

---

## Command Reference

```
dfir-cli
├── phishing
│   ├── analyze           Analyze emails for phishing indicators (--ai for AI-enhanced)
│   ├── dns               DNS analysis on a domain
│   ├── blacklist         Check IPs against DNS blacklists
│   ├── geoip             GeoIP lookup for IPs
│   ├── safe-browsing     Check URLs against Google Safe Browsing
│   ├── checkphish        Check a URL with CheckPhish
│   ├── urlscan           Scan a URL with URLScan.io
│   ├── url-expand        Expand shortened URLs
│   └── enrich            Enrich a URL with threat intelligence
├── enrichment
│   └── lookup            Enrich IOCs across threat intelligence providers
├── exposure
│   └── scan              Scan domains for external exposure
├── credits               View cached API credit balance
├── usage                 View API usage statistics
├── config
│   ├── init              Interactive first-run setup
│   ├── set               Set a configuration value
│   ├── get               Get a configuration value
│   └── list              List all configuration values
├── version               Print version and build information
├── update                Check for and install updates
└── completion            Generate shell completion scripts
```

---

## Usage Examples

### Phishing analysis

Analyze a `.eml` file for phishing indicators:

```bash
dfir-cli phishing analyze --file suspicious.eml
```

Use AI-enhanced analysis for deeper inspection:

```bash
dfir-cli phishing analyze --file suspicious.eml --ai
```

### Phishing toolkit

```bash
# DNS analysis for a domain
dfir-cli phishing dns --domain suspicious-site.com

# Check IPs against blacklists
dfir-cli phishing blacklist --ip 203.0.113.42
dfir-cli phishing blacklist --batch ips.txt

# GeoIP lookup
dfir-cli phishing geoip --ip 203.0.113.42

# Check URLs against Google Safe Browsing
dfir-cli phishing safe-browsing --url https://suspicious-site.com

# Scan a URL with CheckPhish or URLScan.io
dfir-cli phishing checkphish --url https://suspicious-login.com
dfir-cli phishing urlscan --url https://suspicious-login.com

# Expand shortened URLs
dfir-cli phishing url-expand --url https://bit.ly/abc123

# Enrich a URL with threat intelligence
dfir-cli phishing enrich --url https://suspicious-site.com
```

### IOC enrichment

Look up a single indicator:

```bash
dfir-cli enrichment lookup --ip 1.2.3.4
dfir-cli enrichment lookup --domain evil.example.com
dfir-cli enrichment lookup --url "https://phishing.example.com/login"
dfir-cli enrichment lookup --hash 44d88612fea8a8f36de82e1278abb02f
dfir-cli enrichment lookup --email attacker@example.com
```

Batch enrichment from a file with concurrent requests:

```bash
dfir-cli enrichment lookup --batch iocs.txt --concurrency 10
```

### Exposure scanning

Scan a domain for external exposure:

```bash
dfir-cli exposure scan --domain example.com
```

Batch scan with concurrency:

```bash
dfir-cli exposure scan --batch domains.txt --concurrency 5
```

### Account

Check your cached credit balance (from the most recent API call) and usage:

```bash
dfir-cli credits
dfir-cli usage
dfir-cli usage --period 2026-02 --service enrichment
```

`dfir-cli credits` reads locally cached metadata and does not trigger a new API request.

### Piping and scripting

Pipe indicators from stdin:

```bash
cat iocs.txt | dfir-cli enrichment lookup -j
```

Extract specific fields with `jq`:

```bash
dfir-cli enrichment lookup --ip 1.2.3.4 --json | jq '.verdict'
```

---

## Output Formats

| Format | Flag                      | Description                        |
|--------|---------------------------|------------------------------------|
| Table  | *(default)*               | Human-readable tabular output      |
| JSON   | `--output json` or `-j`   | Structured JSON for scripting      |
| JSONL  | `--output jsonl`          | One JSON object per line           |
| Quiet  | `--quiet`                 | Verdict only, minimal output       |

---

## Exit Codes

| Code | Meaning                        |
|------|--------------------------------|
| 0    | Success / Clean                |
| 1    | Error                          |
| 2    | Malicious / High risk detected |
| 3    | Suspicious / Medium risk       |
| 4    | Insufficient credits           |

Exit codes make it easy to integrate `dfir-cli` into automated pipelines and alerting workflows.

---

## Shell Completions

Shell completions let you press **Tab** to auto-complete commands, subcommands, and flags.

If you installed via Homebrew, completions are already installed. If Tab completion isn't working, clear the cache and restart your terminal:

```bash
rm -f ~/.zcompdump*
exec $SHELL -l
```

For manual installation, see the [full documentation](https://github.com/dfir-lab/dfir-cli-releases#shell-completions).

---

## Building from Source

### Prerequisites

- Go 1.26 or later

### Build

```bash
git clone https://github.com/dfir-lab/dfir-cli.git
cd dfir-cli
make build
```

### Test

```bash
make test
```

### Install locally

```bash
make install
```

---

## Contributing

Contributions are welcome. Please open an issue to discuss proposed changes before submitting a pull request. See the repository's contribution guidelines for details.

---

## License

Copyright (c) 2026 DFIR Lab. All rights reserved. See the [LICENSE](LICENSE) file for details.

---

## Links

- [Platform](https://platform.dfir-lab.ch)
- [Documentation](https://platform.dfir-lab.ch/docs)
- [API Reference](https://platform.dfir-lab.ch/api)
- [Support](https://platform.dfir-lab.ch/support)
- [GitHub](https://github.com/dfir-lab/dfir-cli-releases)
