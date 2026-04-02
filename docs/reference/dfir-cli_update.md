## dfir-cli update

Check for and install updates

### Synopsis

Check for a newer version of dfir-cli and install it.

Without flags, dfir-cli downloads and installs the latest version
automatically. Homebrew and Scoop installations are detected and
updated through their respective package managers.

Use --check to only check for updates without installing.

```
dfir-cli update [flags]
```

### Options

```
      --check   Only check for updates, don't install
  -h, --help    help for update
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

