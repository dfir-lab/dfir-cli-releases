## dfir-cli phishing safe-browsing

Check URLs against Google Safe Browsing

### Synopsis

Check one or more URLs against the Google Safe Browsing database to determine
if they are associated with known threats such as malware, phishing, or
unwanted software.

Input methods:
  --url https://example.com                              Single URL check
  --batch urls.txt                                       File with one URL per line
  echo "https://example.com" | dfir-cli phishing safe-browsing   Pipe via stdin

```
dfir-cli phishing safe-browsing [flags]
```

### Examples

```
  dfir-cli phishing safe-browsing --url https://example.com
  dfir-cli phishing safe-browsing --batch urls.txt
  echo "https://example.com" | dfir-cli phishing safe-browsing
```

### Options

```
      --batch string   File with one URL per line (use - for stdin)
  -h, --help           help for safe-browsing
      --url string     URL to check
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

