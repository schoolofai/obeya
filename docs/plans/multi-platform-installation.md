# Multi-Platform Installation Plan

**Goal:** Make `ob` easy to install on Linux and Windows, matching the Homebrew experience on macOS.

## Current State

- GoReleaser already builds binaries for **linux/amd64**, **linux/arm64**, **windows/amd64**
- Archives are published to GitHub Releases (tar.gz for Linux, zip for Windows)
- Only macOS has a managed install channel (Homebrew via `schoolofai/homebrew-tap`)
- `go install` works everywhere but requires Go toolchain

## Plan

### Phase 1: Universal Install Script (Linux + macOS)

**What:** A `curl | sh` install script that detects OS/arch, downloads the right binary from GitHub Releases, and places it in `$HOME/.local/bin` (or `/usr/local/bin` with sudo).

**File:** `scripts/install.sh`

**Behavior:**
1. Detect OS (`uname -s`) and architecture (`uname -m`)
2. Fetch latest release tag from GitHub API (or accept `--version` flag)
3. Download the correct archive from `https://github.com/schoolofai/obeya/releases/download/v{version}/obeya_{version}_{os}_{arch}.tar.gz`
4. Extract the `ob` binary
5. Install to `~/.local/bin` (user-level, no sudo) or `/usr/local/bin` (with `--global` flag)
6. Print post-install PATH instructions if needed

**Usage:**
```bash
curl -fsSL https://raw.githubusercontent.com/schoolofai/obeya/main/scripts/install.sh | sh
# or with a specific version
curl -fsSL https://raw.githubusercontent.com/schoolofai/obeya/main/scripts/install.sh | sh -s -- --version 0.2.0
```

**Why this first:** Covers Linux immediately with zero infrastructure. Also gives macOS users a Homebrew-free option.

---

### Phase 2: Windows Install Script (PowerShell)

**What:** A PowerShell install script for Windows that downloads and installs the binary.

**File:** `scripts/install.ps1`

**Behavior:**
1. Detect architecture (x86_64 only for now)
2. Fetch latest release tag from GitHub API (or accept `-Version` parameter)
3. Download the zip archive from GitHub Releases
4. Extract `ob.exe`
5. Install to `$env:LOCALAPPDATA\obeya\bin`
6. Add to user PATH if not already present
7. Print instructions

**Usage:**
```powershell
irm https://raw.githubusercontent.com/schoolofai/obeya/main/scripts/install.ps1 | iex
```

---

### Phase 3: Scoop (Windows Package Manager)

**What:** Add a Scoop manifest so Windows users can `scoop install obeya`.

**Approach:** Create a Scoop bucket repository (`schoolofai/scoop-obeya`) with a manifest, and add GoReleaser config to auto-update it on release — similar to the existing Homebrew tap setup.

**GoReleaser addition to `.goreleaser.yml`:**
```yaml
scoops:
  - repository:
      owner: schoolofai
      name: scoop-obeya
      token: "{{ .Env.SCOOP_BUCKET_TOKEN }}"
    homepage: https://github.com/schoolofai/obeya
    description: "CLI Kanban board for humans and AI agents"
    license: MIT
```

**Usage:**
```powershell
scoop bucket add obeya https://github.com/schoolofai/scoop-obeya
scoop install obeya
```

**Infra needed:** New GitHub repo `schoolofai/scoop-obeya` + `SCOOP_BUCKET_TOKEN` secret.

---

### Phase 4: Linux Package Repositories (apt/yum via GoReleaser nFPM)

**What:** Produce `.deb` and `.rpm` packages and optionally host them.

**GoReleaser addition to `.goreleaser.yml`:**
```yaml
nfpms:
  - id: obeya
    package_name: obeya
    vendor: schoolofai
    homepage: https://github.com/schoolofai/obeya
    maintainer: "schoolofai"
    description: "CLI Kanban board for humans and AI agents"
    license: MIT
    formats:
      - deb
      - rpm
    bindir: /usr/local/bin
```

This attaches `.deb` and `.rpm` files to the GitHub Release. Users can download and install directly:
```bash
# Debian/Ubuntu
curl -LO https://github.com/schoolofai/obeya/releases/download/v0.2.0/obeya_0.2.0_linux_amd64.deb
sudo dpkg -i obeya_0.2.0_linux_amd64.deb

# RHEL/Fedora
curl -LO https://github.com/schoolofai/obeya/releases/download/v0.2.0/obeya_0.2.0_linux_amd64.rpm
sudo rpm -i obeya_0.2.0_linux_amd64.rpm
```

---

### Phase 5: Update README and Docs

Update `README.md` installation section with all methods, organized by platform:

- **macOS:** Homebrew (recommended), install script, go install
- **Linux:** Install script (recommended), deb/rpm packages, go install
- **Windows:** PowerShell script (recommended), Scoop, go install

---

## Implementation Order & Effort

| Phase | Effort | Impact | Depends On |
|-------|--------|--------|------------|
| 1. Shell install script | Small | High (Linux + macOS) | Nothing |
| 2. PowerShell install script | Small | High (Windows) | Nothing |
| 3. Scoop bucket | Small | Medium (Windows power users) | New repo + secret |
| 4. nFPM deb/rpm | Small | Medium (Linux sysadmins) | Nothing |
| 5. README update | Trivial | High (discoverability) | Phases 1-4 |

Phases 1 and 2 are independent and can be done in parallel. Phase 5 should be done last.

## Out of Scope

- **Snap/Flatpak:** Overkill for a CLI tool; install script covers the same users
- **Chocolatey/winget:** Higher maintenance burden; Scoop + PowerShell script covers Windows well
- **Hosted apt/yum repositories:** GitHub Releases with direct `.deb`/`.rpm` download is sufficient for now
