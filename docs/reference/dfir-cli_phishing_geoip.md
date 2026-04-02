## dfir-cli phishing geoip

Look up geographic location of IPs

### Synopsis

Look up the geographic location of one or more IP addresses using GeoIP data.

Input methods:
  --ip 1.2.3.4                                   Single IP lookup
  --batch ips.txt                                 File with one IP per line
  echo "1.2.3.4" | dfir-cli phishing geoip       Pipe via stdin

```
dfir-cli phishing geoip [flags]
```

### Examples

```
  dfir-cli phishing geoip --ip 8.8.8.8
  dfir-cli phishing geoip --batch ips.txt
  echo "1.2.3.4" | dfir-cli phishing geoip
```

### Options

```
      --batch string   File with one IP per line (use - for stdin)
  -h, --help           help for geoip
      --ip string      IP address to look up
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

