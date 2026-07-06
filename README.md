# Encrata CLI

The official CLI for Encrata.

Built for intelligence lookups, OSINT workflows, automation scripts, and local
developer testing.

---

## Install

### PowerShell (Windows)

```powershell
irm https://raw.githubusercontent.com/Encratahq/encrata-cli/main/install.ps1 | iex
```

Install a specific version:

```powershell
& ([scriptblock]::Create((irm https://raw.githubusercontent.com/Encratahq/encrata-cli/main/install.ps1))) -Version 0.4.6
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
encrata ip 8.8.8.8

# Get the full API response
encrata company tesla --json
```

---

## Authentication

The CLI resolves your API key using this priority chain:

| Priority | Source | How to set |
| -------- | ------ | ---------- |
| 1 | `--api-key` flag | `encrata ip 8.8.8.8 --api-key YOUR_API_KEY` |
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
go run . ip 8.8.8.8
```

4. Point local runs at a local backend.

```powershell
$env:ENCRATA_BASE_URL = "http://localhost:8080"
go run . domain google.com
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

### `encrata email`

Look up a person or identity by email address. The response can include profile
data, company data, social links, breach signals, and related metadata.

```bash
encrata email user@example.com
encrata email user@example.com --fields name,company,social_profiles
encrata email user@example.com --nocache
encrata email user@example.com --json
```

Options:

| Flag | Description |
| ---- | ----------- |
| `--fields` | Limit the response to specific fields |
| `--nocache` | Bypass cache and run a fresh lookup |
| `--country` | Country code hint |
| `--lang` | Language code hint |

---

### `encrata phone`

Look up a phone number. The response can include carrier, country, normalized
format, validity, risk, and related intelligence.

```bash
encrata phone "+14155552671"
encrata phone "+447911123456" --json
```

---

### `encrata ip`

Look up an IP address. The response can include location, ASN, network owner,
company, ISP, and threat signals.

```bash
encrata ip 8.8.8.8
encrata ip 2001:4860:4860::8888 --json
```

---

### `encrata domain`

Investigate a domain. The response can include WHOIS, DNS, SSL, risk,
technology, popularity, URL scan, and related summary data.

```bash
encrata domain google.com
encrata domain google.com --json
```

---

### `encrata company`

Search for company intelligence. Table output summarizes profile data,
knowledge graph data, top search results, and SEC filings when available.

```bash
encrata company tesla
encrata company tesla --json
```

---

### `encrata google`

Run Google OSINT searches or dorking queries.

```bash
encrata google "site:example.com filetype:pdf"
encrata google "intitle:index.of password"
encrata google "open source intelligence" --json
```

---

### `encrata darkweb`

Search dark web intelligence for emails, domains, keywords, breach records, and
onion search results. Enriched results can include LeakCheck breach data and
darkdump onion search hits.

```bash
encrata darkweb user@example.com
encrata darkweb example.com
encrata darkweb "company name" --offset 10
encrata darkweb user@example.com --json
```

Options:

| Flag | Description |
| ---- | ----------- |
| `--offset` | Pagination offset |

---

### `encrata darkweb crawl`

Crawl a `.onion` URL through the dark web crawl endpoint. Use it to check
whether an onion page is live, collect linked onion URLs, and extract page-level
emails, phone numbers, titles, and status codes.

```bash
encrata darkweb crawl http://exampleonionaddress.onion
encrata darkweb crawl http://exampleonionaddress.onion --depth 2
encrata darkweb crawl http://exampleonionaddress.onion --depth 3 --force
encrata darkweb crawl http://exampleonionaddress.onion --json
```

Options:

| Flag | Description |
| ---- | ----------- |
| `--depth` | Crawl depth from 1 to 3. Default is 1 |
| `--force` | Bypass cache and run a fresh crawl |

---

### `encrata validate`

Validate an email address.

```bash
encrata validate user@example.com
encrata validate user@example.com --json
```

---

### `encrata breaches`

Check whether an email appears in known breach data.

```bash
encrata breaches user@example.com
encrata breaches user@example.com --json
```

---

### `encrata scrape`

Fetch raw HTML from a web page. JavaScript rendering is enabled by default.

```bash
encrata scrape https://example.com
encrata scrape https://example.com -o page.html
encrata scrape https://example.com --no-js
encrata scrape https://example.com --wait-for "#main"
```

Options:

| Flag | Description |
| ---- | ----------- |
| `-o, --output-file` | Write HTML to a file |
| `--no-js` | Disable JavaScript rendering |
| `--wait-for` | Wait for a CSS selector |
| `--timeout` | Timeout in milliseconds |

---

### `encrata extract`

Extract clean page content as markdown, text, or structured selector fields.

```bash
encrata extract https://example.com
encrata extract https://example.com --mode markdown
encrata extract https://example.com --mode text
encrata extract https://example.com --selector title=h1 --selector price=.price
```

Options:

| Flag | Description |
| ---- | ----------- |
| `--mode` | `markdown`, `text`, or `selectors` |
| `--selector` | Field selector as `name=css`; repeatable |
| `--no-js` | Disable JavaScript rendering |
| `--timeout` | Timeout in milliseconds |

---

### `encrata screenshot`

Capture a page, viewport, or selected element as PNG or JPEG.

```bash
encrata screenshot https://example.com
encrata screenshot https://example.com -o shot.jpeg --format jpeg
encrata screenshot https://example.com --viewport
encrata screenshot https://example.com --selector "#hero"
```

