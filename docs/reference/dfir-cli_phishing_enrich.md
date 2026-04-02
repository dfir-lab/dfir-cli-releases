## dfir-cli phishing enrich

Enrich a URL with threat intelligence

### Synopsis

Enrich a URL with threat intelligence data including risk scoring,
categorization, and threat intel provider results.

Input methods:
  --url https://suspicious.example.com                       Enrich a single URL
  echo "https://suspicious.example.com" | dfir-cli phishing enrich   Pipe via stdin

Cost: 2 credits per request.

```
dfir-cli phishing enrich [flags]
```

### Examples

```
  dfir-cli phishing enrich --url https://suspicious.example.com
  echo "https://evil.com/phish" | dfir-cli phishing enrich
```

### Options

```
  -h, --help         help for enrich
      --url string   URL to enrich with threat intelligence
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

