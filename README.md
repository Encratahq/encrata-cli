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

The CLI resolves your API key using this priority chain:

| Priority | Source | How to set |
| -------- | ------ | ---------- |
| 1 | `--api-key` flag | `encrata email validity user@example.com --api-key YOUR_API_KEY` |
| 2 | `ENCRATA_API_KEY` env var | `export ENCRATA_API_KEY=YOUR_API_KEY` |
| 3 | Config file | `encrata config set-key YOUR_API_KEY` |

If no key is found, protected commands return an API key error.

Config is saved to:

```text
~/.encrata/config.yaml
```

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

### Global options

| Flag | Description |
| ---- | ----------- |
| `--json` | Print raw JSON output |
| `--api-key` | Override the saved API key |
| `--base-url` | Override the saved API base URL |

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

### `encrata email validity`

Check whether a single email address is valid and deliverable.

```bash
encrata email validity user@example.com
encrata email validity user@example.com --json
```

---

### `encrata email enrich`

Validate an email and enrich it with person and company data.

```bash
encrata email enrich user@example.com
encrata email enrich user@example.com --json
```

---

### `encrata email identity`

Resolve the identity and social profiles behind an email address.

```bash
encrata email identity user@example.com
encrata email identity user@example.com --json
```

---

### `encrata email breaches`

Check whether an email appears in known data breaches.

```bash
encrata email breaches user@example.com
encrata email breaches user@example.com --json
```

---

### `encrata email verify`

Perform a deep SMTP verification of an email address.

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
| `--json` | Print raw JSON output (defaults to a formatted table) |

Pricing: **1 credit per unique password** (single = 1; bulk = number of unique
hashes).

Exit codes (usable as a CI / sign-up guard):

| Code | Meaning |
| ---- | ------- |
| `0` | Check succeeded and nothing was breached |
| `1` | Check succeeded and at least one password was breached |
| non-zero | Auth, credit, network, or validation error |

---


### `encrata email bulk`

Validate a batch of emails from a CSV/text file or STDIN. Small batches stream
live results over the terminal with a progress bar; large batches (or `--job`)
run as an async job that is polled to completion.

```bash
encrata email bulk emails.csv
encrata email bulk emails.csv --out results.csv
encrata email bulk emails.csv --out results.xlsx --columns email,status,trust_grade
encrata email bulk emails.csv --job --out results.json --format json
cat emails.csv | encrata email bulk - --stream
```

Options:

| Flag | Description |
| ---- | ----------- |
| `--stream` | Force live streaming (SSE) mode |
| `--job` | Force async job mode |
| `--out` | Write results to a file (`.csv`, `.xlsx`, or `.json`) |
| `--format` | Export format: `csv`, `xlsx`, or `json` (default: inferred from `--out`) |
| `--columns` | Subset of columns to export (`email`, `status`, `reason` always included) |
| `--found-only` | Skip rows that carry no enrichment data |

Batches larger than 1,000 emails automatically switch to job mode unless
`--stream` is set.

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

Manage asynchronous email jobs. Use these for large inputs that the backend
processes in the background and returns as downloadable results. The command
covers three job types — `validity`, `identity`, and `password` — plus the
underlying bulk-job registry.

#### Validity jobs (file-based)

```bash
encrata jobs create emails.csv
encrata jobs list
encrata jobs status JOB_ID
encrata jobs results JOB_ID --status invalid
encrata jobs download JOB_ID --format csv --out results.csv
encrata jobs cancel JOB_ID
```

Options:

| Command | Flag | Description |
| ------- | ---- | ----------- |
| `results` | `--status` | Filter results by per-row status (e.g. `valid`, `invalid`) |
| `results` | `--page` | Result page to fetch |
| `download` | `--format` | Download format: `csv`, `xlsx`, or `json` |
| `download` | `--status` | Filter rows by status |
| `download` | `--valid-only` | Download only rows whose status is valid |
| `download` | `--out` | Write to a file instead of stdout |

