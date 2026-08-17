# Encrata CLI

The official CLI for Encrata.

Built for email intelligence lookups, validation workflows, async bulk jobs,
contact list management, automation scripts, and local developer testing.

---

## Install

### PowerShell (Windows)

```powershell
irm https://raw.githubusercontent.com/Encratahq/encrata-cli/main/install.ps1 | iex
```

Install a specific version:

```powershell
& ([scriptblock]::Create((irm https://raw.githubusercontent.com/Encratahq/encrata-cli/main/install.ps1))) -Version 0.4.7
```

### Homebrew (macOS / Linux)

```bash
brew tap Encratahq/tap
brew install encrata
```

Or:

```bash
brew install Encratahq/tap/encrata
```

### Node.js

```bash
npm install -g encrata-cli
```

### Manual download

Download a binary from the GitHub releases page:

```text
https://github.com/Encratahq/encrata-cli/releases
```

Windows users can download a `windows_amd64.zip` or `windows_arm64.zip` asset,
extract it, and run `encrata.exe`.

---

## Quickstart

```bash
# Verify the install
encrata version

# Save your API key
encrata config set-key YOUR_API_KEY

# Run your first lookup
encrata email validity user@example.com

# Get the full API response
encrata email enrich user@example.com --json
```

---

## Authentication

Fastest path — `login` saves your key and verifies it:

```bash
encrata login enc_live_xxxxxxxx     # or just: encrata login  (prompts)
encrata whoami                      # confirm account, plan, credits, workspace
encrata logout                      # clear the saved key
```

The CLI resolves your API key using this priority chain:

| Priority | Source | How to set |
| -------- | ------ | ---------- |
| 1 | `--api-key` flag | `encrata email validity user@example.com --api-key YOUR_API_KEY` |
| 2 | `ENCRATA_API_KEY` env var | `export ENCRATA_API_KEY=YOUR_API_KEY` |
| 3 | Config file | `encrata login YOUR_API_KEY` (or `config set-key`) |

If no key is found, protected commands return an API key error.

Config is saved to `~/.encrata/config.yaml` (written `0600`).

### `encrata login` / `logout` / `whoami`

| Command | Description |
| ------- | ----------- |
| `login [api-key]` | Save the key and verify it against the API (prompts if omitted) |
| `logout` | Clear the saved key (the `ENCRATA_API_KEY` env var still applies) |
| `whoami` | Show the authenticated email, plan, remaining credits, role, and active workspace |

---

## Conventions

Behavior is consistent across every command:

| Situation | Output |
| --------- | ------ |
| Single lookup | Human-readable card (key/value) on **stdout** |
| Bulk (`--bulk`) | Progress bar + **summary counts** (not a per-row dump) |
| `--json` | Raw API JSON on **stdout** (full rows for bulk) |
| `--out file` | Results written to `.csv` / `.xlsx` / `.json` |
| Errors & notices | **stderr** (so stdout stays pipe-clean) |

Global flags (all commands): `--json`, `--api-key`, `--base-url`, `--quiet`,
`--no-color` (honors `NO_COLOR`), `--timeout <secs>`. The spinner auto-disables
when output is not a terminal.

### Exit codes

| Code | Meaning |
| ---- | ------- |
| `0` | Success (or a finding when `--fail-on-finding` is not set) |
| `1` | Operational error (bad usage, network, or server) |
| `2` | A finding was detected — breach/leak — only with `--fail-on-finding` |
| `3` | Authentication failed |
| `4` | Insufficient credits |

---

## Configuration

### `encrata config set-key`

Save your API key locally.

```bash
encrata config set-key YOUR_API_KEY
```

### `encrata config set-url`

Use a custom API server, for example a local backend.

```bash
encrata config set-url http://localhost:8080
```

### `encrata config show`

Show the current CLI configuration.

```bash
encrata config show
```

### `encrata config unset`

Clear a saved value and revert it to its default.

```bash
encrata config unset api-key     # remove the saved key
encrata config unset base-url    # revert to the default API URL
encrata config unset output      # revert to table output
```

### Global options

