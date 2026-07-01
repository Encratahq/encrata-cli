# Distributing the Encrata CLI to Windows / PowerShell

You already ship to macOS + Linux via **Homebrew** (auto-published by GoReleaser to
`Encratahq/homebrew-tap`). This plan covers the Windows equivalent.

---

## TL;DR — what "Homebrew for PowerShell" actually is

There is no single answer. Windows has **three** common install channels, roughly in
order of how close they feel to Homebrew:

| Channel | Homebrew analog | Install command | Effort |
|---|---|---|---|
| **Scoop** | Closest match (community package manager, bucket repo) | `scoop install encrata` | Low — GoReleaser generates it |
| **winget** | Microsoft's official manager (preinstalled on Win 11) | `winget install Encrata.CLI` | Medium — manifests + review |
| **PowerShell script** | Like `curl … \| bash` | `irm https://encrata.com/install.ps1 \| iex` | Low — one script |

Recommended: **do Scoop first** (it mirrors your Homebrew flow almost 1:1 and
GoReleaser automates it), then add a **PowerShell install script** for a no-package-
manager option, and optionally **winget** later for reach.

---

## Prerequisite (blocks everything): build Windows binaries

Right now `.goreleaser.yaml` only builds `darwin` + `linux`. Windows users can't be
served until GoReleaser produces a `.exe` in a `.zip`.

### 1. Add Windows to the build matrix

In `.goreleaser.yaml` → `builds:`

```yaml
    goos:
      - darwin
      - linux
      - windows        # add
    goarch:
      - amd64
      - arm64
```

### 2. Ship Windows archives as `.zip`

Homebrew/Linux use `.tar.gz`; Windows tooling (Scoop, winget) expects `.zip`.
Update `archives:` to override the format for Windows:

```yaml
archives:
  - id: default
    name_template: "{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}"
    formats: [tar.gz]
    format_overrides:
      - goos: windows
        formats: [zip]
    files:
      - README.md
```

This yields `encrata_<version>_windows_amd64.zip` / `_arm64.zip` — exactly the asset
names your `npm/encrata/install.js` already looks for. So this step **also fixes
`npm i -g encrata-cli` on Windows** as a bonus.

> After this change, cut a test tag (e.g. `v0.4.0-rc1`) and confirm the Windows
> `.zip` assets appear on the GitHub Release before wiring up any package manager.

---

## Option A — Scoop (recommended, closest to Homebrew)

### How it differs from Homebrew
- Homebrew formula = a Ruby file in a **tap** repo. Scoop manifest = a **JSON** file in
  a **bucket** repo.
- Homebrew builds/installs; Scoop just downloads the prebuilt `.zip`, extracts the
  `.exe`, and adds it to PATH via shims.
- No Xcode/compiler on the user's machine — pure download + unzip.

### Steps
1. **Create a bucket repo**: `Encratahq/scoop-bucket` (public, empty).
2. **Add a `scoops:` block** to `.goreleaser.yaml` (parallel to your `brews:` block):

```yaml
scoops:
  - name: encrata
    repository:
      owner: Encratahq
      name: scoop-bucket
      branch: main
      token: "{{ .Env.SCOOP_BUCKET_GITHUB_TOKEN }}"
    homepage: "https://encrata.com"
    description: "Intelligence lookups from your terminal — email, phone, IP, domain, and OSINT"
    license: MIT
    commit_author:
      name: encrata-bot
      email: hello@encrata.com
```

3. **Add the token secret** `SCOOP_BUCKET_GITHUB_TOKEN` (PAT with write access to
   `Encratahq/scoop-bucket`) in: cli repo → Settings → Secrets and variables → Actions.
   Then reference it in `release.yml` `env:` next to `HOMEBREW_TAP_GITHUB_TOKEN`.
4. **Release** by pushing a tag. GoReleaser fills in version, URLs, and sha256 and
   pushes `encrata.json` to the bucket automatically.