#### Create async jobs (inline input)

Create validity, identity, or password jobs directly from inline values or a
file — no separate upload step required.

```bash
# Validity job from inline emails
encrata jobs bulk_validate_emails --emails a@example.com --emails b@example.com

# Identity job from a file
encrata jobs bulk_email_identity --file emails.csv --file-name "prospects"

# Password breach job from SHA-1 hashes (hashes only — never plaintext)
encrata jobs bulk_password_breaches --sha1s 5BAA61E4C9B93F3F0682250B6CF8331B7EE68FD8
encrata jobs bulk_password_breaches --sha1-file hashes.txt
```

Options:

| Command | Flag | Description |
| ------- | ---- | ----------- |
| `bulk_validate_emails` | `--emails` | Inline email addresses (repeatable) |
| `bulk_validate_emails` | `--file` | Read emails from a file |
| `bulk_validate_emails` | `--file-name` | Optional display name for the job |
| `bulk_validate_emails` | `--batch-id` | Optional grouping ID across related jobs |
| `bulk_email_identity` | `--emails` / `--file` / `--file-name` | Same inputs as above |
| `bulk_password_breaches` | `--sha1s` | Inline SHA-1 hashes (40-char hex, repeatable) |
| `bulk_password_breaches` | `--sha1-file` | Read SHA-1 hashes from a file |
| `bulk_password_breaches` | `--file-name` | Optional display name for the job |

Only UPPER-CASE hex SHA-1 hashes are transmitted for password jobs — plaintext
passwords are never sent.

#### Track and manage any job type

These commands accept `--job-type validity|identity|password` (default
`validity`) and operate on a job ID.

```bash
encrata jobs get_email_job_status  JOB_ID --job-type identity
encrata jobs get_email_job_results JOB_ID --job-type password --page 1 --page-size 100 --breached
encrata jobs download_email_job    JOB_ID --job-type identity --found-only --out out.csv
encrata jobs cancel_email_job      JOB_ID --job-type validity
encrata jobs retry_email_job       JOB_ID --job-type validity
```

Options:

| Command | Flag | Description |
| ------- | ---- | ----------- |
| `get_email_job_results` | `--page` | 1-based page number |
| `get_email_job_results` | `--page-size` | Results per page |
| `get_email_job_results` | `--breached` | Password: breached only; identity: found only |
| `get_email_job_results` | `--found-only` | Identity: found rows only |
| `download_email_job` | `--format` | Download format: `csv` |
| `download_email_job` | `--status` | Validity-only per-row status filter |
| `download_email_job` | `--breached` | Password: breached only; identity: found only |
| `download_email_job` | `--found-only` | Identity: found rows only |
| `download_email_job` | `--out` | Write to a file instead of stdout |

#### Bulk-job registry

Inspect and manage the underlying bulk-job records.

```bash
encrata jobs list_bulk_jobs
encrata jobs get_bulk_job JOB_ID
encrata jobs cancel_bulk_job JOB_ID
```

---

### `encrata lists`

Manage reusable contact lists — named collections of emails you can share across
enrichment and monitoring workflows.

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

Create, list, and revoke API keys.

```bash
encrata keys ls
encrata keys create "CI key"
encrata keys revoke KEY_ID
encrata keys revoke KEY_ID --permanent
```

---

## Credits

Billing is 1 credit per successful unique email. Duplicate emails are
de-duplicated, and invalid or failed checks are not charged.

| Command | Credits |
| ------- | ------- |
| `email validity` | 1 credit per successful email |
| `email enrich` | 1 credit per successful email |
| `email identity` | 1 credit per successful email |
| `email breaches` | 1 credit per successful email |
| `email verify` | 1 credit per successful email |
| `email bulk` | 1 credit per successful unique email |
| `jobs` | 1 credit per successful unique email |
| `lists` | Free — list management does not consume credits |

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