| Flag | Description |
| ---- | ----------- |
| `--json` | Print raw JSON output |
| `--api-key` | Override the saved API key |
| `--base-url` | Override the saved API base URL |
| `--quiet` | Suppress decorative output (headers, spinner); results still print |
| `--no-color` | Disable colored output (also honors the `NO_COLOR` env var) |
| `--timeout` | Per-request timeout in seconds (0 = default 90s) |

Environment variables: `ENCRATA_API_KEY`, `ENCRATA_BASE_URL`, and `NO_COLOR`
are all honored. The config file (`~/.encrata/config.yaml`) is written with
`0600` permissions since it stores the API key. The spinner auto-disables when
output is not a terminal.

---

## Help

Use help to discover commands, flags, and examples from the terminal.

```bash
encrata help
encrata --help
encrata COMMAND --help
encrata COMMAND SUBCOMMAND --help
```

Examples:

```bash
encrata email --help
encrata email bulk --help
encrata jobs --help
encrata keys --help
encrata webhooks --help
encrata workspace --help
encrata lists --help
```

---

## Commands

### `encrata version`

Print the installed CLI version.

```bash
encrata version
```

---

### `encrata update`

Update the CLI binary to the latest GitHub release.

```bash
encrata update
```

---

### Which email command?

| Command | Use it for | Credits |
| ------- | ---------- | ------- |
| `email validity` | Deliverability verdict (valid/invalid/catch-all/risky) **+ full report** | 1/email |
| `email verify` | Quick deep-SMTP yes/no — is the mailbox reachable? | Free |
| `email enrich` | Validity **+ company & domain** signals (the `validity --full` data) | 1/email |
| `email identity` | The **person** — name, role, company, socials, breaches | 1000/email |
| `email breaches` | Data-breach exposure for an address | 1/email |
| `email bulk` | Validate a whole file/list (= `email validity --bulk`) | 1/email |

API endpoint family: these commands now target `/api/cli/*` paths (`/api/cli/email-validity`, `/api/cli/email-enrich`, `/api/cli/email-identity`, `/api/cli/breaches`, `/api/cli/email-verify`, `/api/cli/email-validity-bulk`) rather than legacy `/api/agent/*` aliases.

Every verb runs on one address, or on a file/STDIN list with `--bulk`.

---

### `encrata email validity`

Check whether a single email address is valid and deliverable. Returns
`valid` / `invalid` / `catch-all` / `risky` plus confidence, domain trust,
disposable/role flags, provider and SMTP detail.

```bash
encrata email validity user@example.com
encrata email validity user@example.com --json

# Validate a whole file (or STDIN) — same engine as `email bulk`
encrata email validity emails.csv --bulk
encrata email validity emails.csv --bulk --out results.csv --only valid
```

Bulk mode (`--bulk`) is the uniform spelling of `email bulk` and takes the same
flags: `--out`, `--format`, `--only valid|invalid|found`, `--stream`, `--job`,
`--enrich`, `--concurrency`, `--columns`.

---

### `encrata email enrich`

Validate an email and enrich it with person and company data.

```bash
encrata email enrich user@example.com
encrata email enrich user@example.com --json
```

---

### `encrata email identity`

Resolve the identity and social profiles behind an email address. Pass a single
email, or use `--bulk` with a file (or `-` for STDIN) to resolve a whole list
concurrently.

```bash
# Single email
encrata email identity user@example.com
encrata email identity user@example.com --json

# Bulk from a file (or STDIN)
encrata email identity emails.csv --bulk
encrata email identity emails.csv --bulk --concurrency 16 --out people.csv
encrata email identity emails.csv --bulk --only found --out people.xlsx
```

Bulk options: `--bulk`, `--concurrency` (default 8), `--out` (`.csv`/`.xlsx`/`.json`),
`--format`, `--only found`. Bulk exports flatten to `email`, `found`, `name`,
`company`, `job_role`, `location`; `--format json` writes the raw objects. For
very large lists, prefer the async `jobs --type identity` surface.

---

### `encrata email breaches`

Check whether an email appears in known data breaches. Pass a single email to
check it, or use `--bulk` with a CSV/text file (or `-` for STDIN) to stream a
whole list with a live progress bar.

