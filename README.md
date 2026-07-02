# Encrata CLI

Intelligence lookups from your terminal. Encrata CLI brings email enrichment,
phone intelligence, IP geolocation, domain research, company profiles, Google
dorking, dark web search, web scraping, screenshots, face recognition, bulk
operations, monitors, workflows, webhooks, and API key management into one
command-line tool.

## Install

Choose the install method that fits your environment.

### Windows PowerShell

Recommended for Windows users. This does not require Node.js or npm.

```powershell
irm https://raw.githubusercontent.com/Encratahq/cli/main/install.ps1 | iex
```

The installer downloads the latest Windows release from GitHub, extracts
`encrata.exe` to `%LOCALAPPDATA%\Programs\Encrata`, and adds that directory to
your user `PATH`.

Open a new PowerShell window after installation, then verify:

```powershell
encrata version
```

Install a specific version:

```powershell
& ([scriptblock]::Create((irm https://raw.githubusercontent.com/Encratahq/cli/main/install.ps1))) -Version 0.4.0
```

Update to the latest version by running the same install command again.

### Homebrew

Recommended for macOS users.

```bash
brew tap Encratahq/tap
brew install encrata
```

Or install directly:

```bash
brew install Encratahq/tap/encrata
```

### npm

Recommended for developers who already use Node.js/npm.

```bash
npm install -g encrata-cli
```

The npm package installs a small wrapper that downloads the matching Encrata
binary for your operating system from GitHub Releases.

### Manual Download

