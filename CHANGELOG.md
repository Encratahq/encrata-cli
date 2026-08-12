# Changelog

All notable changes to the Encrata CLI are documented here.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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

[Unreleased]: https://github.com/Encratahq/encrata-cli/compare/v0.9.0...HEAD
[0.9.0]: https://github.com/Encratahq/encrata-cli/releases/tag/v0.9.0
[0.8.0]: https://github.com/Encratahq/encrata-cli/releases/tag/v0.8.0