```bash
# Single email
encrata email breaches user@example.com
encrata email breaches user@example.com --json

# Bulk from a file (or STDIN)
encrata email breaches emails.csv --bulk
cat emails.csv | encrata email breaches - --bulk

# Bulk with an export
encrata email breaches emails.csv --bulk --out breaches.csv
encrata email breaches emails.csv --bulk --out breaches.xlsx
encrata email breaches emails.csv --bulk --format json
```

Options:

| Flag | Description |
| ---- | ----------- |
| `--full` | Show exposed data and registered services (single-email mode) |
| `--bulk` | Check a file (or `-` for STDIN) of emails via streaming |
| `--out` | Write bulk results to a file (`.csv`, `.xlsx`, or `.json`) |
| `--format` | Bulk export format: `csv`, `xlsx`, or `json` (default: inferred from `--out`) |
| `--only` | Export only matching rows: `breached` |
| `--fail-on-finding` | Exit with code 2 if any email is breached (otherwise exit 0) |
| `--json` | Print raw JSON output |

Bulk exports flatten each row to `email`, `breached`, `breach_count`, and
`breaches` (a ` | `-joined list of breach names); `--format json` writes the raw,
nested result objects instead.

Exit codes (usable as a CI / sign-up guard, mirroring `encrata password`):

| Code | Meaning |
| ---- | ------- |
| `0` | Success — or a breach was found but `--fail-on-finding` was not set |
| `1` | Operational error (network, validation, or server) |
| `2` | A breach was found (only with `--fail-on-finding`) |
| `3` | Authentication failed |
| `4` | Insufficient credits |

---

### `encrata email verify`

Deep SMTP-level deliverability check of a single address — connects to the mail
server to confirm the mailbox. **Free.** Use `verify` for a quick
deliverable/undeliverable answer; use `validity` when you also need the full
report (confidence, domain trust, disposable/role flags, provider, SMTP detail
— costs 1 credit).

```bash
encrata email verify user@example.com
encrata email verify user@example.com --json
```

---

### `encrata password`

Check whether a password has appeared in known data breaches (HIBP
k-anonymity). Your password is hashed locally with SHA-1 and **only the hash is
sent** — the plaintext never leaves your machine and is never logged, cached, or
stored.

API endpoint family: these commands target `/api/cli/password-breaches` and
`/api/cli/password-breaches/bulk`.

```bash
# Prompt interactively (no echo — never lands in shell history)
encrata password

# Pass a password as an argument
encrata password 'hunter2'

# Bulk-check a file (one password per line, de-duplicated, max 1000)
encrata password --file passwords.txt

# Bulk-check from STDIN
cat passwords.txt | encrata password --stdin

# Raw JSON output
encrata password 'hunter2' --json
```

Options:

| Flag | Description |
| ---- | ----------- |
| `--file` | Check passwords from a file (one per line) |
| `--stdin` | Read passwords from STDIN (one per line) |
| `--fail-on-finding` | Exit with code 2 if any password is breached (otherwise exit 0) |
| `--json` | Print raw JSON output (defaults to a formatted table) |

Pricing: **1 credit per unique password** (single = 1; bulk = number of unique
hashes).

Exit codes (usable as a CI / sign-up guard):

| Code | Meaning |
| ---- | ------- |
| `0` | Success — or a breach was found but `--fail-on-finding` was not set |
| `1` | Operational error (network, validation, or server) |
| `2` | A breach was found (only with `--fail-on-finding`) |
| `3` | Authentication failed |
| `4` | Insufficient credits |

---


### `encrata email bulk`

Validate a batch of emails from a CSV/text file or STDIN. Small batches stream
live results over the terminal with a progress bar; large batches (or `--job`)
run as an async job that is polled to completion.

By default, bulk validation uses the lean, high-throughput path and returns
`email`, `status`, and `reason` per row. Add `--enrich` to run the full
per-email report instead, so every export column (provider, mx, trust grade,
breaches, etc.) is populated — the same data as `email validity --full`.

```bash
encrata email bulk emails.csv
encrata email bulk emails.csv --out results.csv
encrata email bulk emails.csv --enrich --out results.csv
encrata email bulk emails.csv --enrich --concurrency 16 --only valid --out valid.csv
encrata email bulk emails.csv --out results.xlsx --columns email,status,trust_grade
encrata email bulk emails.csv --job --out results.json --format json
cat emails.csv | encrata email bulk - --stream
```

