## dfir-cli phishing dns

Perform DNS analysis on a domain

### Synopsis

Perform DNS analysis on a domain, returning A, AAAA, MX, NS, TXT, CNAME,
and SOA records.

Input can be supplied via the --domain flag or piped via stdin.

```
dfir-cli phishing dns [flags]
```

### Examples

```
  dfir-cli phishing dns --domain example.com
  echo "example.com" | dfir-cli phishing dns
```

### Options

```
      --domain string   Domain to analyze
  -h, --help            help for dns
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

