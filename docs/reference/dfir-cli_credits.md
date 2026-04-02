## dfir-cli credits

View API credit balance

### Synopsis

Display the credit balance from the most recent API call.

Credit information is updated automatically after every API operation. This
command reads the cached balance — it does not make an API call and therefore
does not consume credits.

```
dfir-cli credits [flags]
```

### Options

```
  -h, --help   help for credits
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