Options:

| Flag | Description |
| ---- | ----------- |
| `-o, --output-file` | Output file path |
| `--format` | `png` or `jpeg` |
| `--viewport` | Capture only the viewport |
| `--selector` | Capture one CSS selector |
| `--timeout` | Timeout in milliseconds |

---

### `encrata face`

Search for matching faces and linked identities from an image URL.

```bash
encrata face https://example.com/photo.jpg
encrata face https://example.com/photo.jpg --threshold 0.8
encrata face https://example.com/photo.jpg --json
```

Options:

| Flag | Description |
| ---- | ----------- |
| `--threshold` | Match confidence threshold from 0 to 1 |

---

### `encrata bulk lookup`

Run synchronous bulk email enrichment. Use this for smaller batches where you
want streamed results in the terminal.

```bash
encrata bulk lookup user@example.com admin@example.com
encrata bulk lookup --file emails.txt
encrata bulk lookup --file emails.txt --fields name,company
encrata bulk lookup --file emails.txt --json
```

Options:

| Flag | Description |
| ---- | ----------- |
| `-f, --file` | Read emails from a file |
| `--fields` | Limit returned fields |

---

### `encrata bulk google`

Run multiple Google OSINT searches.

```bash
encrata bulk google "open source intelligence" "tesla"
encrata bulk google --file queries.txt
encrata bulk google --file queries.txt --json
```

---

### `encrata bulk company`

Run multiple company lookups.

```bash
encrata bulk company tesla openai stripe
encrata bulk company --file companies.txt
encrata bulk company --file companies.txt --json
```

---

### `encrata bulk domain`

Run multiple domain lookups.

```bash
encrata bulk domain example.com encrata.com
encrata bulk domain --file domains.txt
encrata bulk domain --file domains.txt --json
```

---

### `encrata bulk ip`

Run multiple IP lookups.

```bash
encrata bulk ip 8.8.8.8 1.1.1.1
encrata bulk ip --file ips.txt
encrata bulk ip --file ips.txt --json
```

---

### `encrata jobs`

Manage asynchronous bulk email jobs. Use this for larger files where the backend
should process the batch in the background and return a downloadable result.

```bash
encrata jobs create --file emails.txt
encrata jobs list
encrata jobs get JOB_ID
encrata jobs cancel JOB_ID
```

---

### `encrata lists`

Manage reusable contact lists for enrichment and monitoring.

```bash
encrata lists ls
encrata lists create "Prospects" --type email --targets user@example.com
encrata lists get LIST_ID
encrata lists emails LIST_ID
encrata lists add LIST_ID user@example.com admin@example.com
encrata lists remove LIST_ID user@example.com
encrata lists rm LIST_ID
```

---

### `encrata monitors`

Create monitors, trigger runs, and inspect monitor results.

```bash
encrata monitors ls
encrata monitors create "VIP contacts" --emails user@example.com --frequency monthly
encrata monitors create "List monitor" --list-id LIST_ID --frequency weekly
encrata monitors get MONITOR_ID
encrata monitors run MONITOR_ID
encrata monitors runs MONITOR_ID
encrata monitors results MONITOR_ID RUN_ID
encrata monitors results MONITOR_ID RUN_ID --changes-only
encrata monitors all-runs
encrata monitors all-results
```

---

### `encrata workflows`

Manage automation workflows, templates, runs, and workflow secrets.

```bash
encrata workflows ls
encrata workflows templates
encrata workflows create "Daily enrichment" --template-id TEMPLATE_ID
encrata workflows create "Custom workflow" --file workflow.json
encrata workflows get WORKFLOW_ID
encrata workflows update WORKFLOW_ID --status active
encrata workflows runs --workflow-id WORKFLOW_ID
encrata workflows run RUN_ID
encrata workflows secrets ls
encrata workflows secrets set API_TOKEN secret-value
encrata workflows secrets rm API_TOKEN
```

---

### `encrata webhooks`

Register webhooks and inspect delivery attempts.

```bash
encrata webhooks ls
encrata webhooks create https://example.com/hook --events workflow.completed
encrata webhooks update WEBHOOK_ID https://example.com/new-hook --active=false
encrata webhooks test WEBHOOK_ID
encrata webhooks deliveries WEBHOOK_ID
encrata webhooks rm WEBHOOK_ID
```

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

Credit usage depends on the command and whether the backend can serve a cached
result.

| Command | Credits |
| ------- | ------- |
| `email` | 1 credit per lookup; cached results may be free |
| `phone` | 1 credit per lookup |
| `domain` | 1 credit |
| `company` | 1 credit |
| `google` | 1 credit |
| `darkweb` | 1 credit for the first page; pagination may be free |
| `darkweb crawl` | Depends on crawl depth and cache status |
| `scrape` | 1 credit per page |
| `extract` | 1 credit per page |
| `screenshot` | Credits depend on capture settings |
| `face` | 5 credits per search |
| `bulk` | Depends on operation and input count |
| `jobs` | Depends on email count |
| `ip` | Free |
| `validate` | Free |
| `breaches` | Free |

---

## Links

- Documentation: https://docs.encrata.com/cli
- API Reference: https://docs.encrata.com/api-reference
- Dashboard: https://encrata.com
- Releases: https://github.com/Encratahq/encrata-cli/releases

---

## License

MIT
