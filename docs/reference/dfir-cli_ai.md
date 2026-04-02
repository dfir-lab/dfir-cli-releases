## dfir-cli ai

AI-powered DFIR assistant

### Synopsis

Interactive AI assistant specialized in digital forensics and incident response.

Ask questions about forensic artifacts, pipe in tool output for analysis,
or start an interactive chat session for extended investigations.

The AI assistant only answers DFIR-related questions. Topics outside
digital forensics and incident response will be declined.

Requires a Starter, Professional, or Enterprise plan.

Examples:
  # Ask a one-shot question
  dfir-cli ai "What Windows event IDs indicate lateral movement?"

  # Pipe tool output for analysis
  vol.py -f memory.dmp windows.pslist | dfir-cli ai "Analyze for anomalies"

  # Pipe dfir-cli results for deeper analysis
  dfir-cli enrichment lookup --ip 1.2.3.4 -j | dfir-cli ai "Explain these results"

  # Start an interactive chat session
  dfir-cli ai chat

  # Use a specific model
  dfir-cli ai --model haiku "Quick: what is shimcache?"

```
dfir-cli ai [question] [flags]
```

### Options

```
  -h, --help           help for ai
      --model string   AI model to use: "haiku" (fast) or "sonnet" (thorough). Default from config or "sonnet"
      --no-stream      Disable streaming (wait for complete response)
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
* [dfir-cli ai chat](dfir-cli_ai_chat.md)	 - Start an interactive AI chat session

