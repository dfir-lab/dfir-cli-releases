## dfir-cli ai chat

Start an interactive AI chat session

### Synopsis

Start an interactive DFIR AI assistant chat session.

The assistant specializes in digital forensics and incident response.
Conversation history is maintained within the session.

Special commands:
  /help       Show available commands
  /clear      Clear conversation history
  /model NAME Switch model (haiku or sonnet)
  /exit       Exit the chat session
  Ctrl+C      Cancel current response

Requires a Starter, Professional, or Enterprise plan.

```
dfir-cli ai chat [flags]
```

### Options

```
  -h, --help   help for chat
```

### Options inherited from parent commands

```
      --api-key string     Override API key for this invocation
      --api-url string     Override API base URL (default from config)
  -j, --json               Shorthand for --output json
      --model string       AI model to use: "haiku" (fast) or "sonnet" (thorough). Default from config or "sonnet"
      --no-color           Disable colored output
  -o, --output string      Output format: table, json, jsonl, csv (default "table")
  -p, --profile string     Named config profile (default "default")
  -q, --quiet              Minimal output
      --timeout duration   HTTP request timeout (default 1m0s)
  -v, --verbose            Show debug information (HTTP requests/responses)
```

### SEE ALSO

* [dfir-cli ai](dfir-cli_ai.md)	 - AI-powered DFIR assistant

