## dfir-cli phishing url-expand

Expand shortened URLs

### Synopsis

Expand a shortened URL to reveal the final destination and redirect chain.

Input methods:
  --url https://bit.ly/abc123                                   Expand a single URL
  echo "https://bit.ly/abc123" | dfir-cli phishing url-expand   Pipe via stdin

Cost: 1 credit per request.

```
dfir-cli phishing url-expand [flags]
```

### Examples

```
  dfir-cli phishing url-expand --url https://bit.ly/abc123
  echo "https://t.co/xyz" | dfir-cli phishing url-expand
```

### Options

```
  -h, --help         help for url-expand
      --url string   Shortened URL to expand
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