Download prebuilt binaries from
[GitHub Releases](https://github.com/Encratahq/cli/releases).

Windows users should download the `windows_amd64.zip` or `windows_arm64.zip`
asset, extract it, and run `encrata.exe`.

## Setup

```bash
encrata config set-key YOUR_API_KEY
```

Get your API key at
[encrata.com/settings/api-keys](https://encrata.com/settings/api-keys).

Verify your install:

```bash
encrata version
encrata ip 8.8.8.8
```

## Commands

### Email Lookup

Look up a person by email: name, company, role, socials, breaches, and more.

```bash
encrata email user@example.com
encrata email user@example.com --json
encrata email user@example.com --fields name,company,social_profiles --nocache
```

### Phone Lookup

Look up a phone number: carrier, format, country, validation, risk, and breach
data.

```bash
encrata phone "+14155552671"
encrata phone "+447911123456" --json
```

### IP Lookup

Look up an IP address: geolocation, ASN, company, and threat detection.

```bash
encrata ip 8.8.8.8
encrata ip 2001:4860:4860::8888 --json
```

### Domain Search

Investigate a domain: WHOIS, DNS, SSL, threat intelligence, and search results.

```bash
encrata domain tesla.com
```

### Company Search

Find company profiles, employee emails, and knowledge graph data.

```bash
encrata company "OpenAI"
```

### Google Search

Run OSINT dorking queries to find exposed files, admin panels, and public info.

```bash
encrata google "site:example.com filetype:pdf"
```

### Dark Web Search

Search dark web intelligence: credential leaks, forum posts, and market
listings.

```bash
encrata darkweb "user@example.com" --type email
encrata darkweb "example.com" --type domain
```

### Validate and Breaches

Run free email validation and breach checks.

```bash
encrata validate user@example.com
encrata breaches user@example.com
```

### Scrape

Fetch the raw HTML of a web page. JavaScript rendering is enabled by default.

```bash
encrata scrape https://example.com
encrata scrape https://example.com -o page.html
encrata scrape https://example.com --no-js --wait-for "#main"
```

### Extract

Extract clean markdown, text, or structured data from a web page.

```bash
encrata extract https://example.com
encrata extract https://example.com --mode markdown
encrata extract https://example.com --selector title=h1 --selector price=.price
```

### Screenshot

Capture a full-page, viewport, or element screenshot as PNG or JPEG.

```bash
encrata screenshot https://example.com
encrata screenshot https://example.com -o shot.jpeg --format jpeg
encrata screenshot https://example.com --viewport --selector "#hero"
```

### Face Search

Find matching faces and linked identities from an image URL.

```bash
encrata face https://example.com/photo.jpg
encrata face https://example.com/photo.jpg --threshold 0.8
```

### Bulk Operations

Run batch enrichment and search jobs from arguments or a file.

```bash
encrata bulk lookup user@example.com admin@example.com
encrata bulk lookup --file emails.txt --fields name,company
encrata bulk google --file queries.txt
encrata bulk company "OpenAI" "Stripe"
encrata bulk domain example.com encrata.com
encrata bulk ip 8.8.8.8 1.1.1.1
```

### Contact Lists

Manage reusable target lists for enrichment and monitoring.

```bash
encrata lists ls
encrata lists create "Prospects" --type email --targets user@example.com
encrata lists add LIST_ID user@example.com admin@example.com
encrata lists emails LIST_ID
encrata lists remove LIST_ID user@example.com
encrata lists rm LIST_ID
```

### Monitors

Create monitors, trigger runs, and inspect results.

```bash
encrata monitors ls
encrata monitors create "VIP contacts" --emails user@example.com --frequency monthly
encrata monitors create "List monitor" --list-id LIST_ID --frequency weekly
encrata monitors run MONITOR_ID
encrata monitors runs MONITOR_ID
encrata monitors results MONITOR_ID RUN_ID --changes-only
encrata monitors all-runs
```

### Workflows

Manage automation workflows, runs, templates, and secrets.

```bash
encrata workflows ls
encrata workflows templates
encrata workflows create "Daily enrichment" --template-id TEMPLATE_ID
encrata workflows create "Custom workflow" --file workflow.json
encrata workflows update WORKFLOW_ID --status active
encrata workflows runs --workflow-id WORKFLOW_ID
encrata workflows run RUN_ID
encrata workflows secrets set API_TOKEN secret-value
encrata workflows secrets ls
```

### Webhooks

Register webhooks and inspect deliveries.

```bash
encrata webhooks ls
encrata webhooks create https://example.com/hook --events workflow.completed
encrata webhooks update WEBHOOK_ID https://example.com/new-hook --active=false
encrata webhooks test WEBHOOK_ID
encrata webhooks deliveries WEBHOOK_ID
encrata webhooks rm WEBHOOK_ID
```

### API Keys

Create, list, and revoke API keys.

```bash
encrata keys ls
encrata keys create "CI key"
encrata keys revoke KEY_ID
encrata keys revoke KEY_ID --permanent
```

## Global Options

| Flag | Description |
| ---- | ----------- |
| `--json` | Output raw JSON |
| `--api-key` | Override API key for this request |
| `--base-url` | Override API base URL |

## Configuration

```bash
encrata config set-key <key>    # Save API key
encrata config show             # Show current config
encrata config set-url <url>    # Set custom base URL
```

Config is stored in `~/.config/encrata/config.json`.

## Credits

| Command | Credits |
| ------- | ------- |
| `email` | 1 credit per lookup; cached within 24 hours free |
| `phone` | 1 credit per lookup |
| `domain` | 1 credit |
| `company` | 1 credit |
| `google` | 1 credit |
| `darkweb` | 1 credit for the first page; subsequent pages free |
| `scrape` | 1 credit per page |
| `extract` | 1 credit per page |
| `screenshot` | 1 credit per capture |
| `face` | 5 credits per search |
| `bulk` | Credits depend on the operation and number of inputs |
| `ip` | Free |
| `validate` | Free |
| `breaches` | Free |

## Links

- [Documentation](https://docs.encrata.com/cli)
- [API Reference](https://docs.encrata.com/api-reference)
- [Dashboard](https://encrata.com)
- [GitHub Releases](https://github.com/Encratahq/cli/releases)

## License

MIT
