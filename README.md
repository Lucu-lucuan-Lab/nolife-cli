
---

<img src="assets/nolife-logo.avif" alt="NoSleep CLI" width="580">

**An interactive CLI that cuts through ad-walls to batch-download anime seamlessly.**

<p>
  <a href="https://github.com/Lucu-lucuan-Lab/nolife-cli/actions/workflows/ci.yml"><img src="https://github.com/Lucu-lucuan-Lab/nolife-cli/actions/workflows/ci.yml/badge.svg" alt="CI"></a>
  <a href="https://github.com/Lucu-lucuan-Lab/nolife-cli/releases"><img src="https://img.shields.io/github/v/release/Lucu-lucuan-Lab/nolife-cli?label=release" alt="Latest release"></a>
  <img src="https://img.shields.io/badge/platform-Windows%20%7C%20macOS%20%7C%20Linux-0078D4" alt="Platform">
  <img src="https://img.shields.io/badge/Go-1.26.2+-00ADD8" alt="Go 1.26.2+">
</p>

<p>
  <code>nolife download "https://site/series"</code>
  &nbsp;|&nbsp;
  <code>nolife search "frieren"</code>
  &nbsp;|&nbsp;
  <code>nolife resume</code>
</p>

---

nolife is a CLI tool designed to download anime episodes from various streaming and sharing sites. It uses browser automation to bypass link protection and navigate file-hosting pages, providing a seamless download experience.

*Note: nolife requires Google Chrome or Chromium to be installed on your system.*

## Quick Install (Windows)

```powershell
irm https://raw.githubusercontent.com/Lucu-lucuan-Lab/nolife-cli/main/install.ps1 | iex
```

## Install

### Windows

Run the following command in PowerShell to download and install the latest release automatically:

```powershell
irm https://raw.githubusercontent.com/Lucu-lucuan-Lab/nolife-cli/main/install.ps1 | iex
```

The installer:
- downloads the binary for the current Windows architecture
- verifies the binary with the release SHA-256 checksum
- installs `nolife.exe` to `%LOCALAPPDATA%\Programs\nolife`
- appends the install directory to the User `Path`
- checks for the presence of Google Chrome

If your system restricts script execution, use this fallback:
```powershell
powershell -ExecutionPolicy ByPass -c "irm https://raw.githubusercontent.com/Lucu-lucuan-Lab/nolife-cli/main/install.ps1 | iex"
```

### macOS / Linux

Run the following command in Terminal to install the latest release to `/usr/local/bin`:

```bash
curl -sSL https://raw.githubusercontent.com/Lucu-lucuan-Lab/nolife-cli/main/install.sh | bash
```

### Manual Install

Download the binary for your architecture from the latest release. Rename the file to `nolife` (or `nolife.exe` on Windows), place it in a directory on your system `PATH`, and verify it against `checksums.txt` from the same release.

## Update

To update, simply run the installer script again. It will automatically replace the local binary with the latest release. Ensure nolife is not actively downloading before updating.

## Uninstall

### Windows
Run the uninstaller in PowerShell:
```powershell
irm https://raw.githubusercontent.com/Lucu-lucuan-Lab/nolife-cli/main/uninstall.ps1 | iex
```

### macOS / Linux
```bash
sudo rm /usr/local/bin/nolife
```

## Usage

Download all episodes from a series URL:
```bash
nolife download "https://site.com/series/title"
```

Search for a series by name:
```bash
nolife search "frieren"
```

Download a specific episode range and set the quality:
```bash
nolife download "https://site.com/series/title" --episodes 1-5 --quality 1080p
```

Download specific individual episodes:
```bash
nolife download "https://site.com/series/title" --episodes 1,3,7
```

List available episodes without downloading:
```bash
nolife list "https://site.com/series/title"
```

Resume any interrupted downloads:
```bash
nolife resume
```

Show the installed version:
```bash
nolife version
```

## Options

These flags can be appended to the `download` command:

| Flag              | Default | Description                                                                |
|-------------------|---------|----------------------------------------------------------------------------|
| `-o, --output`    | `""`    | Output directory (e.g., `-o ./my-anime`).                                  |
| `-e, --episodes`  | `""`    | Episode selection (e.g., `1-5`, `1,3,7`, `all`, `latest`).                 |
| `-q, --quality`   | `""`    | Preferred quality (`highest`, `1080p`, `720p`, `480p`).                    |
| `--concurrent`    | `0`     | Number of concurrent downloads (0 = use config default).                   |
| `--skip-existing` | `true`  | Skip downloading files that already exist in the output directory.         |

These flags apply globally:

| Flag             | Default | Description                                                                |
|------------------|---------|----------------------------------------------------------------------------|
| `--headless`     | `true`  | Run the browser in the background without showing a window.                |
| `--no-headless`  | `false` | Show the browser window (useful for debugging or solving captchas).        |
| `--config`       | `""`    | Path to a custom config file.                                              |

## Build

Requirements:
- Go 1.26.2 or newer
- Google Chrome

Build from source:
```bash
go build -o nolife ./cmd/nolife
```

Run tests:
```bash
go test ./...
```

## Release

Releases are built automatically by GitHub Actions when a version tag is pushed:

```bash
git tag v0.1.0
git push origin v0.1.0
```

The release workflow builds binaries for Windows, macOS, and Linux (amd64 and arm64), generates SHA-256 checksums, and publishes them to the GitHub release page.
