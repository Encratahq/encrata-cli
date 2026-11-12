# Changelog

All notable changes to the Encrata CLI are documented here.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.14.1] - 2026-11-12

### Fixed

- `encrata malware` scans now print `Credits used` for each lookup (ENC-2755).
- `encrata malware qr` / `barcode` now list the warnings raised on each decoded
  link (e.g. "URL uses a raw IP address"), matching the dashboard (ENC-2756).

### Changed

- `encrata workflows` (`wf`) is hidden from help, the command list and shell
  completion, and removed from the README. It still runs if invoked directly.

## [0.14.0] - 2026-11-03

### Added
- `malware url|file|image|qr|barcode` - single malware scans returning a verdict
  (`clean`, `suspicious`, `malicious`, `unknown`), a 0-100 risk score, the
  signals behind it, decoded codes (qr/barcode) and indicators of compromise.
  `malware file` uploads up to 10MB inline, or scans a larger file (up to 100MB)
  by download link with `--source-url`. `--stix` attaches a STIX 2.1 bundle and
  `--fail-on-finding` exits 2 on a suspicious or malicious verdict. Targets
  `POST /api/cli/malware/scan`.
- `jobs watch [--job <id>] [--type <kind>]` - wait for bulk jobs to finish over
  the live completion stream (`GET /api/cli/subscribe`) instead of polling. With
  `--job` it exits 0 when the job completed and 1 when it failed or was
  cancelled; without it, it follows every job until Ctrl+C (`--json` prints one
  object per job). The stream reconnects with backoff and de-duplicates
  replayed events.
- `email breaches --bulk` (job mode) now also listens on the completion stream,
  so the wait ends as soon as the server announces the job finished; the status
  poll remains for the progress bar and as the fallback.
- Desktop notifications when a bulk job finishes: Encrata-branded, with a
  **View results** button that opens the job on the dashboard (Windows; the link
  is in the text on macOS/Linux). Off by default and local to the machine:
  `config notifications on|off|test`, `--on completed,failed,cancelled`, and
  per-run `--notify` / `--no-notify`. Jobs under 30s in the foreground don't
  notify unless `--notify` is passed, and bursts while following `jobs watch`
  are grouped into one notification.
- `--job` and `--stream` on `email breaches --bulk` to force an async job or a
  live stream (mutually exclusive, like `email bulk`). Ctrl+C while waiting on
  the job stops watching without cancelling it and prints the job ID.
- `jobs ... --type breaches` - `create`, `list`, `status`, `results`,
  `download`, `cancel` and `retry` for bulk email-breaches jobs.
- Insufficient-credits (402) failures now follow the server's message with how
  to check your balance and top up.

### Changed
- Breaches jobs use the jobs service's current routes: create goes through
  `POST /api/cli/jobs/bulk` (`{"type":"breaches", ...}`), and results, download,
  cancel and retry address the job by id (`/api/cli/jobs/{id}[/download|/cancel|/retry]`).
  The previous per-kind `/api/cli/jobs/breaches/...` routes were removed
  server-side and returned 405/404. Results are keyset-paged only, so
  `jobs results --type breaches --page N` walks the cursor to page N.

### Fixed
- `email breaches <file> --bulk` no longer dies part-way through large lists with
  `stream error: stream ID 1; INTERNAL_ERROR; received from peer`. The whole list
  used to go out as one streaming request, which the server caps at 120s; lists
  over 200 emails now run as an async breaches job (create, live progress while
  polling, then the rows are downloaded into the same result file, columns and
  summary as a live run). Lists of 200 or fewer still stream live.
- A bulk stream that is cut off (`email breaches --bulk --stream`,
  `email bulk --stream`) no longer discards the rows that already arrived: they
  are saved to the result file (`Saved 455 of 8,569 results to ... (partial).`),
  or included in the `--json` output, and the raw transport error is replaced
  with how long the run lasted, how far it got, and the `--job` command to re-run
  it.

## [0.13.0] - 2026-09-18

### Added
- `email compliance <email>` - resolve whether you may cold-email an address and
  under whose law, returning the governing jurisdiction, its rule, main laws and
  restrictions, and attribution confidence. Targets `/api/cli/email/compliance`.
- `breaches github scan <repo>` and `breaches github get <scan-id>` - scan a
  GitHub repository for committed secrets and fetch a prior scan by ID. `--deep`
  walks the full commit history in the background; `--ref`, `--min-severity`,
  `--fail-on-finding` and `--out` are supported. Targets
  `/api/cli/breaches/github[/{id}]`.

## [0.12.1] - 2026-09-02

### Changed
- Migrated all CLI API calls to the restructured nested `/api/cli/*` route tree.
  Email verbs now target `/api/cli/email/*` (`email/validity`, `email/identity`,
  `email/breaches`, `email/validity/bulk`, `email/bulk-validity-search`,
  `email/bulk-breaches-search`), password breach lookups target
  `/api/cli/email/password/breaches[/bulk]`, and every async job/export route
  moved under `/api/cli/jobs/*` (`jobs/validity`, `jobs/identity`,
  `jobs/password`, `jobs/bulk`, `jobs/exports`, plus their
  `results`/`download`/`cancel`/`retry` sub-paths). No user-facing command or
  flag changes - the surface is identical, only the backend endpoints moved.

## [0.12.0] - 2026-08-26

### Removed
- `email verify` and `email enrich` commands (and their `/api/cli/email-verify`
  and `/api/cli/email-enrich` calls) - these are no longer backed by the API.
  Use `email validity` for the deliverability verdict and full report.

