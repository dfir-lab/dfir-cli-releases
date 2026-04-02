## dfir-cli phishing urlscan

Scan a URL with URLScan.io

### Synopsis

Submit a URL to URLScan.io for analysis.

Costs 3 credits per scan.

Input methods:
  --url https://example.com
  echo "https://example.com" | dfir-cli phishing urlscan

```
dfir-cli phishing urlscan [flags]
```

### Options

```
  -h, --help         help for urlscan
      --url string   URL to scan (required)
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

* [dfir-cli phishing](dfir-cli_phishing.md)	 - Analyse phishing emails and URLs

