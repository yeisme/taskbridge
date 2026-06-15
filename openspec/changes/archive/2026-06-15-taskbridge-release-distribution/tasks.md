# Tasks: TaskBridge Release Distribution

Owner: `cli/taskbridge`  
Primary surface: `taskbridge` package publishing  
Project type: Go Cobra CLI

Parallel lanes:

- Lane A: GoReleaser artifact contract.
- Lane B: GitHub Actions release and smoke workflows.
- Lane C: package-manager docs and install guidance.
- Lane D: verification and closeout evidence.

## 0. Baseline

- [x] 0.1 Record the current release config state by running `task release:check` from `cli/taskbridge`.

  Owner: `cli/taskbridge`  
  Lane: sequential  
  Depends on: none  
  Acceptance: record whether GoReleaser is available and whether `.goreleaser.yaml` validates before edits.  
  Validation command: `task release:check`  
  Observed baseline: `task release:check` failed before edits because `goreleaser` was not installed in PATH (`"goreleaser": executable file not found in $PATH`, Task exit 201).  
  Failure re-check: implementation must compare against `cli/skillctl/.goreleaser.yaml` and validate with an installed GoReleaser before closeout.

- [x] 0.2 Record the current workflow and docs publishing surface.

  Owner: `cli/taskbridge`  
  Lane: sequential  
  Depends on: none  
  Acceptance: note existing Homebrew, Scoop, nFPM, checksum, release workflow, and docs paths that will change.  
  Observed baseline: `.goreleaser.yaml` already builds archives, source archive, `taskbridge_<version>_checksums.txt`, nFPM `.deb`/`.rpm`/`.apk`, Scoop manifest, and Homebrew `brews` Formula output under `Formula`; `.github/workflows/release.yml` exists but uses one `PUBLISHER_TOKEN` as both `GITHUB_TOKEN` and package publisher token; `.github/workflows/post-release.yml` is missing; `docs/release-management.md` still has Chinese prose and `/Users/yeshugen/...`; README install docs still use `brew install taskbridge` and `scoop install yeisme/taskbridge`.  
  Failure re-check: if release files move during implementation, re-read the subproject tree before editing.

## 1. Lane A: GoReleaser artifact contract

- [x] 1.1 Update `.goreleaser.yaml` archive and checksum naming.

  Owner: `cli/taskbridge`  
  Lane: A  
  Depends on: 0.1, 0.2  
  Acceptance: archives include project, version, OS, and normalized architecture; Windows stays zip; non-Windows stays tar.gz; checksum output is `checksums.txt` using sha256.  
  Validation command: `task release:check`  
  Observed result: `task release:check` passed after GoReleaser v2.12.6 was installed; `task release:local` generated versioned archives such as `taskbridge_1.0.6-SNAPSHOT-4895fe2_Linux_x86_64.tar.gz` and `checksums.txt`.

- [x] 1.2 Add SPDX SBOM generation for release archives.

  Owner: `cli/taskbridge`  
  Lane: A  
  Depends on: 1.1  
  Acceptance: `.goreleaser.yaml` includes an SBOM stanza that generates an archive-level SPDX JSON document.  
  Validation command: `task release:local`  
  Observed result: `task release:local` generated archive SBOM files including `taskbridge_1.0.6-SNAPSHOT-4895fe2_Linux_x86_64.tar.gz.spdx.json`.

- [x] 1.3 Preserve nFPM `.deb`, `.rpm`, and `.apk` release assets.

  Owner: `cli/taskbridge`  
  Lane: A  
  Depends on: 1.1  
  Acceptance: direct Linux package artifacts remain configured, install the `taskbridge` binary under `/usr/bin`, and do not create provider credentials or runtime state.  
  Validation command: `task release:local`  
  Observed result: `task release:local` generated `.deb`, `.rpm`, and `.apk` files for linux amd64 and arm64; nFPM config contains only README/release docs/license contents and no install scripts.

- [x] 1.4 Migrate Homebrew publishing from formula to cask.

  Owner: `cli/taskbridge`  
  Lane: A  
  Depends on: 1.1  
  Acceptance: `.goreleaser.yaml` uses `homebrew_casks`, writes `Casks/taskbridge.rb` in `yeisme/homebrew-tap`, installs the `taskbridge` binary, and uses `skip_upload: "{{ .IsSnapshot }}"`.  
  Validation command: `task release:check`  
  Observed result: GoReleaser config validates, no `brews` stanza remains, and `task release:local` generated `dist/homebrew/Casks/taskbridge.rb`.

