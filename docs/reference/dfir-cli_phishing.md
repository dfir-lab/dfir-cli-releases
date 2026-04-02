## dfir-cli phishing

Analyse phishing emails and URLs

### Options

```
  -h, --help   help for phishing
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
* [dfir-cli phishing analyze](dfir-cli_phishing_analyze.md)	 - Analyze an email for phishing indicators
* [dfir-cli phishing blacklist](dfir-cli_phishing_blacklist.md)	 - Check IPs against DNS blacklists
* [dfir-cli phishing checkphish](dfir-cli_phishing_checkphish.md)	 - Check a URL with CheckPhish
* [dfir-cli phishing dns](dfir-cli_phishing_dns.md)	 - Perform DNS analysis on a domain
* [dfir-cli phishing enrich](dfir-cli_phishing_enrich.md)	 - Enrich a URL with threat intelligence
* [dfir-cli phishing geoip](dfir-cli_phishing_geoip.md)	 - Look up geographic location of IPs
* [dfir-cli phishing safe-browsing](dfir-cli_phishing_safe-browsing.md)	 - Check URLs against Google Safe Browsing
* [dfir-cli phishing url-expand](dfir-cli_phishing_url-expand.md)	 - Expand shortened URLs
* [dfir-cli phishing urlscan](dfir-cli_phishing_urlscan.md)	 - Scan a URL with URLScan.io