> `email bulk emails.csv` and `email validity emails.csv --bulk` are equivalent
> — same engine and flags. Use whichever reads better.

Options:

| Flag | Description |
| ---- | ----------- |
| `--stream` | Force live streaming (SSE) mode |
| `--job` | Force async job mode |
| `--enrich` | Run the full per-email report so every column is filled (1 credit per email) |
| `--concurrency` | Parallel lookups when `--enrich` is set (default 8) |
| `--out` | Write results to a file (`.csv`, `.xlsx`, or `.json`) |
| `--format` | Export format: `csv`, `xlsx`, or `json` (default: inferred from `--out`) |
| `--columns` | Subset of columns to export (`email`, `status`, `reason` always included) |
| `--only` | Export only matching rows: `valid`, `invalid`, or `found` |

Batches larger than 1,000 emails automatically switch to job mode unless
`--stream` is set. The lean path bills 1 credit per successful unique email;
`--enrich` bills 1 credit per email (full report), so it is opt-in.

#### Export columns

CSV and XLSX exports flatten each result into a fixed, ordered column set.
Booleans render as `yes`/`no` and lists join with ` | `. `email`, `status` and
`reason` are always present; the rest are empty when not returned:

```text
email, status, reason, message, confidence, disposable, role, role_name,
free_provider, provider, did_you_mean, canonical, domain, mx, smtp_mx_host,
smtp_catch_all, smtp_greylisted, trust_grade, spf, dmarc, dmarc_policy, dkim,
mta_sts, tls_rpt, bimi, dnssec, person_signal_count, person_signal_sources,
registrar, domain_created_at, domain_age_days, breaches_count, gravatar,
registered_services, google_account, checked_at
```

Use `--columns email,status,trust_grade` to pick a subset. `--format json`
writes the raw, nested result objects (unflattened) instead of a flat table.

---


### `encrata jobs`

Manage asynchronous jobs for large inputs the backend processes in the
background. **One namespace, three job types** selected with `--type`
(`validity` is the default; also `identity` and `password`).

```bash
# Validity (default) — from a file or STDIN
encrata jobs create emails.csv
encrata jobs list
encrata jobs status JOB_ID
encrata jobs results JOB_ID --status invalid
encrata jobs download JOB_ID --format csv --out results.csv
encrata jobs cancel JOB_ID
encrata jobs retry JOB_ID

# Identity — same verbs, add --type identity
encrata jobs create emails.csv --type identity
encrata jobs results JOB_ID --type identity --found-only
encrata jobs download JOB_ID --type identity --out people.csv

# Password — hashes only, never plaintext
encrata jobs create --type password --sha1-file hashes.txt
encrata jobs create --type password --password-file passwords.txt   # hashed locally
encrata jobs results JOB_ID --type password --breached
```

Every verb (`create`, `list`, `status`, `results`, `download`, `cancel`,
`retry`) accepts `--type validity|identity|password`.

Options:

| Command | Flag | Description |
| ------- | ---- | ----------- |
| all | `--type` | Job type: `validity` (default), `identity`, `password` |
| `create` (password) | `--sha1s` / `--sha1-file` / `--password-file` | Hash sources (plaintext hashed locally, never sent) |
| `create` | `--file-name` | Optional display name |
| `results` | `--status` | Validity: filter by per-row status |
| `results` | `--page` / `--page-size` | Pagination |
| `results` | `--found-only` | Identity: only enriched rows |
| `results` | `--breached` | Password: only breached rows |
| `download` | `--format` | Validity: `csv`, `xlsx`, or `json` |
| `download` | `--valid-only` / `--found-only` / `--breached` | Per-type row filters |
| `download` | `--out` | Write to a file instead of stdout |
| `retry` | | Re-drive dead-lettered chunks (validity/identity; password has no retry) |

> **Note:** `jobs <verb> --type` is the single async-job surface. The former
> standalone `identity-jobs` / `password-jobs` groups and the MCP-style names
> (`bulk-validate-emails`, `get-email-job-status`, …) have been removed.

---