- [x] 1.5 Harden Scoop manifest publishing.

  Owner: `cli/taskbridge`  
  Lane: A  
  Depends on: 1.1  
  Acceptance: `.goreleaser.yaml` writes `bucket/taskbridge.json` in `yeisme/scoop-bucket`, uses release archives, sets homepage/description/license, and skips upload for snapshots.  
  Validation command: `task release:check`  
  Observed result: GoReleaser config validates and `task release:local` generated `dist/scoop/bucket/taskbridge.json`.

## 2. Lane B: GitHub Actions release and smoke workflows

- [x] 2.1 Harden the release workflow token boundary.

  Owner: `cli/taskbridge`  
  Lane: B  
  Depends on: 0.2  
  Acceptance: `.github/workflows/release.yml` uses `${{ secrets.GITHUB_TOKEN }}` for the current repository release and a separate `PUBLISHER_TOKEN` for tap/bucket writes, preferably minted through `actions/create-github-app-token`.  
  Validation command: `task release:check` plus workflow YAML review.  
  Observed result: release workflow parses as YAML, uses `${{ secrets.GITHUB_TOKEN }}` for GoReleaser `GITHUB_TOKEN`, mints a package publisher token with `actions/create-github-app-token@v3`, and falls back to `secrets.PUBLISHER_TOKEN` only with an explicit warning.

- [x] 2.2 Add snapshot release rehearsal smoke.

  Owner: `cli/taskbridge`  
  Lane: B  
  Depends on: 2.1  
  Acceptance: manual `workflow_dispatch` builds snapshot artifacts without publishing package-manager metadata and runs local `taskbridge --version`, `taskbridge --help`, and `taskbridge demo today`.  
  Validation command: `task release:local`  
  Observed result: `release.yml` snapshot job runs GoReleaser snapshot publish-skip and smokes the built binary with `--version`, `--help`, and `demo today`; local `task release:local` plus binary smoke passed.

- [x] 2.3 Add tag release gates.

  Owner: `cli/taskbridge`  
  Lane: B  
  Depends on: 2.1  
  Acceptance: semver tag releases run dependency verification, tests, integration evidence, release config check, and release binary smoke before publishing.  
  Validation commands: `task test`, `task test:integration`, `task release:check`.  
  Observed result: `release.yml` tag job runs module download/verify/tidy diff, `task test`, `task test:integration`, lint, GoReleaser check, release build smoke, then publish.

- [x] 2.4 Add post-release workflow for archive and Homebrew smoke.

  Owner: `cli/taskbridge`  
  Lane: B  
  Depends on: 1.1, 1.4, 2.3  
  Acceptance: new `.github/workflows/post-release.yml` triggers on `release.published` and manual tag input, downloads `checksums.txt`, verifies the runner archive, unpacks it, runs `--version`, `--help`, `demo today`, installs via Homebrew cask on macOS, and repeats binary smoke.  
  Validation command: workflow YAML parse plus local artifact smoke.  
  Observed result: `post-release.yml` parses as YAML and includes archive checksum smoke plus `brew tap yeisme/tap https://github.com/yeisme/homebrew-tap`, `brew install --cask taskbridge`, `taskbridge --version`, `taskbridge --help`, and `taskbridge demo today`.

- [x] 2.5 Add Windows Scoop smoke or record its gated prerequisite.

  Owner: `cli/taskbridge`  
  Lane: B  
  Depends on: 1.5, 2.4  
  Acceptance: Windows runner either installs from the Scoop bucket and runs `taskbridge --version`, or this task records the exact missing public bucket/token prerequisite and keeps manifest generation covered by GoReleaser validation.  
  Validation command: workflow YAML parse and GoReleaser manifest generation.  
  Observed result: `post-release.yml` includes a Windows Scoop smoke gated by `RUN_SCOOP_SMOKE=true`; otherwise it emits a notice that Scoop smoke is skipped until `yeisme/scoop-bucket` is accessible from GitHub-hosted Windows runners and Scoop bootstrap is allowed. Manifest generation is covered by `task release:local`.

## 3. Lane C: package-manager docs and install guidance

