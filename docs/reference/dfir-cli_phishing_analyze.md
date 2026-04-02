## dfir-cli phishing analyze

Analyze an email for phishing indicators

### Synopsis

Analyze a raw email, .eml file, or email headers for phishing indicators.

Input methods:
  --file suspicious.eml                         Read from an .eml file
  --raw "From: attacker@evil.com\n..."          Inline raw email content
  cat email.eml | dfir-cli phishing analyze     Pipe via stdin

Use --ai for AI-enhanced analysis (costs 10 credits instead of 1).

```
dfir-cli phishing analyze [flags]
```

### Options

```
      --ai            Use AI-enhanced analysis (10 credits)
      --file string   Path to an .eml email file
  -h, --help          help for analyze
      --raw string    Raw email content as a string
      --type string   Input type: headers, eml, raw (auto-detected if omitted)
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