### `encrata lists`

Manage reusable contact lists — named collections of emails you can share across
enrichment and monitoring workflows.

API endpoint family: these commands target `/api/cli/lists*`.

```bash
encrata lists list
encrata lists create "Prospects" --emails a@example.com --emails b@example.com
encrata lists create "Q3 Leads" --file emails.csv
encrata lists get LIST_ID
encrata lists emails LIST_ID
encrata lists add LIST_ID --emails new@example.com
encrata lists remove LIST_ID --emails old@example.com
encrata lists delete LIST_ID
```

Subcommands:

| Command | Aliases | Description |
| ------- | ------- | ----------- |
| `list` | `ls` | List all contact lists |
| `create` | | Create a list, optionally seeding emails |
| `get` | `show` | Show a list's details |
| `emails` | | List every email in a list |
| `add` | | Add emails to a list |
| `remove` | `rm-emails` | Remove emails from a list |
| `delete` | `rm`, `del` | Delete a list permanently |

Options:

| Command | Flag | Description |
| ------- | ---- | ----------- |
| `create` | `--emails` | Initial email addresses (repeatable) |
| `create` | `--file` | Read initial emails from a file |
| `add` | `--emails` / `--file` | Emails to add |
| `remove` | `--emails` / `--file` | Emails to remove |

---

### `encrata keys`

Create, list, rename, enable/disable, cap, and revoke API keys.

```bash
encrata keys ls
encrata keys create "CI key"
encrata keys rename KEY_ID "Prod key"
encrata keys enable KEY_ID
encrata keys disable KEY_ID
encrata keys limit KEY_ID --credits 5000
encrata keys limit KEY_ID --unlimited
encrata keys revoke KEY_ID
encrata keys revoke KEY_ID --permanent
```

Subcommands:

| Command | Description |
| ------- | ----------- |
| `ls` | List API keys with ID, name, prefix, status, credits used, and limit |
| `create` | Create a new API key (the full key is shown once) |
| `rename` | Rename a key: `keys rename <id> <new-name>` |
| `enable` | Re-enable a disabled key |
| `disable` | Disable a key without deleting it (reversible with `enable`) |
| `limit` | Set (`--credits <N>`) or clear (`--unlimited`) a key's credit cap |
| `revoke` | Disable a key; add `--permanent` to delete it for good |

Options:

| Command | Flag | Description |
| ------- | ---- | ----------- |
| `limit` | `--credits` | Credit cap for the key (must be >= 0) |
| `limit` | `--unlimited` | Remove the credit cap (unlimited usage) |
| `revoke` | `--permanent` | Permanently delete the key instead of disabling it |

---

### `encrata webhooks`

Register HTTPS endpoints that receive real-time event notifications from your
workspace. Deliveries are signed with HMAC-SHA256 so your receiver can verify
them.

```bash
encrata webhooks create https://example.com/hook --events lookup.completed,credits.low
encrata webhooks list
encrata webhooks update WEBHOOK_ID --disable
encrata webhooks test WEBHOOK_ID
encrata webhooks deliveries WEBHOOK_ID --limit 20
encrata webhooks delete WEBHOOK_ID
```

Subcommands: `list`, `create`, `update`, `delete`, `test`, `deliveries`.

Valid event types:

```text
lookup.completed, apikey.created, apikey.revoked, credits.low, credits.exhausted
```

Options:

| Command | Flag | Description |
| ------- | ---- | ----------- |
| `create` | `--events` | Events to subscribe to (comma-separated, required) |
| `create` | `--description` | Optional description |
| `update` | `--url` | New HTTPS endpoint URL |
| `update` | `--events` | Replace subscribed events (comma-separated) |
| `update` | `--description` | New description |
| `update` | `--enable` / `--disable` | Activate or deactivate the webhook |
| `delete` | `--yes` | Skip the confirmation prompt |
| `deliveries` | `--limit` | Maximum deliveries to show (default 20) |

`create` returns the signing **secret once** — store it as `ENCRATA_WEBHOOK_SECRET`
in your receiver to verify the `X-Encrata-Signature` header. URLs must use HTTPS.
Managing webhooks requires a selected workspace.

---

### `encrata workspace`

