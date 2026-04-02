## dfir-cli

DFIR Lab CLI — Threat intelligence from the command line

### Synopsis

DFIR Lab CLI wraps the DFIR Platform API to bring threat intelligence
directly into your terminal. Analyze phishing campaigns, scan for credential
exposures, and enrich indicators of compromise (IOCs) — all from the command line.

Capabilities include:
  - Phishing analysis: analyze emails and investigate URLs via phishing toolkit commands
  - Exposure scanning: search for leaked credentials across breach datasets
  - IOC enrichment: look up domains, IPs, hashes, and emails against curated threat feeds

Authenticate with an API key from https://platform.dfir-lab.ch and start investigating.

### Options

```
      --api-key string     Override API key for this invocation
      --api-url string     Override API base URL (default from config)
  -h, --help               help for dfir-cli
  -j, --json               Shorthand for --output json
      --no-color           Disable colored output
  -o, --output string      Output format: table, json, jsonl, csv (default "table")
  -p, --profile string     Named config profile (default "default")
  -q, --quiet              Minimal output
      --timeout duration   HTTP request timeout (default 1m0s)
  -v, --verbose            Show debug information (HTTP requests/responses)
```

### SEE ALSO

* [dfir-cli ai](dfir-cli_ai.md)	 - AI-powered DFIR assistant
* [dfir-cli completion](dfir-cli_completion.md)	 - Generate shell completion scripts
* [dfir-cli config](dfir-cli_config.md)	 - Manage CLI configuration and authentication
* [dfir-cli credits](dfir-cli_credits.md)	 - View API credit balance
* [dfir-cli enrichment](dfir-cli_enrichment.md)	 - Enrich IOCs across threat intelligence providers
* [dfir-cli exposure](dfir-cli_exposure.md)	 - Scan domains and IPs for external exposure
* [dfir-cli phishing](dfir-cli_phishing.md)	 - Analyse phishing emails and URLs
* [dfir-cli update](dfir-cli_update.md)	 - Check for and install updates
* [dfir-cli usage](dfir-cli_usage.md)	 - Display locally recorded API usage statistics
* [dfir-cli version](dfir-cli_version.md)	 - Print version and build information

