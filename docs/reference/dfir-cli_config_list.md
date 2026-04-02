## dfir-cli config list

List all configuration values

```
dfir-cli config list [flags]
```

### Options

```
  -h, --help     help for list
      --unmask   Show full API key without masking
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

* [dfir-cli config](dfir-cli_config.md)	 - Manage CLI configuration and authentication