### Changed
- Unified row filtering on a single `--only valid|invalid|found|breached` flag
  across `email bulk`, `email breaches`, and `jobs results` / `jobs download`.
  The old `--valid-only` / `--found-only` / `--breached` flags still work as
  hidden aliases.
- Clarified `email bulk` help: `--job` runs and waits like the `jobs` command,
  and `--enrich` fills every export column (same 1 credit/email, just slower).
- Example addresses in help now use `jane@acme.com` instead of
  `user@example.com`.

### Added
- `email bulk` prints a notice when a batch auto-switches to async job mode
  (>1000 emails), and a hint that lean exports only fill `email,status,reason`
  unless `--enrich` is set.

## [0.11.2] - 2026-08-17

### Changed
- Maintenance release; no user-facing command changes.

## [0.11.1] - 2026-08-17

### Fixed
- Windows `encrata update` no longer leaves the old binary behind: the npm
  update now runs in a detached process that waits for the CLI to exit, then
  installs the new version and removes the leftover `.encrata-cli-*` staging
  folders (fixes the `EPERM ... unlink ... encrata-bin.exe` cleanup warnings).

## [0.11.0] - 2026-08-17

### Changed
- Default API base URL is now `https://cli.encrata.com` (was `https://api.encrata.com`).
- Workflows commands (`encrata workflows` / `wf` and their subcommands) are
  hidden from help and command listings. They remain functional if invoked
  directly, but are no longer part of the documented surface.

## [0.10.0] - 2026-08-14

### Added
- Routing path migration hardening across CLI command groups and related docs.

### Changed
- Version bump for the latest CLI release package line.

## [0.9.1] - 2026-08-12

### Changed
- Maintenance release; no user-facing command changes.

## [0.9.0] - 2026-08-10

### Added
- Bulk email jobs — submit, poll, summarise and download large enrichment runs
  for validity, identity and breach modes.

### Changed
- Internal reorganisation: large command and API files split into focused units
  (jobs, keys, lists, webhooks, workflows, workspace, render/export helpers) for
  maintainability. No user-facing command surface removed.

## [0.8.0] - 2026-07-31

### Added
- **Workflows (bulk enrichment)** — `encrata workflows` (alias `wf`):
  - `upload <file>` — upload a CSV/TXT/XLSX of emails (`POST /api/workflows/files`,
    multipart field `file`); prints `{id, filename, row_count, identifier_type,
    identifier_column}`.
  - `run <workflow-id> --file <file-id>` — trigger a bulk run
    (`POST /api/workflows/{id}/run` with body `{file_id}`); supports
    `--idempotency-key` for safe retries.
  - `status <run-id>` — run status, credits and per-step breakdown
    (`GET /api/workflows/runs/{id}` → `{run, steps}`).
  - `runs` — list recent runs (`GET /api/workflows/runs`).
  - `cancel <run-id>` — request cooperative cancellation.
  - `download <run-id> --out <file>` — stream the enriched results CSV
    (follows the `302` to the presigned URL).
- **Integrations** — `encrata workflows integrations` (alias `int`):
  - `list`, `providers`, `session`, `save`, `create-sheet <id>`, `disconnect <id>`.
  - Tokens and connection secrets are never printed; `503` (Nango not configured)
    is surfaced as a clear, non-leaking message.
- `encrata --version` flag (in addition to `encrata version`), now printing
  version, commit and build date.

### Changed
- Version metadata moved to a single source of truth, `internal/version`
  (`Version`/`Commit`/`Date`), injected at build time via `-ldflags`. GoReleaser
  now stamps commit and date as well as the tag.

### Notes / contract reconciliation
- Validate export columns already match the backend: the verdict is read from
  `status` (with `validity` as the backward-compat alias) — there is no `verdict`
  field and no top-level `deliverable` boolean, so none was added. `disposable`,
  `role`/`role_name` and `reason` are included; transport failures
  (`ip_blocked`/`unreachable`/`greylist_unresolved`) surface via `reason`.
- Breaches use the canonical nested shape `breached` + `breach_info{breach_count,
  services[{name,domain,breach_date,data_types}], interests, exposed_data}`;
  `interests` are `{category, signal}` objects. The canonical `/api/email/breaches`
  endpoint has no flat `count`/string-`services` aliases.
- The connected-account spreadsheet route is `/api/workflows/integrations/{id}/sheet`
  (not `/create-sheet`) and returns `{spreadsheet_id, spreadsheet_url, sheet_name}`.

[Unreleased]: https://github.com/Encratahq/encrata-cli/compare/v0.14.1...HEAD
[0.14.1]: https://github.com/Encratahq/encrata-cli/releases/tag/v0.14.1
[0.14.0]: https://github.com/Encratahq/encrata-cli/releases/tag/v0.14.0
[0.13.0]: https://github.com/Encratahq/encrata-cli/releases/tag/v0.13.0
[0.12.1]: https://github.com/Encratahq/encrata-cli/releases/tag/v0.12.1
[0.12.0]: https://github.com/Encratahq/encrata-cli/releases/tag/v0.12.0
[0.11.2]: https://github.com/Encratahq/encrata-cli/releases/tag/v0.11.2
[0.11.1]: https://github.com/Encratahq/encrata-cli/releases/tag/v0.11.1
[0.11.0]: https://github.com/Encratahq/encrata-cli/releases/tag/v0.11.0
[0.10.0]: https://github.com/Encratahq/encrata-cli/releases/tag/v0.10.0
[0.9.1]: https://github.com/Encratahq/encrata-cli/releases/tag/v0.9.1
[0.9.0]: https://github.com/Encratahq/encrata-cli/releases/tag/v0.9.0
[0.8.0]: https://github.com/Encratahq/encrata-cli/releases/tag/v0.8.0
