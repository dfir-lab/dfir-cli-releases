# dfir-cli Manual Test Cases

**Document version:** 1.0
**Date:** 2026-03-27
**Tool under test:** dfir-cli (DFIR Lab CLI)
**API endpoint:** https://dfir-lab.ch/api/v1

## Test Environment Prerequisites

- A machine running macOS, Linux, or Windows
- A valid DFIR Lab API key (format: `sk-dfir-...`, 20-128 characters)
- A secondary/invalid API key for negative tests (e.g., `sk-dfir-invalidkey000000`)
- The `dfir-cli` binary built or installed and available in `$PATH`
- Internet connectivity to `https://dfir-lab.ch`
- A sample `.eml` phishing email file (referred to as `sample-phish.eml` below)
- `jq` installed for JSON output validation
- Shell: bash or zsh

## Exit Code Reference

| Code | Meaning                        |
|------|--------------------------------|
| 0    | Success / Clean                |
| 1    | Error                          |
| 2    | Malicious / High risk detected |
| 3    | Suspicious / Medium risk       |
| 4    | Insufficient credits           |

---

## UC1: Installation and First Run

### TC-UC1-001: Verify binary executes without errors

**Priority:** Critical
**Category:** Functional
**Estimated Time:** 2 minutes
**Preconditions:** Binary is installed and in `$PATH`

| Step | Action | Expected Result |
|------|--------|-----------------|
| 1 | Run `dfir-cli` with no arguments | Custom help/usage template is displayed. Output contains "DFIR Lab CLI" header, COMMANDS section (phishing, exposure, enrichment), ACCOUNT section (`credits`, `usage`), CONFIGURATION section (config), OTHER section (version, completion, update), GETTING STARTED examples, and LEARN MORE link. Exit code 0. |
| 2 | Run `dfir-cli --help` | Same help output as step 1. Exit code 0. |
| 3 | Run `dfir-cli -h` | Same help output as step 1. Exit code 0. |

**Pass criteria:** All three invocations produce identical help output and exit 0.

---

### TC-UC1-002: Verify --version flag

**Priority:** Critical
**Category:** Functional
**Estimated Time:** 2 minutes
**Preconditions:** Binary is installed

| Step | Action | Expected Result |
|------|--------|-----------------|
| 1 | Run `dfir-cli --version` | Prints version string (e.g., `dfir-cli version 0.x.x`). Exit code 0. |
| 2 | Run `dfir-cli version` | Prints build information including version, build date, commit hash, Go version, OS, and architecture. Exit code 0. |
| 3 | Run `dfir-cli ver` | Same output as step 2 (alias). Exit code 0. |
| 4 | Run `dfir-cli version --output json` | Prints JSON object with keys: `version`, `build_date`, `commit`, `go_version`, `os`, `arch`. Validate with `jq .`. Exit code 0. |

**Pass criteria:** Version info is displayed in both human and JSON format. The `ver` alias works.

---

### TC-UC1-003: Verify unknown command handling

**Priority:** Medium
**Category:** Negative
**Estimated Time:** 1 minute
**Preconditions:** Binary is installed

| Step | Action | Expected Result |
|------|--------|-----------------|
| 1 | Run `dfir-cli notacommand` | Error message displayed. Exit code 1. No stack trace or panic. |
| 2 | Run `dfir-cli enrichment notasub` | Error message about unknown subcommand. Exit code 1. |

**Pass criteria:** Graceful error messages, no panics, exit code 1.

---

### TC-UC1-004: Verify mutually exclusive flags --verbose and --quiet

**Priority:** High
**Category:** Negative
**Estimated Time:** 1 minute
**Preconditions:** Binary is installed

| Step | Action | Expected Result |
|------|--------|-----------------|
| 1 | Run `dfir-cli version -v -q` | Error: `--verbose and --quiet cannot be used together`. Exit code 1. |
| 2 | Run `dfir-cli version --verbose --quiet` | Same error as step 1. |

**Pass criteria:** Combining verbose and quiet is rejected with a clear message.

---

### TC-UC1-005: Verify background update check on every invocation

**Priority:** Low
**Category:** Functional
**Estimated Time:** 2 minutes
**Preconditions:** Binary is a release build (not `dev`)

| Step | Action | Expected Result |
|------|--------|-----------------|
| 1 | Run `dfir-cli version` and observe stderr | If a new version exists, an update notice is printed after the command output. If already latest, no notice is shown. The main command output is not delayed by the check. |

**Notes:** The update check is non-blocking. If the check has not completed by the time the command finishes, no notice is shown.

---

## UC2: Configuration and API Key Setup

### TC-UC2-001: Interactive config init (happy path)

**Priority:** Critical
**Category:** Functional
**Estimated Time:** 3 minutes
**Preconditions:** No existing config file at `~/.config/dfir-cli/config.yaml`. Delete it if present: `rm -f ~/.config/dfir-cli/config.yaml`

