# Encrata CLI

Intelligence lookups from your terminal. Email enrichment, phone intelligence, IP geolocation, domain research, company profiles, Google dorking, dark web search, web scraping, screenshots, and face recognition — all from a single command.

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
& ([scriptblock]::Create((irm https://raw.githubusercontent.com/Encratahq/cli/main/install.ps1))) -Version 0.3.1
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

Verify the install:

```bash
encrata version
```

### npm

Recommended for developers who already use Node.js/npm.

```bash
npm install -g encrata-cli
```

Verify the install:

```bash
encrata version
```

The npm package installs a small wrapper that downloads the matching Encrata
binary for your operating system from GitHub Releases.

### Manual Download

Download prebuilt binaries from [GitHub Releases](https://github.com/Encratahq/cli/releases).

Windows users should download the `windows_amd64.zip` or `windows_arm64.zip`
asset, extract it, and run `encrata.exe`.

## Setup

```bash
encrata config set-key YOUR_API_KEY
```

Get your API key at [encrata.com/settings/api-keys](https://encrata.com/settings/api-keys).

## Commands

### Email Lookup

Look up a person by email — name, company, role, socials, breaches, and more.

```bash
encrata email user@example.com
encrata email user@example.com --json
```

### Phone Lookup

Look up a phone number — carrier, format, country, validation, risk, and breach data.

```bash
encrata phone "+14155552671"
encrata phone "+447911123456" --json
```

### IP Lookup

Look up an IP address — geolocation, ASN, company, and threat detection.

```bash
encrata ip 8.8.8.8
encrata ip 2001:4860:4860::8888 --json
```

### Domain Search

Investigate a domain — WHOIS, DNS, SSL, threat intel, and search results.

```bash
encrata domain tesla.com
```

### Company Search

Find company profiles, employee emails, and knowledge graph data.

```bash
encrata company "OpenAI"
```

### Google Search

OSINT dorking — find exposed files, admin panels, and public info.

```bash
encrata google "site:example.com filetype:pdf"
```

### Dark Web Search

Search dark web intelligence — credential leaks, forum posts, market listings.

```bash
encrata darkweb "user@example.com" --type email
encrata darkweb "example.com" --type domain
```

### Scrape

Fetch the raw HTML of a web page — renders JavaScript and bypasses bot blocks.

```bash
encrata scrape https://example.com
encrata scrape https://example.com -o page.html
encrata scrape https://example.com --no-js --wait-for "#main"
```

### Extract

Extract clean markdown or structured data from a web page.

```bash
encrata extract https://example.com
encrata extract https://example.com --mode markdown
encrata extract https://example.com --selector title=h1 --selector price=.price
```

### Screenshot

Capture a full-page or element screenshot (PNG or JPEG).

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

## Options

| Flag | Description |
|------|-------------|
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

- `email` — 1 credit per lookup (cached within 24h free)
- `phone` — 1 credit per lookup
- `domain` — 1 credit
- `company` — 1 credit
- `google` — 1 credit
- `darkweb` — 1 credit (first page; subsequent pages free)
- `scrape` — 1 credit per page
- `extract` — 1 credit per page
- `screenshot` — 1 credit per capture
- `face` — 5 credits per search
- `ip` — free
- `validate` — free
- `breaches` — free

## Links

- [Documentation](https://docs.encrata.com/cli)
- [API Reference](https://docs.encrata.com/api-reference)
- [Dashboard](https://encrata.com)

## License

MIT