Manage workspaces and their members (alias: `ws`). Member, `update`, and
`delete` operations act on your **current (active) workspace**, which is tracked
server-side — run `workspace switch <id>` first to change it.

Workspace commands call the CLI workspace namespace: `/api/cli/workspaces*`.

```bash
encrata workspace list
encrata workspace create "Acme Inc" --slug acme
encrata workspace switch WORKSPACE_ID
encrata workspace update --name "Acme Corp" --slug acme
encrata workspace delete            # deletes the CURRENT workspace (admin only)

encrata workspace members
encrata workspace members invite teammate@acme.com --role tech
encrata workspace members set-role MEMBER_ID --role admin
encrata workspace members remove MEMBER_ID
```

Subcommands: `list`, `create`, `switch`, `update`, `delete`, and
`members` (`list`, `invite`, `set-role`, `remove`).

Valid member roles: `admin`, `tech`, `readonly` (the creator is the owner).

Options:

| Command | Flag | Description |
| ------- | ---- | ----------- |
| `create` | `--slug` | Custom slug (auto-generated when omitted) |
| `create` | `--logo-url` | Logo URL |
| `update` | `--name` | New name (**required**) |
| `update` | `--slug` | New slug (regenerated from name if omitted) |
| `update` | `--logo-url` | New logo URL |
| `update` | `--id` | Target workspace ID (defaults to current) |
| `delete` | `--yes` | Skip the confirmation prompt |
| `members invite` | `--role` | `admin`, `tech`, or `readonly` (default `readonly`) |
| `members set-role` | `--role` | New role for the member |
| `members remove` | `--yes` | Skip the confirmation prompt |

`update`, `delete`, and member changes are **admin-only** and always target your
active workspace. Destructive actions (`delete`, `members remove`) confirm first
unless `--yes`.

> The `workspace` command is also available as **`ws`** — e.g. `ws list`,
> `ws switch <id>`, `ws members list`.

---

## Shell completion

Generate a completion script for your shell:

```bash
encrata completion bash   > /etc/bash_completion.d/encrata
encrata completion zsh    > "${fpath[1]}/_encrata"
encrata completion fish   > ~/.config/fish/completions/encrata.fish
encrata completion powershell | Out-String | Invoke-Expression
```

Run `encrata completion --help` for shell-specific install instructions.

---

## Credits

Billing is 1 credit per successful unique email. Duplicate emails are
de-duplicated, and invalid or failed checks are not charged.

| Command | Credits |
| ------- | ------- |
| `email validity` | 1 credit per successful email |
| `email enrich` | 1 credit per successful email |
| `email identity` | 1 credit per successful email |
| `email breaches` | 1 credit per successful email (single or `--bulk`) |
| `email verify` | 1 credit per successful email |
| `email bulk` | 1 credit per successful unique email |
| `jobs` (validity / identity) | 1 credit per successful unique email |
| `jobs --type password` | 1 credit per unique password hash |
| `lists` | Free — list management does not consume credits |
| `keys` | Free — key management does not consume credits |
| `webhooks` | Free — webhook management does not consume credits |
| `workspace` | Free — workspace management does not consume credits |

---

## Local development

Use this when you want to change the CLI and run your local build.

### Prerequisites

- Go 1.25.4+
- An Encrata API key
- Optional: a local Encrata backend running on `http://localhost:8080`

### Setup

1. Clone the repo.

```bash
git clone https://github.com/Encratahq/encrata-cli.git
cd encrata-cli
```

2. Run tests.

```bash
go test ./...
```

3. Run the CLI locally.

```bash
go run . version
go run . email validity user@example.com
```

4. Point local runs at a local backend.

```powershell
$env:ENCRATA_BASE_URL = "http://localhost:8080"
go run . email enrich user@example.com
```

### Build locally

```bash
go build -o encrata .
```

On Windows:

```powershell
go build -o encrata.exe .
```

Output: `./encrata` or `./encrata.exe`.

---

## Links

- Documentation: https://docs.encrata.com/cli
- API Reference: https://docs.encrata.com/api-reference
- Dashboard: https://encrata.com
- Releases: https://github.com/Encratahq/encrata-cli/releases

---

## License

MIT
