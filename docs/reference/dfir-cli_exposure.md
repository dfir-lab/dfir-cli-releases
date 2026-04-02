## dfir-cli exposure

Scan domains and IPs for external exposure

### Synopsis

Scan domains and IPs for external exposure using the DFIR Lab API.

The exposure module probes a target for SSL/TLS configuration, DNS records,
and other publicly visible attack surface indicators, then computes an
overall risk score.

### Options

```
  -h, --help   help for exposure
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
* [dfir-cli exposure scan](dfir-cli_exposure_scan.md)	 - Scan domains for external exposure

