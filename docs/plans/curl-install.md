# Curl Install Script — Implementation Plan

## Overview

Add a `curl | sh` installer so users can install jot-cli without Homebrew or Go. This is the standard pattern for Go CLI tools (goreleaser, golangci-lint, etc.) and provides the lowest-friction install path.

Usage:

```bash
curl -fsSL https://raw.githubusercontent.com/danjdewhurst/jot-cli/main/install.sh | sh
```

---

## 1. Create `install.sh` (project root)

A POSIX-compatible shell script that:

### Detect OS and architecture

Map `uname -s` / `uname -m` to GoReleaser's naming conventions:

| `uname -s` | `uname -m`      | Archive OS | Archive Arch |
|-------------|-----------------|------------|--------------|
| Darwin      | arm64           | darwin     | arm64        |
| Darwin      | x86_64          | darwin     | amd64        |
| Linux       | aarch64         | linux      | arm64        |
| Linux       | x86_64          | linux      | amd64        |

Abort with a clear error on unsupported platforms (e.g. Windows/WSL FreeBSD, 32-bit).

### Resolve version

- Default: query `https://api.github.com/repos/danjdewhurst/jot-cli/releases/latest` and extract the tag name
- Override: `VERSION=0.3.0 curl ... | sh` to pin a specific release

### Download and verify

1. Download `jot-cli_{VERSION}_{os}_{arch}.tar.gz` from GitHub Releases
2. Download `checksums.txt`
3. Verify SHA256 using `sha256sum` (Linux) or `shasum -a 256` (macOS)
4. Abort if checksum fails

### Extract and install

1. Extract `jot-cli` binary from the tarball
2. Install to `${INSTALL_DIR:-$HOME/.local/bin}` (matches Makefile convention)
3. Create `j` symlink → `jot-cli`
4. Print success message
5. If install dir is not in `$PATH`, print a hint to add it

### Environment variable overrides

| Variable      | Default          | Purpose                  |
|---------------|------------------|--------------------------|
| `VERSION`     | latest release   | Pin a specific version   |
| `INSTALL_DIR` | `~/.local/bin`   | Custom install location  |

### Dependencies

Only standard tools: `curl`, `tar`, `sha256sum` or `shasum`. No `sudo` required by default.

---

## 2. Update `README.md`

Add a "Shell script" install method between Homebrew and Go install:

```markdown
### Shell script

```bash
curl -fsSL https://raw.githubusercontent.com/danjdewhurst/jot-cli/main/install.sh | sh
```

To install a specific version or to a custom directory:

```bash
VERSION=0.3.0 INSTALL_DIR=/usr/local/bin curl -fsSL https://raw.githubusercontent.com/danjdewhurst/jot-cli/main/install.sh | sh
```
```

---

## Verification

1. Run `shellcheck install.sh` — no warnings
2. Test: `sh install.sh` — binary at `~/.local/bin/jot-cli`, `j` symlink works, correct version
3. Test: `INSTALL_DIR=/tmp/test-jot sh install.sh` — custom dir works
4. Test error path: run on unsupported arch or without `curl`