- [x] 3.1 Rewrite release management docs.

  Owner: `cli/taskbridge`  
  Lane: C  
  Depends on: 1.4, 1.5, 2.1  
  Acceptance: `docs/release-management.md` is English, removes local absolute paths, documents tag release flow, token model, package channels, rollback behavior, and real commands.  
  Validation command: manual read plus `openspec validate taskbridge-release-distribution --strict`.  
  Observed result: release management docs are English, include required commands and token model, and stale local path/formula references were removed.

- [x] 3.2 Add or update release packaging docs.

  Owner: `cli/taskbridge`  
  Lane: C  
  Depends on: 1.1, 1.2, 1.3, 1.4, 1.5  
  Acceptance: docs describe Homebrew cask, Scoop, GitHub archives, Go install, and direct nFPM asset installs; docs state packages do not create credentials, provider auth, or runtime task state.  
  Validation command: manual read plus `openspec validate taskbridge-release-distribution --strict`.  
  Observed result: `docs/release-packaging.md` documents only supported channels and states installers do not create provider credentials, OAuth tokens, local task stores, or remote Todo writes.

- [x] 3.3 Update README install section.

  Owner: `cli/taskbridge`  
  Lane: C  
  Depends on: 3.1, 3.2  
  Acceptance: README install examples use Homebrew cask, Scoop, GitHub Release archives, and source build/go install commands that match the actual module and release artifact names.  
  Validation command: manual read plus `taskbridge --help` after local build.  
  Observed result: README now uses `brew install --cask taskbridge`, `scoop install taskbridge`, Release archive guidance, direct Linux package asset guidance, `go install github.com/yeisme/taskbridge@latest`, and `task build` for source builds.

## 4. Lane D: verification and closeout evidence

- [x] 4.1 Run focused local release verification.

  Owner: `cli/taskbridge`  
  Lane: D  
  Depends on: 1.1, 1.2, 1.3, 1.4, 1.5, 2.1, 2.2, 3.1, 3.2, 3.3  
  Acceptance: local release checks, tests, and docs validation have observed results recorded in this task file.  
  Validation commands: `task release:check`, `task release:local`, `task test`, `task test:integration`, `openspec validate taskbridge-release-distribution --strict`, `task check`.  
  Observed result: local gates passed. Latest `task test:integration` wrote redacted evidence `temp/integration-test-runs/20260615T120904Z-1788151/summary.json` with `status=success` and `exit_code=0`; `task check` passed with golangci-lint `0 issues`; workflow YAML parse passed for release and post-release workflows.

- [x] 4.2 Run release artifact smoke against built dist.

  Owner: `cli/taskbridge`  
  Lane: D  
  Depends on: 4.1  
  Acceptance: a built release binary runs `--version`, `--help`, and `demo today`; checksum and archive contents are inspected through generated dist artifacts.  
  Validation command: use the generated `dist/` artifacts from `task release:local`.  
  Observed result: `(cd dist && sha256sum --check checksums.txt --ignore-missing)` verified release archives, SBOMs, nFPM packages, and source archive; `dist/taskbridge_linux_amd64_v1/taskbridge --version`, `--help`, and `demo today` all passed.

- [x] 4.3 Record first real tag post-release evidence or blocker.

  Owner: `cli/taskbridge`  
  Lane: D  
  Depends on: 2.4, 2.5, 4.1, 4.2  
  Acceptance: after the first real semver tag using this change, record GitHub Release, Homebrew cask, and Scoop evidence or exact external blockers in this task file before archiving the change.  
  Validation commands: `brew tap yeisme/tap https://github.com/yeisme/homebrew-tap`, `brew install --cask taskbridge`, `taskbridge --version`, `taskbridge demo today`, `scoop bucket add yeisme https://github.com/yeisme/scoop-bucket`, `scoop install taskbridge`, `taskbridge --version`.  
  External blocker recorded: no real semver tag from this unmerged working tree has been published yet, so public GitHub Release, Homebrew cask, and Scoop install smoke cannot produce direct evidence in this implementation session. The release and post-release workflows now collect that evidence after tag publication; Scoop smoke is additionally gated by `RUN_SCOOP_SMOKE=true` until `yeisme/scoop-bucket` access and Scoop bootstrap are allowed on GitHub-hosted Windows runners.
