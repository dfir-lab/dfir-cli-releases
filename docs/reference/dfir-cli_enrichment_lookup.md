## dfir-cli enrichment lookup

Enrich IOCs across threat intelligence providers

### Synopsis

Look up indicators of compromise (IOCs) across multiple threat intelligence
providers. Supports IPs, domains, URLs, hashes, and email addresses.

Input can be supplied via typed flags, the generic --ioc flag, a batch file,
or stdin.

```
dfir-cli enrichment lookup [flags]
```

### Examples

```
  dfir-cli enrichment lookup --ip 1.2.3.4
  dfir-cli enrichment lookup --domain evil.com
  dfir-cli enrichment lookup --ioc evil.com
  dfir-cli enrichment lookup --batch iocs.txt --type ip
  echo "1.2.3.4" | dfir-cli enrichment lookup --type ip
```

### Options

```
      --batch string       File with one IOC per line (use - for stdin)
      --concurrency int    Parallel requests for batch mode (1-20) (default 5)
      --domain string      Look up a domain
      --email string       Look up an email address
      --hash string        Look up a file hash
  -h, --help               help for lookup
      --ioc string         Look up any IOC (auto-detect type)
      --ip string          Look up an IP address
      --min-score int      Only show providers above this score (0-100)
      --providers string   Comma-separated provider filter
      --type string        Force IOC type: ip, domain, url, hash, email
      --url string         Look up a URL
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

* [dfir-cli enrichment](dfir-cli_enrichment.md)	 - Enrich IOCs across threat intelligence providers

