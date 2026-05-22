# Encrata CLI

Intelligence lookups from your terminal. Email enrichment, phone intelligence, IP geolocation, domain research, company profiles, Google dorking, and dark web search — all from a single command.

## Install

```bash
npm install -g encrata-cli
```

Or download the binary from [Releases](https://github.com/Encratahq/cli/releases).

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
- `ip` — free
- `validate` — free
- `breaches` — free

## Links

- [Documentation](https://docs.encrata.com/cli)
- [API Reference](https://docs.encrata.com/api-reference)
- [Dashboard](https://encrata.com)

## License

MIT
