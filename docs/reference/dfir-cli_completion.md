## dfir-cli completion

Generate shell completion scripts

### Synopsis

Generate shell completion scripts for dfir-cli.

To install completions:

  bash:
    $ dfir-cli completion bash > /etc/bash_completion.d/dfir-cli

  zsh:
    $ dfir-cli completion zsh > "${fpath[1]}/_dfir-cli"

  fish:
    $ dfir-cli completion fish > ~/.config/fish/completions/dfir-cli.fish

  powershell:
    PS> dfir-cli completion powershell | Out-String | Invoke-Expression

```
dfir-cli completion [bash|zsh|fish|powershell] [flags]
```

### Options

```
  -h, --help   help for completion
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