### User install
```powershell
scoop bucket add encrata https://github.com/Encratahq/scoop-bucket
scoop install encrata
scoop update encrata
```

---

## Option B — PowerShell install script (no package manager)

Mirrors your `curl … | bash` story. Good default for docs / quickstart.

### Steps
1. Author `install.ps1` that:
   - Detects arch (`$env:PROCESSOR_ARCHITECTURE` → `amd64`/`arm64`).
   - Resolves the latest release tag from the GitHub API (or accepts `-Version`).
   - Downloads `encrata_<version>_windows_<arch>.zip`.
   - Extracts `encrata.exe` to `"$env:LOCALAPPDATA\Programs\Encrata"`.
   - Adds that dir to the **user** PATH (`[Environment]::SetEnvironmentVariable`).
2. Host it at a stable URL (e.g. `https://encrata.com/install.ps1`, or raw GitHub).

### User install
```powershell
irm https://encrata.com/install.ps1 | iex
```

Notes / gotchas:
- Users may need `Set-ExecutionPolicy -Scope Process RemoteSigned` first.
- PATH changes require a new terminal session to take effect.
- Sign the binary (Authenticode) later to reduce SmartScreen warnings.

---

## Option C — winget (official, widest reach, most process)

### How it differs from Homebrew
- Manifests are **YAML** submitted to the central `microsoft/winget-pkgs` repo and go
  through **automated + human review** (slower, not instant like a tap push).
- Installed by default on Windows 11 → best discoverability.

### Steps
1. Add a `winget:` block to `.goreleaser.yaml`:

```yaml
winget:
  - name: encrata
    publisher: Encrata
    short_description: "Intelligence lookups from your terminal"
    license: MIT
    homepage: "https://encrata.com"
    repository:
      owner: Encratahq
      name: winget-pkgs           # your fork of microsoft/winget-pkgs
      branch: "encrata-{{ .Version }}"
      token: "{{ .Env.WINGET_GITHUB_TOKEN }}"
      pull_request:
        enabled: true
        base:
          owner: microsoft
          name: winget-pkgs
          branch: master
```

2. Maintain a fork `Encratahq/winget-pkgs`; GoReleaser opens a PR upstream each release.
3. First submission requires reserving the package identifier `Encrata.CLI`.

### User install
```powershell
winget install Encrata.CLI
```

---

## Chocolatey (optional, enterprise-heavy audiences)

GoReleaser also supports `chocolateys:`. Skip unless customers specifically ask —
Scoop + winget cover the modern Windows story.

---

## Recommended rollout order

1. **[Blocking]** Add `windows` + `.zip` overrides to `.goreleaser.yaml`; verify assets
   on an RC release. (Also unbreaks npm on Windows.)
2. **Scoop bucket** (`Encratahq/scoop-bucket`) + `scoops:` block + token. Fastest, most
   Homebrew-like win.
3. **`install.ps1`** one-liner for docs/quickstart.
4. **winget** once naming + signing are sorted, for reach.

## Docs to update after shipping
- `cli/README.md` — add a Windows/PowerShell "Install" section.
- `encrata/docs/cli.mdx` and `encrata/docs/quickstart.mdx` — add Scoop / `irm | iex`.
- Homebrew tap README already exists; add a scoop-bucket README mirroring it.

## Key differences vs Homebrew (summary)
- **Archive format**: Windows = `.zip`, not `.tar.gz`.
- **Binary**: `encrata.exe`, not `encrata`.
- **Manifest**: Scoop = JSON bucket, winget = YAML PR to Microsoft, vs Homebrew Ruby tap.
- **No build on user machine**: all Windows channels download the prebuilt `.exe`.
- **PATH**: handled by Scoop shims / your script / winget, not symlinks in `/usr/local/bin`.
- **Code signing/SmartScreen**: a Windows-only concern with no Homebrew equivalent.