| Step | Action | Expected Result |
|------|--------|-----------------|
| 1 | Run `dfir-cli config init` | Wizard displays: "Welcome to dfir-cli configuration!", "Setting up profile: default", "Enter your API key: " prompt. |
| 2 | Type your valid API key (e.g., `sk-dfir-abcdef1234567890xxxx`) and press Enter | Key is read without echo (hidden input on TTY). Output: "Configuration saved successfully! (profile: default)" followed by "Next steps" guidance. Exit code 0. |
| 3 | Run `ls -la ~/.config/dfir-cli/config.yaml` | File exists with permissions `0600` (-rw-------). |
| 4 | Run `ls -ld ~/.config/dfir-cli/` | Directory exists with permissions `0700` (drwx------). |
| 5 | Run `cat ~/.config/dfir-cli/config.yaml` | YAML file contains `active_profile: default` and a `profiles.default` section with `api_key`, `api_url` (https://dfir-lab.ch/api/v1), `output_format` (table), `timeout` (1m0s), `concurrency` (5), `no_color` (false). |

**Pass criteria:** Config file created with correct permissions, correct defaults, and the API key stored.

---

### TC-UC2-002: Config init with piped input (non-TTY)

**Priority:** High
**Category:** Functional
**Estimated Time:** 2 minutes
**Preconditions:** No existing config, or use `--force`

| Step | Action | Expected Result |
|------|--------|-----------------|
| 1 | Run `echo "sk-dfir-abcdef1234567890xxxx" \| dfir-cli config init --force` | Key is read from stdin (not hidden since non-TTY). Config saved message displayed. Exit code 0. |
| 2 | Run `dfir-cli config get api-key` | Masked API key displayed (e.g., `sk-dfir-****...xxxx`). |

**Pass criteria:** Piped API key is accepted and stored correctly.

---

### TC-UC2-003: Config init rejects invalid API key formats

**Priority:** Critical
**Category:** Input Validation
**Estimated Time:** 3 minutes
**Preconditions:** None

| Step | Action | Expected Result |
|------|--------|-----------------|
| 1 | Run `echo "" \| dfir-cli config init --force` | Error: "API key cannot be empty". Exit code 1. |
| 2 | Run `echo "invalid-key-format" \| dfir-cli config init --force` | Error: "invalid API key: API key must start with \"sk-dfir-\"". Stderr includes link to https://dfir-lab.ch/settings/api-keys. Exit code 1. |
| 3 | Run `echo "sk-dfir-short" \| dfir-cli config init --force` | Error: "invalid API key: API key is too short (minimum 20 characters)". Exit code 1. |
| 4 | Run `echo "sk-dfir-$(python3 -c "print('a'*200)")" \| dfir-cli config init --force` | Error: "invalid API key: API key is too long (maximum 128 characters)". Exit code 1. |

**Pass criteria:** All invalid key formats are rejected with clear error messages.

---

### TC-UC2-004: Config init with existing config (overwrite prompt)

**Priority:** High
**Category:** Functional
**Estimated Time:** 2 minutes
**Preconditions:** A valid config file already exists

| Step | Action | Expected Result |
|------|--------|-----------------|
| 1 | Run `dfir-cli config init` | Prompt: "Configuration already exists for profile \"default\". Overwrite? [y/N]: " |
| 2 | Type `N` and press Enter | Output: "Aborted." Config file unchanged. Exit code 0. |
| 3 | Run `dfir-cli config init` again | Same prompt appears. |
| 4 | Type `y` and press Enter, then enter a valid API key | Config overwritten. "Configuration saved successfully!" displayed. |
| 5 | Run `dfir-cli config init --force` | No overwrite prompt. Goes directly to API key prompt. |

**Pass criteria:** Overwrite prompt respects user choice; `--force` bypasses it.

---

### TC-UC2-005: Config set and get for all valid keys

**Priority:** Critical
**Category:** Functional
**Estimated Time:** 5 minutes
**Preconditions:** Config initialized with `config init`

| Step | Action | Expected Result |
|------|--------|-----------------|
| 1 | Run `dfir-cli config set api-key sk-dfir-newkey1234567890ab` | Output: `Set "api-key" to "sk-dfir-****...90ab" (profile: default)`. API key is masked in output. |
| 2 | Run `dfir-cli config get api-key` | Returns masked key: `sk-dfir-****...90ab` |
| 3 | Run `dfir-cli config get api-key --unmask` | Returns full key: `sk-dfir-newkey1234567890ab` |
| 4 | Run `dfir-cli config set api-url https://staging.dfir-lab.ch/api/v1` | Output: `Set "api-url" to "https://staging.dfir-lab.ch/api/v1" (profile: default)` |
| 5 | Run `dfir-cli config get api-url` | Returns `https://staging.dfir-lab.ch/api/v1` |
| 6 | Run `dfir-cli config set output-format json` | Output confirms change. |
| 7 | Run `dfir-cli config get output-format` | Returns `json` |
| 8 | Run `dfir-cli config set timeout 2m30s` | Output confirms change. |
| 9 | Run `dfir-cli config get timeout` | Returns `2m30s` |
| 10 | Run `dfir-cli config set concurrency 10` | Output confirms change. |
| 11 | Run `dfir-cli config get concurrency` | Returns `10` |
| 12 | Run `dfir-cli config set no-color true` | Output confirms change. |
| 13 | Run `dfir-cli config get no-color` | Returns `true` |

**Pass criteria:** All config keys can be set and retrieved. API key is masked by default.

---

### TC-UC2-006: Config set validation for invalid values

**Priority:** High
**Category:** Input Validation
**Estimated Time:** 3 minutes
**Preconditions:** Config initialized

| Step | Action | Expected Result |
|------|--------|-----------------|
| 1 | Run `dfir-cli config set api-key bad-prefix` | Error: "invalid API key: API key must start with \"sk-dfir-\"". Exit code 1. |
| 2 | Run `dfir-cli config set api-url ""` | Error: "api-url cannot be empty". Exit code 1. |
| 3 | Run `dfir-cli config set output-format xml` | Error: "invalid output format \"xml\". Valid formats: table, json, jsonl, csv". Exit code 1. |
| 4 | Run `dfir-cli config set timeout notaduration` | Error containing "invalid timeout value". Exit code 1. |
| 5 | Run `dfir-cli config set timeout -5s` | Error: "timeout must be a positive duration". Exit code 1. |
| 6 | Run `dfir-cli config set concurrency 0` | Error: "invalid concurrency value \"0\": must be a positive integer between 1 and 100". Exit code 1. |
| 7 | Run `dfir-cli config set concurrency 101` | Same style error about range 1-100. Exit code 1. |
| 8 | Run `dfir-cli config set concurrency abc` | Error about invalid concurrency value. Exit code 1. |
| 9 | Run `dfir-cli config set no-color maybe` | Error: "invalid no-color value \"maybe\": must be true or false". Exit code 1. |
| 10 | Run `dfir-cli config set not-a-key value` | Error: "unknown config key \"not-a-key\". Valid keys: api-key, api-url, output-format, timeout, concurrency, no-color". Exit code 1. |

**Pass criteria:** All invalid values are rejected with descriptive error messages.

---

### TC-UC2-007: Config list displays all settings

**Priority:** High
**Category:** Functional
**Estimated Time:** 2 minutes
**Preconditions:** Config initialized with known values

| Step | Action | Expected Result |
|------|--------|-----------------|
| 1 | Run `dfir-cli config list` | Displays: Profile name, api-key (masked), api-url, output-format, timeout, concurrency, no-color. All values match what was set. |
| 2 | Run `dfir-cli config list --unmask` | Same output but api-key is shown in full. |

**Pass criteria:** All config values listed correctly. Masking toggled by `--unmask`.

---

### TC-UC2-008: Config get before init

**Priority:** Medium
**Category:** Edge Case
**Estimated Time:** 1 minute
**Preconditions:** No config file exists. `rm -f ~/.config/dfir-cli/config.yaml`

| Step | Action | Expected Result |
|------|--------|-----------------|
| 1 | Run `dfir-cli config get api-key` | Error: "config file not found. Run: dfir-cli config init". Exit code 1. |
| 2 | Run `dfir-cli config list` | Same error as step 1. |

**Pass criteria:** Clear guidance to run `config init` when no config exists.

---

### TC-UC2-009: Multiple profiles

**Priority:** High
**Category:** Functional
**Estimated Time:** 4 minutes
**Preconditions:** Default profile already initialized

| Step | Action | Expected Result |
|------|--------|-----------------|
| 1 | Run `echo "sk-dfir-staging1234567890xx" \| dfir-cli config init --profile staging --force` | Config saved for profile "staging". |
| 2 | Run `dfir-cli config list --profile staging` | Shows staging profile config with the staging API key. |
| 3 | Run `dfir-cli config list` | Shows default profile config (different key from staging). |
| 4 | Run `dfir-cli config set api-url https://staging.example.com --profile staging` | Set confirmed for staging profile. |
| 5 | Run `dfir-cli config get api-url --profile staging` | Returns `https://staging.example.com` |
| 6 | Run `dfir-cli config get api-url` | Returns the default profile URL (unchanged). |

**Pass criteria:** Profiles are isolated; changes to one do not affect the other.

---

### TC-UC2-010: API key precedence (flag > env > config)

**Priority:** Critical
**Category:** Functional
**Estimated Time:** 3 minutes
**Preconditions:** Config initialized with a valid API key. Requires `--verbose` to observe which key is used.

| Step | Action | Expected Result |
|------|--------|-----------------|
| 1 | Run `dfir-cli enrichment lookup --ip 8.8.8.8 -v` 2>&1 and inspect stderr | Verbose line shows `(auth: sk-dfir-***...XXXX)` where XXXX matches the last 4 chars of the config file key. |
| 2 | Run `DFIR_LAB_API_KEY=sk-dfir-envkey12345678901234 dfir-cli enrichment lookup --ip 8.8.8.8 -v` 2>&1 and inspect stderr | Verbose auth shows last 4 chars matching the env var key (`1234`). |
| 3 | Run `DFIR_LAB_API_KEY=sk-dfir-envkey12345678901234 dfir-cli enrichment lookup --ip 8.8.8.8 --api-key sk-dfir-flagkey1234567890abcd -v` 2>&1 and inspect stderr | Verbose auth shows last 4 chars matching the flag key (`abcd`). Also, stderr warning: "Warning: passing --api-key on the command line exposes it in process listings and shell history." |

**Pass criteria:** Flag overrides env, env overrides config file. Warning shown for `--api-key` flag.

---

### TC-UC2-011: Environment variable for config directory

**Priority:** Medium
**Category:** Functional
**Estimated Time:** 2 minutes
**Preconditions:** None

| Step | Action | Expected Result |
|------|--------|-----------------|
| 1 | Run `DFIR_LAB_CONFIG_DIR=/tmp/dfir-test-config echo "sk-dfir-testkey1234567890xx" \| dfir-cli config init --force` | Config saved. |
| 2 | Run `ls -la /tmp/dfir-test-config/config.yaml` | File exists with 0600 permissions. |
| 3 | Run `DFIR_LAB_CONFIG_DIR=/tmp/dfir-test-config dfir-cli config list` | Shows config from the custom directory. |
| 4 | Clean up: `rm -rf /tmp/dfir-test-config` | Directory removed. |

**Pass criteria:** `DFIR_LAB_CONFIG_DIR` overrides the default config location.

---

## UC3: IOC Enrichment Workflow

### TC-UC3-001: Single IP address lookup (table output)

**Priority:** Critical
**Category:** Functional
**Estimated Time:** 3 minutes
**Preconditions:** Valid API key configured. Sufficient credits.

| Step | Action | Expected Result |
|------|--------|-----------------|
| 1 | Run `dfir-cli enrichment lookup --ip 8.8.8.8` | Spinner displayed while processing ("Enriching indicators..."). Table output with header "IOC Enrichment: 8.8.8.8 (IP)". Shows Verdict (CLEAN/SUSPICIOUS/MALICIOUS), Score bar, Consensus (X/Y providers flagged). Provider table with columns: Provider, Verdict, Score, Details. Credits footer at bottom. Exit code 0 (clean), 2 (malicious), or 3 (suspicious) matching the verdict. |
| 2 | Verify exit code: `echo $?` | Matches the verdict: 0 for clean, 2 for malicious, 3 for suspicious. |

**Pass criteria:** Table renders correctly, exit code matches verdict.

---

### TC-UC3-002: Domain lookup

**Priority:** Critical
**Category:** Functional
**Estimated Time:** 2 minutes
**Preconditions:** Valid API key configured

| Step | Action | Expected Result |
|------|--------|-----------------|
| 1 | Run `dfir-cli enrichment lookup --domain google.com` | Table output with header containing "google.com (DOMAIN)". Results displayed. |
| 2 | Run `dfir-cli enrichment lookup --domain evil-known-domain.example` | Results include provider verdicts. Exit code matches highest severity. |

**Pass criteria:** Domain lookups return results with correct type label.

---

### TC-UC3-003: Hash lookup

**Priority:** High
**Category:** Functional
**Estimated Time:** 2 minutes
**Preconditions:** Valid API key configured

| Step | Action | Expected Result |
|------|--------|-----------------|
| 1 | Run `dfir-cli enrichment lookup --hash 44d88612fea8a8f36de82e1278abb02f` (EICAR MD5) | Table output with "(HASH)" type label. Likely malicious verdict. Exit code 2. |
| 2 | Run `dfir-cli enrichment lookup --hash 275a021bbfb6489e54d471899f7db9d1663fc695ec2fe2a2c4538aabf651fd0f` (EICAR SHA-256) | Similar result for SHA-256 hash. |

**Pass criteria:** Both MD5 and SHA-256 hashes resolve correctly.

---

### TC-UC3-004: Email lookup

**Priority:** High
**Category:** Functional
**Estimated Time:** 2 minutes
**Preconditions:** Valid API key configured

| Step | Action | Expected Result |
|------|--------|-----------------|
| 1 | Run `dfir-cli enrichment lookup --email test@example.com` | Table output with "(EMAIL)" type label. Results from providers displayed. |

**Pass criteria:** Email IOC type handled and labeled correctly.

---

### TC-UC3-005: URL lookup

**Priority:** High
**Category:** Functional
**Estimated Time:** 2 minutes
**Preconditions:** Valid API key configured

| Step | Action | Expected Result |
|------|--------|-----------------|
| 1 | Run `dfir-cli enrichment lookup --url "https://example.com/phishing"` | Table output with "(URL)" type label. Results displayed. |

**Pass criteria:** URL IOC type handled correctly. Quotes preserve the full URL.

---

### TC-UC3-006: Generic --ioc flag with auto-detection

**Priority:** High
**Category:** Functional
**Estimated Time:** 3 minutes
**Preconditions:** Valid API key configured

| Step | Action | Expected Result |
|------|--------|-----------------|
| 1 | Run `dfir-cli enrichment lookup --ioc 1.2.3.4` | Auto-detected as IP. Header shows "(IP)". |
| 2 | Run `dfir-cli enrichment lookup --ioc evil.com` | Auto-detected as domain. Header shows "(DOMAIN)". |
| 3 | Run `dfir-cli enrichment lookup --ioc user@evil.com` | Auto-detected as email (contains @). Header shows "(EMAIL)". |
| 4 | Run `dfir-cli enrichment lookup --ioc 44d88612fea8a8f36de82e1278abb02f` | Auto-detected as hash (32 hex chars = MD5). Header shows "(HASH)". |
| 5 | Run `dfir-cli enrichment lookup --ioc https://evil.com/path` | Auto-detected as URL (starts with https://). Header shows "(URL)". |
| 6 | Run `dfir-cli enrichment lookup --ioc 2001:db8::1` | Auto-detected as IP (IPv6). Header shows "(IP)". |

**Pass criteria:** All IOC types correctly auto-detected.

---

### TC-UC3-007: Override auto-detection with --type

**Priority:** Medium
**Category:** Functional
**Estimated Time:** 2 minutes
**Preconditions:** Valid API key configured

| Step | Action | Expected Result |
|------|--------|-----------------|
| 1 | Run `dfir-cli enrichment lookup --ioc something.suspicious --type domain` | Forced to domain type regardless of auto-detection. |
| 2 | Run `dfir-cli enrichment lookup --ioc something --type invalid` | Error: "invalid IOC type \"invalid\". Valid types: ip, domain, url, hash, email". Exit code 1. |

**Pass criteria:** `--type` overrides auto-detection; invalid types rejected.

---

### TC-UC3-008: Batch file lookup

**Priority:** Critical
**Category:** Functional
**Estimated Time:** 4 minutes
**Preconditions:** Valid API key configured. Create test file.

**Test data file (`/tmp/test-iocs.txt`):**
```
# Test IOC batch file
1.2.3.4
evil.com
44d88612fea8a8f36de82e1278abb02f

# Empty line above should be skipped
google.com
```

| Step | Action | Expected Result |
|------|--------|-----------------|
| 1 | Create the file above at `/tmp/test-iocs.txt` | File created. |
| 2 | Run `dfir-cli enrichment lookup --batch /tmp/test-iocs.txt` | Multiple results displayed, one per IOC. Comment lines and empty lines are skipped. 4 IOCs processed (1.2.3.4, evil.com, hash, google.com). Types auto-detected. Credits footer shown. |
| 3 | Verify exit code | Highest severity verdict across all results determines exit code. |

**Pass criteria:** Batch file parsed correctly. Comments/blanks skipped. All IOCs enriched. Chunked into batches of 10 if more than 10 indicators.

---

### TC-UC3-009: Batch from stdin with --batch -

**Priority:** High
**Category:** Functional
**Estimated Time:** 2 minutes
**Preconditions:** Valid API key configured

| Step | Action | Expected Result |
|------|--------|-----------------|
| 1 | Run `echo -e "1.2.3.4\n8.8.8.8" \| dfir-cli enrichment lookup --batch - --type ip` | Both IPs enriched and results displayed. |
| 2 | Run `echo "1.2.3.4" \| dfir-cli enrichment lookup --batch -` (no --type) | Error: "--type is required when reading from stdin". Exit code 1. |

**Pass criteria:** Stdin batch requires explicit `--type`.

---

### TC-UC3-010: Stdin pipe without --batch flag

**Priority:** High
**Category:** Functional
**Estimated Time:** 2 minutes
**Preconditions:** Valid API key configured

| Step | Action | Expected Result |
|------|--------|-----------------|
| 1 | Run `echo "1.2.3.4" \| dfir-cli enrichment lookup --type ip` | IP enriched from stdin. Results displayed. |
| 2 | Run `echo "1.2.3.4" \| dfir-cli enrichment lookup` (no --type) | Error: "--type is required when reading from stdin". Exit code 1. |

**Pass criteria:** Stdin requires `--type` for plain pipe (not just --batch).

---

### TC-UC3-011: JSON output format

**Priority:** Critical
**Category:** Functional
**Estimated Time:** 2 minutes
**Preconditions:** Valid API key configured

| Step | Action | Expected Result |
|------|--------|-----------------|
| 1 | Run `dfir-cli enrichment lookup --ip 8.8.8.8 --output json` | Valid JSON output with `results` array and `meta` object. Validate: `dfir-cli enrichment lookup --ip 8.8.8.8 -o json \| jq .` parses cleanly. |
| 2 | Inspect JSON structure | Each result has: `indicator` (type, value), `verdict`, `score`, `providers` map. Meta has: `request_id`, `credits_used`, `credits_remaining`. |

**Pass criteria:** Output is valid, parseable JSON with correct structure.

---

### TC-UC3-012: JSONL output format

**Priority:** High
**Category:** Functional
**Estimated Time:** 2 minutes
**Preconditions:** Valid API key configured

| Step | Action | Expected Result |
|------|--------|-----------------|
| 1 | Run `dfir-cli enrichment lookup --ip 8.8.8.8 --output jsonl` | One JSON object per line. Each line parses independently with `jq`. |
| 2 | Run batch with JSONL: `dfir-cli enrichment lookup --batch /tmp/test-iocs.txt -o jsonl \| wc -l` | Line count equals number of IOCs in the file (4). |

**Pass criteria:** Each result on its own line; parseable independently.

---

### TC-UC3-013: Quiet output mode

**Priority:** High
**Category:** Functional
**Estimated Time:** 2 minutes
**Preconditions:** Valid API key configured

| Step | Action | Expected Result |
|------|--------|-----------------|
| 1 | Run `dfir-cli enrichment lookup --ip 8.8.8.8 -q` | Output is a single line: `VERDICT SCORE VALUE` (e.g., `CLEAN 0 8.8.8.8`). No table, no headers, no credits footer. |
| 2 | Run batch quiet: `dfir-cli enrichment lookup --batch /tmp/test-iocs.txt -q` | One line per IOC in format `VERDICT SCORE VALUE`. |

**Pass criteria:** Minimal output suitable for scripting.

---

### TC-UC3-014: Provider and score filtering

**Priority:** Medium
**Category:** Functional
**Estimated Time:** 3 minutes
**Preconditions:** Valid API key configured

| Step | Action | Expected Result |
|------|--------|-----------------|
| 1 | Run `dfir-cli enrichment lookup --ip 8.8.8.8 -o json \| jq '.results[0].providers \| keys'` | Note the full provider list. |
| 2 | Run `dfir-cli enrichment lookup --ip 8.8.8.8 --providers "ProviderA,ProviderB" -o json \| jq '.results[0].providers \| keys'` | Only the specified providers appear in results (case-insensitive match). |
| 3 | Run `dfir-cli enrichment lookup --ip 8.8.8.8 --min-score 50 -o json` | Only providers with score >= 50 included. |

**Pass criteria:** Filters narrow down provider results correctly.

---

### TC-UC3-015: Enrichment lookup with no input shows help

**Priority:** Medium
**Category:** Edge Case
**Estimated Time:** 1 minute
**Preconditions:** Valid API key configured. Running in a TTY (not piped).

| Step | Action | Expected Result |
|------|--------|-----------------|
| 1 | Run `dfir-cli enrichment lookup` (no flags, no stdin) | Help text for the lookup subcommand is displayed. Exit code 0. |

**Pass criteria:** No error; help displayed as fallback.

---

### TC-UC3-016: Non-TTY auto-detection switches to JSON

**Priority:** High
**Category:** Functional
**Estimated Time:** 2 minutes
**Preconditions:** Valid API key configured

| Step | Action | Expected Result |
|------|--------|-----------------|
| 1 | Run `dfir-cli enrichment lookup --ip 8.8.8.8 \| cat` | Output is JSON (not table), because stdout is not a TTY. Colors disabled. Validate with `dfir-cli enrichment lookup --ip 8.8.8.8 \| jq .`. |

**Pass criteria:** When piped, output auto-switches to JSON and disables color.

---

### TC-UC3-017: Exit code 2 for malicious IOC

**Priority:** Critical
**Category:** Functional
**Estimated Time:** 2 minutes
**Preconditions:** Valid API key configured. Need a known-malicious indicator.

| Step | Action | Expected Result |
|------|--------|-----------------|
| 1 | Run `dfir-cli enrichment lookup --hash 44d88612fea8a8f36de82e1278abb02f; echo "Exit: $?"` | If verdict is "malicious", exit code is 2. |

**Notes:** The actual verdict depends on the API response. Use a known-malicious EICAR hash. If the API does not flag it, this test should be marked as dependent on test data.

**Pass criteria:** Exit code 2 when any result has malicious verdict.

---

### TC-UC3-018: CSV output format

**Priority:** Medium
**Category:** Functional
**Estimated Time:** 2 minutes
**Preconditions:** Valid API key configured

| Step | Action | Expected Result |
|------|--------|-----------------|
| 1 | Run `dfir-cli enrichment lookup --ip 8.8.8.8 --output csv` | Output is in CSV format (if supported by the enrichment renderer). If CSV is not supported for enrichment, an appropriate error message is shown. |

**Notes:** The output flag accepts `csv` globally. Verify behavior for enrichment specifically.

---

## UC4: Phishing Email Analysis

### TC-UC4-001: Analyze .eml file (standard analysis)

**Priority:** Critical
**Category:** Functional
**Estimated Time:** 3 minutes
**Preconditions:** Valid API key configured. `sample-phish.eml` file available.

| Step | Action | Expected Result |
|------|--------|-----------------|
| 1 | Run `dfir-cli phishing analyze --file sample-phish.eml` | Spinner: "Analyzing email for phishing indicators..." Table output: "Phishing Analysis" header, Verdict (CLEAN/SUSPICIOUS/MALICIOUS/HIGHLY_MALICIOUS) with color, Score bar, Summary text, Authentication Results (SPF, DKIM, DMARC, ARC), Key Findings list, Suspicious Indicators table (Category, Description, Severity), Extracted IOCs table (Type, Value, Verdict), Recommended Actions numbered list, Credits footer. |
| 2 | Verify exit code: `echo $?` | 0 for safe/clean, 2 for malicious/highly_malicious, 3 for suspicious. |

**Pass criteria:** Full analysis table rendered with all sections. Exit code matches verdict.

---

### TC-UC4-002: Analyze with AI-enhanced mode

**Priority:** Critical
**Category:** Functional
**Estimated Time:** 3 minutes
**Preconditions:** Valid API key configured. Sufficient credits (AI costs 10 credits). `sample-phish.eml` available.

| Step | Action | Expected Result |
|------|--------|-----------------|
| 1 | Run `dfir-cli phishing analyze --file sample-phish.eml --ai` | Spinner: "Running AI-enhanced phishing analysis..." Table includes standard analysis PLUS an "AI Assessment" section with: Risk Level (with confidence percentage), Summary, Model name, AI Key Findings list, AI Recommended Actions numbered list. |
| 2 | Check credits: `dfir-cli credits` | Last used shows 10 credits (AI analysis cost). |

**Pass criteria:** AI section displayed. 10 credits consumed.

---

### TC-UC4-003: Analyze raw email content via --raw

**Priority:** High
**Category:** Functional
**Estimated Time:** 2 minutes
**Preconditions:** Valid API key configured

| Step | Action | Expected Result |
|------|--------|-----------------|
| 1 | Run `dfir-cli phishing analyze --raw "From: attacker@evil.com\nTo: victim@company.com\nSubject: Urgent\n\nClick here: http://evil.com"` | Analysis results displayed. Input type detected as "raw". |

**Pass criteria:** Raw string content accepted and analyzed.

---

### TC-UC4-004: Analyze via stdin pipe

**Priority:** High
**Category:** Functional
**Estimated Time:** 2 minutes
**Preconditions:** Valid API key configured. `sample-phish.eml` available.

| Step | Action | Expected Result |
|------|--------|-----------------|
| 1 | Run `cat sample-phish.eml \| dfir-cli phishing analyze` | Email content read from stdin. Analysis results displayed. Input type: "raw". |

**Pass criteria:** Stdin pipe accepted for phishing analysis.

---

### TC-UC4-005: JSON output for phishing analysis

**Priority:** High
**Category:** Functional
**Estimated Time:** 2 minutes
**Preconditions:** Valid API key configured

| Step | Action | Expected Result |
|------|--------|-----------------|
| 1 | Run `dfir-cli phishing analyze --file sample-phish.eml -o json \| jq .` | Valid JSON with verdict, score, authentication results, key findings, suspicious indicators, extracted IOCs, recommended actions. |

**Pass criteria:** Parseable JSON output with complete data structure.

---

### TC-UC4-006: JSONL output for phishing analysis

**Priority:** Medium
**Category:** Functional
**Estimated Time:** 1 minute
**Preconditions:** Valid API key configured

| Step | Action | Expected Result |
|------|--------|-----------------|
| 1 | Run `dfir-cli phishing analyze --file sample-phish.eml -o jsonl` | Single JSON line (one analysis = one line). Parseable with `jq`. |

**Pass criteria:** Valid JSONL output.

---

### TC-UC4-007: CSV output unsupported for phishing

**Priority:** Medium
**Category:** Negative
**Estimated Time:** 1 minute
**Preconditions:** Valid API key configured

| Step | Action | Expected Result |
|------|--------|-----------------|
| 1 | Run `dfir-cli phishing analyze --file sample-phish.eml -o csv` | Error: "CSV output is not supported for phishing analysis". Exit code 1. |

**Pass criteria:** Clear error message for unsupported format.

---

### TC-UC4-008: Quiet mode for phishing

**Priority:** Medium
**Category:** Functional
**Estimated Time:** 1 minute
**Preconditions:** Valid API key configured

| Step | Action | Expected Result |
|------|--------|-----------------|
| 1 | Run `dfir-cli phishing analyze --file sample-phish.eml -q` | Output: single line `VERDICT SCORE` (e.g., `SUSPICIOUS 65`). No table or details. |

**Pass criteria:** Minimal verdict-only output.

---

### TC-UC4-009: File not found error

**Priority:** High
**Category:** Negative
**Estimated Time:** 1 minute
**Preconditions:** None

| Step | Action | Expected Result |
|------|--------|-----------------|
| 1 | Run `dfir-cli phishing analyze --file /nonexistent/email.eml` | Error: "file not found: /nonexistent/email.eml". Exit code 1. |

**Pass criteria:** Clear file-not-found error.

---

### TC-UC4-010: File too large (over 5MB)

**Priority:** Medium
**Category:** Edge Case
**Estimated Time:** 2 minutes
**Preconditions:** None

| Step | Action | Expected Result |
|------|--------|-----------------|
| 1 | Create a 6MB file: `dd if=/dev/zero of=/tmp/large.eml bs=1M count=6` | File created. |
| 2 | Run `dfir-cli phishing analyze --file /tmp/large.eml` | Error: "file too large (max 5MB)". Exit code 1. |
| 3 | Clean up: `rm /tmp/large.eml` | File removed. |

**Pass criteria:** Files exceeding 5MB rejected.

---

### TC-UC4-011: No input provided error

**Priority:** High
**Category:** Negative
**Estimated Time:** 1 minute
**Preconditions:** Running in a TTY (not piped)

| Step | Action | Expected Result |
|------|--------|-----------------|
| 1 | Run `dfir-cli phishing analyze` (no flags, no stdin) | Error: "no input provided." followed by usage examples showing --file, --raw, and cat pipe methods. Exit code 1. |

**Pass criteria:** Clear error with usage guidance.

---

### TC-UC4-012: Legacy --url flag is hidden and rejected

**Priority:** Medium
**Category:** Negative
**Estimated Time:** 1 minute
**Preconditions:** None

| Step | Action | Expected Result |
|------|--------|-----------------|
| 1 | Run `dfir-cli phishing analyze --help` | The `--url` flag is not shown in help output. |
| 2 | Run `dfir-cli phishing analyze --url https://phishing.example.com` | CLI prints a deprecation warning for `--url` and then returns: "--url is not supported for phishing analysis." with tip to use `dfir-cli enrichment lookup --url`. Exit code 1. |

**Pass criteria:** The public help surface does not advertise URL analysis for this command, while legacy `--url` usage still fails with a clear migration path.

---

### TC-UC4-013: Auto-detection of .eml file extension

**Priority:** Medium
**Category:** Functional
**Estimated Time:** 2 minutes
**Preconditions:** Valid API key configured

| Step | Action | Expected Result |
|------|--------|-----------------|
| 1 | Run `dfir-cli phishing analyze --file sample-phish.eml` (with .eml extension) | Input type auto-detected as "eml". |
| 2 | Copy the file: `cp sample-phish.eml /tmp/email.txt` | File copied. |
| 3 | Run `dfir-cli phishing analyze --file /tmp/email.txt` | Input type auto-detected as "raw" (non-.eml extension). |
| 4 | Run `dfir-cli phishing analyze --file /tmp/email.txt --type eml` | Input type overridden to "eml" via --type flag. |

**Pass criteria:** Extension-based auto-detection works; --type overrides it.

---

## UC5: Exposure Scanning

### TC-UC5-001: Single domain scan (table output)

**Priority:** Critical
**Category:** Functional
**Estimated Time:** 4 minutes (scans can take up to 3 minutes)
**Preconditions:** Valid API key configured. Sufficient credits.

| Step | Action | Expected Result |
|------|--------|-----------------|
| 1 | Run `dfir-cli exposure scan --domain example.com` | Spinner: "Scanning example.com..." Table output: "Exposure Scan: example.com" header, Risk Level (with color badge), Risk Score (bar), Status, Cached (Yes/No), Providers list, SSL Grade (if available, with A/A+ green, B yellow, others red), Duration. Credits footer. Hint: "For full details, re-run with --output json". |
| 2 | Verify exit code | 0 for low/none risk, 2 for critical/high risk, 3 for medium risk. |

**Pass criteria:** Scan completes, table rendered with all fields, exit code matches risk level.

---

### TC-UC5-002: Exposure scan with JSON output

**Priority:** High
**Category:** Functional
**Estimated Time:** 4 minutes
**Preconditions:** Valid API key configured

| Step | Action | Expected Result |
|------|--------|-----------------|
| 1 | Run `dfir-cli exposure scan --domain example.com -o json \| jq .` | Valid JSON with `data` (target, risk_level, risk_score, status, cached, providers, results, stats) and `meta` (request_id, credits_used, credits_remaining). |

**Pass criteria:** Full scan details available in JSON output.

---

### TC-UC5-003: Batch domain scanning

**Priority:** High
**Category:** Functional
**Estimated Time:** 8 minutes
**Preconditions:** Valid API key configured

**Test data file (`/tmp/test-domains.txt`):**
```
# Domains to scan
example.com
google.com
```

| Step | Action | Expected Result |
|------|--------|-----------------|
| 1 | Create `/tmp/test-domains.txt` with the content above | File created. |
| 2 | Run `dfir-cli exposure scan --batch /tmp/test-domains.txt` | Progress messages on stderr: `[1/2] Scanning example.com...`, `[2/2] Scanning google.com...`. Results for each domain displayed, separated by blank lines. |
| 3 | Verify exit code | Highest severity across all scans. |

**Pass criteria:** All domains scanned sequentially. Progress shown. Results separated.

---

### TC-UC5-004: Exposure scan via stdin

**Priority:** High
**Category:** Functional
**Estimated Time:** 4 minutes
**Preconditions:** Valid API key configured

| Step | Action | Expected Result |
|------|--------|-----------------|
| 1 | Run `echo "example.com" \| dfir-cli exposure scan` | Domain read from stdin. Scan results displayed. |

**Pass criteria:** Single domain from stdin accepted.

---

### TC-UC5-005: Exposure scan quiet mode

**Priority:** Medium
**Category:** Functional
**Estimated Time:** 4 minutes
**Preconditions:** Valid API key configured

| Step | Action | Expected Result |
|------|--------|-----------------|
| 1 | Run `dfir-cli exposure scan --domain example.com -q` | Single line output: `LEVEL SCORE TARGET` (e.g., `LOW 15 example.com`). No table, no spinner, no credits footer. |

**Pass criteria:** Minimal output for scripting.

---

### TC-UC5-006: Exposure scan with JSONL output

**Priority:** Medium
**Category:** Functional
**Estimated Time:** 4 minutes
**Preconditions:** Valid API key configured

| Step | Action | Expected Result |
|------|--------|-----------------|
| 1 | Run `dfir-cli exposure scan --domain example.com -o jsonl` | Single JSON line with data and meta. Parseable with `jq`. |

**Pass criteria:** Valid JSONL output.

---

### TC-UC5-007: No target specified error

**Priority:** High
**Category:** Negative
**Estimated Time:** 1 minute
**Preconditions:** Running in TTY

| Step | Action | Expected Result |
|------|--------|-----------------|
| 1 | Run `dfir-cli exposure scan` (no flags, no stdin) | Error: "no target specified. Use --domain, --batch, or pipe via stdin". Exit code 1. |

**Pass criteria:** Clear error with input method guidance.

---

### TC-UC5-008: Batch file with empty content

**Priority:** Medium
**Category:** Edge Case
**Estimated Time:** 1 minute
**Preconditions:** None

| Step | Action | Expected Result |
|------|--------|-----------------|
| 1 | Create an empty file: `touch /tmp/empty-domains.txt` | File created. |
| 2 | Run `dfir-cli exposure scan --batch /tmp/empty-domains.txt` | Error: "batch input contained no targets". Exit code 1. |
| 3 | Create file with only comments: `echo "# just a comment" > /tmp/comment-only.txt` | File created. |
| 4 | Run `dfir-cli exposure scan --batch /tmp/comment-only.txt` | Same error: "batch input contained no targets". Exit code 1. |

**Pass criteria:** Empty/comment-only batch files rejected with clear message.

---

### TC-UC5-009: Batch file not found

**Priority:** Medium
**Category:** Negative
**Estimated Time:** 1 minute
**Preconditions:** None

| Step | Action | Expected Result |
|------|--------|-----------------|
| 1 | Run `dfir-cli exposure scan --batch /tmp/nonexistent-file.txt` | Error: "open batch file: ..." (file not found). Exit code 1. |

**Pass criteria:** File not found error propagated clearly.

---

### TC-UC5-010: Timeout behavior for long scans

**Priority:** High
**Category:** Performance
**Estimated Time:** 2 minutes
**Preconditions:** Valid API key configured

| Step | Action | Expected Result |
|------|--------|-----------------|
| 1 | Run `dfir-cli exposure scan --domain example.com --timeout 1s` | The global timeout is 1s, but exposure scans enforce a minimum of 3 minutes. Verify: the scan does NOT fail after 1 second. The 3-minute minimum override applies. |
| 2 | Run `dfir-cli exposure scan --domain example.com --timeout 5m` | Custom timeout of 5 minutes used (exceeds the 3-minute minimum). |

**Notes:** The code enforces `timeout = max(user_timeout, 3_minutes)`. This means short timeouts are silently upgraded.

**Pass criteria:** Minimum 3-minute timeout enforced for exposure scans.

---

### TC-UC5-011: Target type hint

**Priority:** Low
**Category:** Functional
**Estimated Time:** 4 minutes
**Preconditions:** Valid API key configured

| Step | Action | Expected Result |
|------|--------|-----------------|
| 1 | Run `dfir-cli exposure scan --domain example.com --target-type domain` | Explicit domain type. Results displayed normally. |
| 2 | Run `dfir-cli exposure scan --domain example.com --target-type auto` | Default auto-detection. Same behavior as without the flag. |

**Pass criteria:** `--target-type` accepted and passed to the API.

---

## UC6: Error Handling and Edge Cases

### TC-UC6-001: Invalid API key (401 Unauthorized)

**Priority:** Critical
**Category:** Security / Error Handling
**Estimated Time:** 2 minutes
**Preconditions:** Set an invalid API key

| Step | Action | Expected Result |
|------|--------|-----------------|
| 1 | Run `dfir-cli enrichment lookup --ip 8.8.8.8 --api-key sk-dfir-invalidkey00000000` | Error: "authentication failed: invalid API key. Run: dfir-cli config init" (or similar message from API). Stderr also shows the --api-key warning. Exit code 1. |
| 2 | Run `DFIR_LAB_API_KEY=sk-dfir-invalidkey00000000 dfir-cli enrichment lookup --ip 8.8.8.8` | Same authentication error. No --api-key warning (env var is safe). Exit code 1. |

**Pass criteria:** 401 errors produce a clear "authentication failed" message with guidance. No key exposure in error output.

---

### TC-UC6-002: Insufficient credits (402 Payment Required)

**Priority:** Critical
**Category:** Error Handling
**Estimated Time:** 2 minutes
**Preconditions:** An API key with zero or very few credits remaining

| Step | Action | Expected Result |
|------|--------|-----------------|
| 1 | Run `dfir-cli enrichment lookup --ip 8.8.8.8` (with depleted credits) | Error on stderr: "Error: insufficient credits: not enough credits to complete this request" with link to billing page and suggestion to run `dfir-cli credits`. Exit code 4. |
| 2 | Run `dfir-cli phishing analyze --file sample-phish.eml` (with depleted credits) | Error: "Error: insufficient credits to perform this analysis." with guidance: "Check your balance: dfir-cli credits". Exit code 4. |
| 3 | Run `dfir-cli exposure scan --domain example.com` (with depleted credits) | Insufficient credits error. Exit code 4. |

**Pass criteria:** Exit code 4 for all insufficient credit scenarios. Clear message with billing link.

---

### TC-UC6-003: Rate limiting (429 Too Many Requests)

**Priority:** High
**Category:** Error Handling
**Estimated Time:** 5 minutes
**Preconditions:** Valid API key. Ability to trigger rate limits (rapid requests).

| Step | Action | Expected Result |
|------|--------|-----------------|
| 1 | Run many rapid requests in a loop: `for i in $(seq 1 50); do dfir-cli enrichment lookup --ip 1.2.3.$i -q & done; wait` | Some requests may receive 429 responses. The client automatically retries with exponential backoff (up to 3 retries). |
| 2 | Run with verbose: `dfir-cli enrichment lookup --ip 1.2.3.4 -v` during rate limiting | Verbose stderr shows: `[verbose] 429 rate limited, retrying in Xs (attempt N/3)`. |

**Notes:** This test depends on the API's actual rate limit thresholds. May not be easily reproducible.

**Pass criteria:** Automatic retry on 429. Retry-After header respected. After max retries, error message shown.

---

### TC-UC6-004: Server error (5xx) with retries

**Priority:** High
**Category:** Error Handling
**Estimated Time:** 3 minutes
**Preconditions:** Valid API key. Use a mock/staging server or --api-url pointing to one that returns 500.

| Step | Action | Expected Result |
|------|--------|-----------------|
| 1 | Run `dfir-cli enrichment lookup --ip 8.8.8.8 --api-url http://localhost:9999 -v` (assuming a local server returning 500) | Verbose output shows retry attempts with exponential backoff. After 3 retries (4 total attempts), error: "request failed after 4 attempts: API error (500): ...". Exit code 1. |

**Notes:** Requires a mock server or accept that this tests the retry path. Can also be verified by pointing to a non-existent URL.

**Pass criteria:** 5xx triggers retries. After exhausting retries, clear error message.

---

### TC-UC6-005: Network timeout / unreachable server

**Priority:** High
**Category:** Error Handling
**Estimated Time:** 2 minutes
**Preconditions:** Valid API key configured

| Step | Action | Expected Result |
|------|--------|-----------------|
| 1 | Run `dfir-cli enrichment lookup --ip 8.8.8.8 --api-url https://192.0.2.1:9999 --timeout 3s` | After 3 seconds, error: "execute request: ..." with context deadline or connection refused. Exit code 1. No hang. |
| 2 | Run `dfir-cli enrichment lookup --ip 8.8.8.8 --api-url https://nonexistent.invalid.domain.example -v` | DNS resolution failure. Error message displayed. Exit code 1. |

**Pass criteria:** Timeout and DNS errors produce clear messages. No indefinite hang.

---

### TC-UC6-006: Verbose mode shows HTTP details

**Priority:** High
**Category:** Functional
**Estimated Time:** 2 minutes
**Preconditions:** Valid API key configured

| Step | Action | Expected Result |
|------|--------|-----------------|
| 1 | Run `dfir-cli enrichment lookup --ip 8.8.8.8 -v 2>&1 \| head -5` | Stderr contains: `[verbose] POST /enrichment/lookup (auth: sk-dfir-***...XXXX)` -- API key is redacted. On success: `[verbose] 200 OK (XXms, credits: N used, N remaining)`. |
| 2 | Verify API key redaction | Only prefix `sk-dfir-` and last 4 characters visible. Middle replaced with `***...`. |

**Pass criteria:** Verbose output shows HTTP method, path, redacted auth, status, timing, and credit info.

---

### TC-UC6-007: --no-color flag disables ANSI colors

**Priority:** Medium
**Category:** UI/UX
**Estimated Time:** 2 minutes
**Preconditions:** Valid API key configured. Terminal with color support.

| Step | Action | Expected Result |
|------|--------|-----------------|
| 1 | Run `dfir-cli enrichment lookup --ip 8.8.8.8 --no-color \| cat -v` | No ANSI escape sequences (`^[[`) in the output. Text is plain. |
| 2 | Run `NO_COLOR=1 dfir-cli enrichment lookup --ip 8.8.8.8 \| cat -v` | Same: no ANSI codes. The NO_COLOR env var is respected. |
| 3 | Run `DFIR_LAB_NO_COLOR=1 dfir-cli enrichment lookup --ip 8.8.8.8 \| cat -v` | Same: no ANSI codes. |

**Pass criteria:** All three methods (flag, NO_COLOR, DFIR_LAB_NO_COLOR) disable color output.

---

### TC-UC6-008: No API key configured

**Priority:** Critical
**Category:** Error Handling
**Estimated Time:** 2 minutes
**Preconditions:** Remove config and unset env vars: `rm -f ~/.config/dfir-cli/config.yaml; unset DFIR_LAB_API_KEY`

| Step | Action | Expected Result |
|------|--------|-----------------|
| 1 | Run `dfir-cli enrichment lookup --ip 8.8.8.8` | Error: "no API key configured. Run: dfir-cli config init". Exit code 1. |
| 2 | Run `dfir-cli phishing analyze --file sample-phish.eml` | Same error. Exit code 1. |
| 3 | Run `dfir-cli exposure scan --domain example.com` | Same error. Exit code 1. |

**Pass criteria:** All API commands fail with clear guidance when no key is set.

---

### TC-UC6-009: Credits command with no prior API call

**Priority:** Medium
**Category:** Edge Case
**Estimated Time:** 1 minute
**Preconditions:** Remove state file: `rm -f ~/.config/dfir-cli/state.json`

| Step | Action | Expected Result |
|------|--------|-----------------|
| 1 | Run `dfir-cli credits` | "No credit information available yet." followed by guidance to run any API command. Exit code 0. |
| 2 | Run `dfir-cli credits -o json` | JSON: `{"error":"no credit information available"}`. Exit code 0. |
| 3 | Run `dfir-cli credits -q` | Error (no output to stdout): "no credit information available". Exit code 1. |

**Pass criteria:** Graceful handling when no state file exists.

---

### TC-UC6-010: Credits command after an API call

**Priority:** High
**Category:** Functional
**Estimated Time:** 3 minutes
**Preconditions:** Valid API key configured

| Step | Action | Expected Result |
|------|--------|-----------------|
| 1 | Run `dfir-cli enrichment lookup --ip 8.8.8.8` | Successful lookup. |
| 2 | Run `dfir-cli credits` | "Credit Balance (as of last API call)" with: Credits Remaining, Last Used, Last Request timestamp. Note: "Credit balance is updated after each API operation." |
| 3 | Run `dfir-cli credits -o json \| jq .` | JSON with: `credits_remaining`, `last_credits_used`, `last_request_at`. |
| 4 | Run `dfir-cli credits -q` | Just the number (e.g., `950`). Exit code 0. |
| 5 | Verify state file: `ls -la ~/.config/dfir-cli/state.json` | File exists with 0600 permissions. |

**Pass criteria:** Credit state persisted and displayed correctly in all output modes.

---

### TC-UC6-011: Shell completion generation

**Priority:** Medium
**Category:** Functional
**Estimated Time:** 3 minutes
**Preconditions:** Binary installed

| Step | Action | Expected Result |
|------|--------|-----------------|
| 1 | Run `dfir-cli completion bash` | Bash completion script output to stdout. Non-empty. Contains `_dfir-cli` or similar function name. Exit code 0. |
| 2 | Run `dfir-cli completion zsh` | Zsh completion script output. Non-empty. Exit code 0. |
| 3 | Run `dfir-cli completion fish` | Fish completion script output. Non-empty. Exit code 0. |
| 4 | Run `dfir-cli completion powershell` | PowerShell completion script output. Non-empty. Exit code 0. |
| 5 | Run `dfir-cli completion` (no argument) | Error about requiring exactly 1 argument. Exit code 1. |
| 6 | Run `dfir-cli completion unsupported` | Error: "unsupported shell: unsupported". Exit code 1. |

**Pass criteria:** All four shells produce valid completion scripts. Missing/invalid arguments handled.

---

### TC-UC6-012: Update check command

**Priority:** Medium
**Category:** Functional
**Estimated Time:** 2 minutes
**Preconditions:** Binary is a release build (not `dev`)

| Step | Action | Expected Result |
|------|--------|-----------------|
| 1 | Run `dfir-cli update --check` | Stderr: "Current version: X.Y.Z". Checks for updates (spinner). Either "You are already running the latest version." (green) or "New version available: X.Y.Z -> A.B.C" with release notes link, followed by "Run 'dfir-cli update' to install the update." |
| 2 | Run `dfir-cli update` | Same check, but if an update exists, shows installation instructions for Homebrew, Linux, Windows, and Go install. |

**Pass criteria:** Update check completes. Instructions shown when update available.

---

### TC-UC6-013: Update command with dev build

**Priority:** Low
**Category:** Edge Case
**Estimated Time:** 1 minute
**Preconditions:** Running a development build (version = "dev")

| Step | Action | Expected Result |
|------|--------|-----------------|
| 1 | Run `dfir-cli update` (with dev build) | Error: "cannot update a development build. Install from a release instead". Exit code 1. |

**Pass criteria:** Dev builds cannot self-update.

---

### TC-UC6-014: --api-key flag security warning

**Priority:** High
**Category:** Security
**Estimated Time:** 1 minute
**Preconditions:** None

| Step | Action | Expected Result |
|------|--------|-----------------|
| 1 | Run `dfir-cli enrichment lookup --ip 8.8.8.8 --api-key sk-dfir-testkey1234567890ab 2>&1 \| head -3` | Stderr contains: "Warning: passing --api-key on the command line exposes it in process listings and shell history." and "Prefer: export DFIR_LAB_API_KEY=sk-dfir-..." |

**Pass criteria:** Security warning displayed when using --api-key flag.

---

### TC-UC6-015: Global --timeout flag

**Priority:** Medium
**Category:** Functional
**Estimated Time:** 2 minutes
**Preconditions:** Valid API key configured

| Step | Action | Expected Result |
|------|--------|-----------------|
| 1 | Run `dfir-cli enrichment lookup --ip 8.8.8.8 --timeout 2s --api-url https://192.0.2.1:9999 2>&1` | Request times out after approximately 2 seconds. Error message displayed. |
| 2 | Run `DFIR_LAB_TIMEOUT=3s dfir-cli enrichment lookup --ip 8.8.8.8 --api-url https://192.0.2.1:9999 2>&1` | Timeout from env var: approximately 3 seconds. |

**Pass criteria:** Timeout flag and env var respected.

---

### TC-UC6-016: Config profile name validation

**Priority:** Medium
**Category:** Input Validation
**Estimated Time:** 2 minutes
**Preconditions:** None

| Step | Action | Expected Result |
|------|--------|-----------------|
| 1 | Run `dfir-cli config list --profile ""` | Error: "profile name cannot be empty". |
| 2 | Run `dfir-cli config list --profile "has.dot"` | Error: "profile name cannot contain dots". |
| 3 | Run `dfir-cli config list --profile "has space"` | Error: "profile name cannot contain whitespace". |
| 4 | Run `dfir-cli config list --profile "$(python3 -c "print('a'*65)")"` | Error: "profile name too long (max 64 characters)". |

**Pass criteria:** Invalid profile names rejected with descriptive errors.

---

### TC-UC6-017: Ctrl+C (SIGINT) cancellation

**Priority:** High
**Category:** Functional
**Estimated Time:** 3 minutes
**Preconditions:** Valid API key configured

| Step | Action | Expected Result |
|------|--------|-----------------|
| 1 | Run `dfir-cli exposure scan --domain example.com` and press Ctrl+C during the spinner | Scan is cancelled. Spinner stops. Process exits promptly (within 1-2 seconds). No zombie processes. |
| 2 | Run `dfir-cli enrichment lookup --batch /tmp/test-iocs.txt` and press Ctrl+C mid-batch | Processing stops. Partial results may or may not be shown. Process exits cleanly. |

**Pass criteria:** SIGINT handled gracefully. No hang, no panic.

---

### TC-UC6-018: Double flag specification

**Priority:** Low
**Category:** Edge Case
**Estimated Time:** 2 minutes
**Preconditions:** Valid API key configured

| Step | Action | Expected Result |
|------|--------|-----------------|
| 1 | Run `dfir-cli enrichment lookup --ip 1.2.3.4 --domain evil.com` | The first matching typed flag (--ip) takes precedence. Lookup performed for 1.2.3.4 only. |
| 2 | Run `dfir-cli enrichment lookup --ip 1.2.3.4 --ioc evil.com` | Typed flag --ip takes precedence over --ioc. Lookup for 1.2.3.4. |

**Notes:** The code checks typed flags in order (ip, domain, url, hash, email) before --ioc.

**Pass criteria:** Precedence: typed flag > --ioc > --batch > stdin.

---

### TC-UC6-019: Enrichment with --batch pointing to nonexistent file

**Priority:** Medium
**Category:** Negative
**Estimated Time:** 1 minute
**Preconditions:** None

| Step | Action | Expected Result |
|------|--------|-----------------|
| 1 | Run `dfir-cli enrichment lookup --batch /tmp/does-not-exist.txt` | Error: "reading batch file: open /tmp/does-not-exist.txt: no such file or directory". Exit code 1. |

**Pass criteria:** Clear file-not-found error.

---

### TC-UC6-020: API URL override via flag and env

**Priority:** Medium
**Category:** Functional
**Estimated Time:** 2 minutes
**Preconditions:** Valid API key configured

| Step | Action | Expected Result |
|------|--------|-----------------|
| 1 | Run `dfir-cli enrichment lookup --ip 8.8.8.8 --api-url https://custom-api.example.com/api/v1 -v 2>&1` | Verbose output shows request going to `https://custom-api.example.com/api/v1/enrichment/lookup`. (Will likely fail but validates the URL is used.) |
| 2 | Run `DFIR_LAB_API_URL=https://env-api.example.com/api/v1 dfir-cli enrichment lookup --ip 8.8.8.8 -v 2>&1` | Verbose output shows the env URL being used. |

**Pass criteria:** API URL override works via both flag and environment variable.

---

### TC-UC6-021: no-color accepted values for config set

**Priority:** Low
**Category:** Input Validation
**Estimated Time:** 2 minutes
**Preconditions:** Config initialized

| Step | Action | Expected Result |
|------|--------|-----------------|
| 1 | Run `dfir-cli config set no-color true` | Accepted. |
| 2 | Run `dfir-cli config set no-color 1` | Accepted (truthy). |
| 3 | Run `dfir-cli config set no-color yes` | Accepted (truthy). |
| 4 | Run `dfir-cli config set no-color false` | Accepted. |
| 5 | Run `dfir-cli config set no-color 0` | Accepted (falsy). |
| 6 | Run `dfir-cli config set no-color no` | Accepted (falsy). |

**Pass criteria:** Boolean-like values (true/false/1/0/yes/no) accepted case-insensitively.

---

### TC-UC6-022: Config file permission security

**Priority:** High
**Category:** Security
**Estimated Time:** 2 minutes
**Preconditions:** Config initialized

| Step | Action | Expected Result |
|------|--------|-----------------|
| 1 | Run `stat -f "%Lp" ~/.config/dfir-cli/config.yaml` (macOS) or `stat -c "%a" ~/.config/dfir-cli/config.yaml` (Linux) | Returns `600`. |
| 2 | Run `chmod 644 ~/.config/dfir-cli/config.yaml` then `dfir-cli config set no-color true` | After the set operation, check permissions again. File should be back to `600` (atomic write with enforced permissions). |
| 3 | Verify directory: `stat -f "%Lp" ~/.config/dfir-cli/` (macOS) or `stat -c "%a" ~/.config/dfir-cli/` (Linux) | Returns `700`. |

**Pass criteria:** Config file always written with 0600; directory with 0700. Sensitive data (API key) protected from other users.

---

### TC-UC6-023: Stdin read size limit (10 MB)

**Priority:** Low
**Category:** Edge Case
**Estimated Time:** 2 minutes
**Preconditions:** None

| Step | Action | Expected Result |
|------|--------|-----------------|
| 1 | Generate 11MB of data: `dd if=/dev/zero bs=1M count=11 \| dfir-cli enrichment lookup --type ip 2>&1` | The CLI reads up to 10 MB from stdin (LimitReader). Data beyond 10 MB is truncated. Behavior depends on whether the truncated data forms valid IOCs, but no OOM or crash should occur. |

**Notes:** The `maxStdinSize` constant is 10 MB. This test verifies the safety limit.

**Pass criteria:** No crash, no excessive memory usage. Graceful handling of large stdin.

---

### TC-UC6-024: Enrichment batch chunking (more than 10 IOCs)

**Priority:** Medium
**Category:** Functional
**Estimated Time:** 4 minutes
**Preconditions:** Valid API key configured. Create a file with >10 IOCs.

| Step | Action | Expected Result |
|------|--------|-----------------|
| 1 | Create `/tmp/many-iocs.txt` with 15 IP addresses (e.g., `1.1.1.1` through `1.1.1.15`, one per line) | File created with 15 lines. |
| 2 | Run `dfir-cli enrichment lookup --batch /tmp/many-iocs.txt -v 2>&1` | Verbose output shows two API calls: first with 10 indicators, second with 5 indicators. All 15 results displayed. |

**Pass criteria:** IOCs chunked into batches of 10 (`batchAPILimit`). All results aggregated.

---

## Test Summary

| Use Case | Test Cases | Critical | High | Medium | Low |
|----------|-----------|----------|------|--------|-----|
| UC1: Installation & First Run | 5 | 2 | 1 | 1 | 1 |
| UC2: Configuration & API Key | 11 | 3 | 5 | 2 | 1 |
| UC3: IOC Enrichment | 18 | 5 | 8 | 4 | 1 |
| UC4: Phishing Analysis | 13 | 2 | 5 | 5 | 1 |
| UC5: Exposure Scanning | 11 | 1 | 5 | 4 | 1 |
| UC6: Error Handling & Edge Cases | 24 | 3 | 10 | 8 | 3 |
| **Total** | **82** | **16** | **34** | **24** | **8** |

## Suggested Execution Order

1. UC1 (Installation) -- verify binary works at all
2. UC2 (Configuration) -- set up config needed by other tests
3. UC6-008, UC6-001 (No key / invalid key) -- verify auth gating
4. UC3 (Enrichment) -- core functionality
5. UC4 (Phishing) -- core functionality
6. UC5 (Exposure) -- core functionality (longest running due to scan times)
7. UC6 (remaining) -- error handling and edge cases

## Estimated Total Testing Time

- Critical tests only: ~45 minutes
- Critical + High: ~90 minutes
- Full suite: ~120 minutes (excluding exposure scan wait times which can add 30+ minutes)

## Test Data Requirements

| Item | Description | Location |
|------|-------------|----------|
| Valid API key | `sk-dfir-...` format, with sufficient credits | Stored in config or env |
| Invalid API key | `sk-dfir-invalidkey00000000` | Used via --api-key flag |
| Depleted API key | Key with 0 credits remaining | For UC6-002 tests |
| sample-phish.eml | A phishing email in .eml format | Project test fixtures |
| test-iocs.txt | Batch file with mixed IOC types | Created during testing |
| test-domains.txt | Batch file with domain names | Created during testing |
| many-iocs.txt | 15+ IOCs for chunking test | Created during testing |

## Known Limitations

- Rate limit tests (TC-UC6-003) are hard to reproduce reliably without knowing exact API thresholds
- Server error retry tests (TC-UC6-004) ideally require a mock server
- Exit codes for verdict-based tests depend on actual API classification of test indicators
- Exposure scan tests take 1-3 minutes each due to API processing time
- The `config init` hidden input test (TC-UC2-001 step 2) cannot be fully automated in CI since it requires a real TTY
