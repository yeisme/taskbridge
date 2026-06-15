## Why

TaskBridge is ready for a repeatable package publishing path now that the CLI control plane is mostly stabilized. Its current release skeleton already builds archives, checksums, nFPM packages, a Homebrew formula, and a Scoop manifest, but it does not yet match the stronger `skillctl` release baseline.

This change makes TaskBridge release distribution boring and verifiable: one tag produces GitHub Release assets, checksums, SBOMs, Linux packages, a Homebrew cask, a Scoop manifest, and post-release smoke evidence. Package managers install only the `taskbridge` binary and static release metadata; they must not create credentials, mutate `~/.taskbridge`, log in to providers, or write remote Todo platforms.

## What Changes

- Align TaskBridge package publishing with the `skillctl` release pattern: GoReleaser-driven archives, checksums, source archive, SBOM, nFPM `.deb`/`.rpm`/`.apk` assets, Homebrew cask, and Scoop manifest.
- Migrate the Homebrew publishing surface from a formula-style `brews` stanza to a cask-style `homebrew_casks` stanza under `yeisme/homebrew-tap/Casks/taskbridge.rb`.
- Keep Scoop publishing under `yeisme/scoop-bucket/bucket/taskbridge.json` and ensure snapshot releases do not publish tap or bucket updates.
- Harden release workflows so tag releases use repository-scoped `GITHUB_TOKEN` for the GitHub Release and a separate short-lived publisher token for cross-repository tap/bucket writes.
- Add post-release smoke checks that verify downloaded archives, checksums, executable startup, `--help`, `--version`, and no-auth demo behavior.
- Rewrite release packaging docs and README install guidance in English with real commands a human can run.

## Impact

- Owner: `cli/taskbridge`.
- Primary files: `.goreleaser.yaml`, `.github/workflows/release.yml`, a new `.github/workflows/post-release.yml`, `docs/release-management.md`, optional `docs/release-packaging.md`, `README.md`, and release-related Taskfile entries only if needed.
- User-facing scope: install commands, package manager instructions, release operator docs, and post-release verification expectations.
- Security scope: package-manager publishing tokens, release artifacts, checksums, SBOMs, and avoidance of credential/state mutation during install.
- Out of scope: Chocolatey, Winget, APT/YUM/APK repositories, Docker images, MCP registry publishing, provider behavior changes, TaskBridge command semantics, OAuth onboarding, and automatic runtime state setup during installation.
