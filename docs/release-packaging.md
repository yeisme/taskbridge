# TaskBridge Release Packaging

TaskBridge supports five install channels. All channels install only the `taskbridge` binary and static release metadata. Installers do not create provider credentials, OAuth tokens, local task stores, or remote Todo writes.

After installing, run the no-auth onboarding checks first:

```bash
taskbridge doctor
taskbridge quickstart
taskbridge demo today
```

Then configure providers explicitly with commands such as:

```bash
taskbridge auth status
taskbridge auth login microsoft
taskbridge auth login todoist
taskbridge provider list
```

## Homebrew Cask

Use Homebrew on macOS or Linux:

```bash
brew tap yeisme/tap
brew install taskbridge
```

The formula is generated during non-snapshot releases and published to `yeisme/homebrew-tap/Formula/taskbridge.rb`.

## Scoop

Use Scoop on Windows:

```powershell
scoop bucket add yeisme https://github.com/yeisme/scoop-bucket
scoop install taskbridge
```

The manifest is generated during non-snapshot releases and published to `yeisme/scoop-bucket/bucket/taskbridge.json`.

## GitHub Release Archives

Download the archive for your platform from the GitHub Release page:

- Linux and macOS: `taskbridge_X.Y.Z_<OS>_<arch>.tar.gz`
- Windows: `taskbridge_X.Y.Z_Windows_<arch>.zip`

Archive names include the project, version, title-cased OS, and normalized architecture such as `x86_64` or `arm64`.

Verify the selected download with the release checksum file before running it. Example for Linux x86_64:

```bash
archive=taskbridge_X.Y.Z_Linux_x86_64.tar.gz
awk -v archive="$archive" '$2 == archive { print }' checksums.txt > selected-checksums.txt
test -s selected-checksums.txt
sha256sum --check selected-checksums.txt
```

Unpack the archive, place `taskbridge` on your `PATH`, and smoke-test the binary:

```bash
taskbridge --version
taskbridge --help
taskbridge demo today
```

## Go Install

Developers with Go 1.25 or newer can install directly from the module:

```bash
go install github.com/yeisme/taskbridge@latest
```

For a source checkout, build through the project task so the local binary receives the same buildinfo ldflags used by release builds:

```bash
git clone https://github.com/yeisme/taskbridge.git
cd taskbridge
task build
```

## Direct Linux Package Assets

Each tag release publishes nFPM-generated `.deb`, `.rpm`, and `.apk` files as direct GitHub Release assets. Download the package file from the release page and install it with your platform's local package command.

These files are direct assets only. TaskBridge does not publish repository-backed APT, YUM, or APK channels for this release path.
