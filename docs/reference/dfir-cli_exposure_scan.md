## dfir-cli exposure scan

Scan domains for external exposure

### Synopsis

Scan one or more domains for external exposure.

Input methods:
  --domain example.com                          Single domain
  echo "example.com" | dfir-cli exposure scan   Stdin
  --batch domains.txt                           Batch file (one domain per line, use - for stdin)

The scan may take up to 3 minutes per target depending on the providers
queried. A spinner is displayed while waiting.

```
dfir-cli exposure scan [flags]
```

### Examples

```
  dfir-cli exposure scan --domain example.com
  dfir-cli exposure scan --domain example.com --target-type domain
  dfir-cli exposure scan --batch domains.txt
  echo "example.com" | dfir-cli exposure scan
```

### Options

```
      --batch string         File with one domain per line (use - for stdin)
      --concurrency int      Parallel requests for batch mode (1-20) (default 5)
      --domain string        Target domain to scan
  -h, --help                 help for scan
      --target-type string   Target type hint: domain, ip, auto (default "auto")
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

* [dfir-cli exposure](dfir-cli_exposure.md)	 - Scan domains and IPs for external exposure

