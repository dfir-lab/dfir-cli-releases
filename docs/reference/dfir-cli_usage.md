## dfir-cli usage

Display locally recorded API usage statistics

### Synopsis

Display locally recorded API usage statistics including request counts,
credit consumption, and a breakdown by service.

The period flag controls which billing period to display:
  current   — the current month (default)
  previous  — the previous month
  YYYY-MM   — a specific month (e.g. 2026-03)

Optionally filter by service (phishing, exposure, enrichment, ai) and limit
the number of top operations shown.

Usage is built from successful dfir-cli API calls recorded on this machine.
The command does not make a network request.

```
dfir-cli usage [flags]
```

### Examples

```
  dfir-cli usage
  dfir-cli usage --period previous
  dfir-cli usage --period 2026-01 --service enrichment
  dfir-cli usage --top 5
```

### Options

```
  -h, --help             help for usage
      --period string    Billing period: current, previous, or YYYY-MM (default "current")
      --service string   Filter by service: phishing, exposure, enrichment, ai
      --top int          Number of top operations to display (default 10)
```

### Options inherited from parent commands

```
      --api-key string     Override API key for this invocation
      --api-url string     Override API base URL (default from config)
  -j, --json               Shorthand for --output json
      --no-color           Disable colored output
  -o, --output string      Output format: table, json, jsonl, csv (default "table")
  -p, --profile string     Named config profile (default "default")
  -q, --quiet              Minimal output
      --timeout duration   HTTP request timeout (default 1m0s)
  -v, --verbose            Show debug information (HTTP requests/responses)
```

### SEE ALSO

* [dfir-cli](dfir-cli.md)	 - DFIR Lab CLI — Threat intelligence from the command line

