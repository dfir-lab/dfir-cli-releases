# dfir-cli

**Digital Forensics & Incident Response CLI**

[![Go Report Card](https://goreportcard.com/badge/github.com/ForeGuards/dfir-cli)](https://goreportcard.com/report/github.com/ForeGuards/dfir-cli)
[![GitHub Release](https://img.shields.io/github/v/release/ForeGuards/dfir-cli)](https://github.com/ForeGuards/dfir-cli/releases/latest)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![CI](https://github.com/ForeGuards/dfir-cli/actions/workflows/ci.yml/badge.svg)](https://github.com/ForeGuards/dfir-cli/actions/workflows/ci.yml)

A powerful command-line toolkit for SOC analysts and incident responders, powered by the [DFIR Lab](https://dfir-lab.ch) API.

---

## Features

- **Phishing email analysis** -- standard and AI-enhanced detection
- **IOC enrichment** -- IP, domain, URL, hash, and email lookups with multi-provider results
- **External exposure scanning** for domains
- **Multiple output formats** -- table, JSON, JSONL
- **Batch processing** from files or stdin
- **Shell completions** for bash, zsh, and fish
- **Cross-platform** -- macOS, Linux, Windows
- **Configurable profiles** with secure API key storage

---

## Quick Start

```bash
# Install (macOS)
brew install ForeGuards/tap/dfir-cli

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
brew install ForeGuards/tap/dfir-cli
```

### Linux (curl)

```bash
curl -fsSL https://dfir-lab.ch/install.sh | sh
```

### Linux (APT / deb)

Download the `.deb` package from the [latest release](https://github.com/ForeGuards/dfir-cli/releases/latest) and install:

```bash
sudo dpkg -i dfir-cli_*.deb
```

### Windows (Scoop)

```powershell
scoop bucket add foreguards https://github.com/ForeGuards/scoop-bucket.git
scoop install dfir-cli
```

### Windows (PowerShell)

```powershell
iwr https://dfir-lab.ch/install.ps1 | iex
```

### Go install

```bash
go install github.com/ForeGuards/dfir-cli/cmd/dfir-cli@latest
```

### From source

```bash
git clone https://github.com/ForeGuards/dfir-cli.git
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

### Config location

Configuration is stored at `~/.config/dfir-cli/config.yaml`.

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

### IOC enrichment

Look up a single indicator:

```bash
dfir-cli enrichment lookup --ip 1.2.3.4
dfir-cli enrichment lookup --domain evil.example.com
dfir-cli enrichment lookup --url "https://phishing.example.com/login"
dfir-cli enrichment lookup --hash 44d88612fea8a8f36de82e1278abb02f
dfir-cli enrichment lookup --email attacker@example.com
```

Batch enrichment from a file (one indicator per line):

```bash
dfir-cli enrichment lookup --batch iocs.txt
```

### Exposure scanning

Scan a domain for external exposure:

```bash
dfir-cli exposure scan --domain example.com
```

### Credits

Check your remaining API credits:

```bash
dfir-cli credits
```

### Piping and scripting

Pipe indicators from stdin:

```bash
cat iocs.txt | dfir-cli enrichment lookup --output json
```

Extract specific fields with `jq`:

```bash
dfir-cli enrichment lookup --ip 1.2.3.4 --output json | jq '.verdict'
```

---

## Output Formats

| Format | Flag              | Description                        |
|--------|-------------------|------------------------------------|
| Table  | *(default)*       | Human-readable tabular output      |
| JSON   | `--output json`   | Structured JSON for scripting      |
| JSONL  | `--output jsonl`  | One JSON object per line           |
| Quiet  | `--quiet`         | Verdict only, minimal output       |

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

### Bash

```bash
dfir-cli completion bash > /etc/bash_completion.d/dfir-cli
```

### Zsh

```bash
dfir-cli completion zsh > "${fpath[1]}/_dfir-cli"
```

### Fish

```bash
dfir-cli completion fish > ~/.config/fish/completions/dfir-cli.fish
```

---

## Building from Source

### Prerequisites

- Go 1.26 or later

### Build

```bash
git clone https://github.com/ForeGuards/dfir-cli.git
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

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

---

## Links

- [Website](https://dfir-lab.ch)
- [Documentation](https://dfir-lab.ch/docs)
- [API Reference](https://dfir-lab.ch/api)
- [Support](https://dfir-lab.ch/support)
- [GitHub](https://github.com/ForeGuards/dfir-cli)
