## dfir-cli phishing blacklist

Check IPs against DNS blacklists

### Synopsis

Check one or more IP addresses against DNS blacklists (DNSBLs).

Input can be supplied via --ip for a single address, --batch for a file with
one IP per line, or piped via stdin.

```
dfir-cli phishing blacklist [flags]
```

### Examples

```
  dfir-cli phishing blacklist --ip 1.2.3.4
  dfir-cli phishing blacklist --batch ips.txt
  echo "1.2.3.4" | dfir-cli phishing blacklist
```

### Options

```
      --batch string   File with one IP per line (use - for stdin)
  -h, --help           help for blacklist
      --ip string      Single IP address to check
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

