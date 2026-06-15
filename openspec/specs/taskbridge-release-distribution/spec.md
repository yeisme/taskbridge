# taskbridge-release-distribution Specification

## Purpose
TBD - created by archiving change taskbridge-release-distribution. Update Purpose after archive.
## Requirements
### Requirement: TaskBridge releases SHALL publish verifiable artifacts

TaskBridge SHALL publish release artifacts through GoReleaser from semver tags, with checksums, source archive, SBOM, archives, and Linux package assets.

#### Scenario: Release archives are named and checksummed
- **WHEN** a semver tag release runs
- **THEN** GoReleaser SHALL produce platform archives for supported Linux, macOS, and Windows architectures
- **AND** archive names SHALL include project, version, OS, and normalized architecture
- **AND** Windows archives SHALL use zip while non-Windows archives SHALL use tar.gz
- **AND** `checksums.txt` SHALL include sha256 checksums for released artifacts.

#### Scenario: SBOM is generated for release artifacts
- **WHEN** GoReleaser builds release archives
- **THEN** it SHALL generate an SPDX JSON SBOM for the archive artifacts
- **AND** the SBOM SHALL be published with the GitHub Release assets.

#### Scenario: Linux packages remain direct release assets
- **WHEN** GoReleaser builds Linux packages
- **THEN** it SHALL produce `.deb`, `.rpm`, and `.apk` package assets where supported
- **AND** those packages SHALL install the `taskbridge` binary without creating credentials, provider auth state, local task stores, or remote Todo writes
- **AND** documentation SHALL describe them as direct release assets, not as APT/YUM/APK repository channels.

### Requirement: Package-manager publishing SHALL use Homebrew cask and Scoop manifests

TaskBridge SHALL publish package-manager metadata for Homebrew and Scoop through generated GoReleaser output, not hand-written package files in this repository.

#### Scenario: Homebrew cask is published to the Yeisme tap
- **WHEN** a non-snapshot semver tag release is published
- **THEN** GoReleaser SHALL update `yeisme/homebrew-tap` under `Casks/taskbridge.rb`
- **AND** the cask SHALL install the `taskbridge` binary
- **AND** TaskBridge SHALL NOT keep a formula compatibility path in this change.

#### Scenario: Scoop manifest is published to the Yeisme bucket
- **WHEN** a non-snapshot semver tag release is published
- **THEN** GoReleaser SHALL update `yeisme/scoop-bucket` under `bucket/taskbridge.json`
- **AND** the manifest SHALL install the Windows release archive for `taskbridge`
- **AND** the manifest SHALL include homepage, description, and license metadata.

#### Scenario: Snapshot releases do not update package indexes
- **WHEN** a snapshot release rehearsal runs
- **THEN** GoReleaser SHALL build local artifacts
- **AND** it SHALL NOT publish Homebrew cask updates
- **AND** it SHALL NOT publish Scoop manifest updates.

### Requirement: Release workflows SHALL separate release and package-index permissions

TaskBridge release automation SHALL use distinct credentials for current-repository GitHub Releases and cross-repository package-index publishing.

#### Scenario: GitHub Release uses repository-scoped token
- **WHEN** the release workflow publishes GitHub Release assets
- **THEN** it SHALL use the current repository release token such as `${{ secrets.GITHUB_TOKEN }}`
- **AND** it SHALL NOT require a broad package publisher token for current-repository release asset upload.

#### Scenario: Tap and bucket writes use a separate publisher token
- **WHEN** GoReleaser pushes Homebrew or Scoop metadata
- **THEN** it SHALL receive a separate `PUBLISHER_TOKEN`
- **AND** the preferred source SHALL be a short-lived GitHub App token scoped to `homebrew-tap` and `scoop-bucket`
- **AND** any PAT fallback SHALL be documented as an operator exception rather than the primary design.

### Requirement: Post-release smoke SHALL verify released installs

TaskBridge SHALL verify released artifacts after publication with commands that require no provider credentials.

#### Scenario: Archive smoke validates checksum and executable startup
- **WHEN** a release is published or a post-release workflow is manually run for a tag
- **THEN** the workflow SHALL download `checksums.txt`
- **AND** download the runner platform archive
- **AND** verify sha256
- **AND** unpack the archive
- **AND** run `taskbridge --version`, `taskbridge --help`, and `taskbridge demo today`.

#### Scenario: Homebrew cask smoke validates package install
- **WHEN** the post-release smoke runs on macOS after cask publishing
- **THEN** it SHALL tap `yeisme/homebrew-tap`
- **AND** install `taskbridge` through `brew install --cask taskbridge`
- **AND** run `taskbridge --version` and `taskbridge demo today`.

#### Scenario: Scoop smoke is either verified or explicitly blocked
- **WHEN** the post-release workflow covers Windows install behavior
- **THEN** it SHALL either install from `yeisme/scoop-bucket` and run `taskbridge --version`
- **OR** record the exact public bucket, token, or runner prerequisite that blocks end-to-end Scoop install verification
- **AND** it SHALL NOT claim Scoop install verification when only manifest generation was tested.

### Requirement: Release documentation SHALL match implemented package channels

TaskBridge SHALL document only package channels implemented by this change and SHALL use real commands a human can run.

#### Scenario: Install docs describe supported channels
- **WHEN** a user reads README or release packaging docs
- **THEN** the documented install paths SHALL include Homebrew cask, Scoop, GitHub Release archives, Go install, and direct Linux package assets
- **AND** docs SHALL NOT advertise Chocolatey, Winget, APT/YUM/APK repositories, Docker images, or MCP registry publishing unless implemented.

#### Scenario: Install docs describe side-effect boundaries
- **WHEN** a user installs TaskBridge through a package manager
- **THEN** docs SHALL state that installation does not configure provider credentials, create OAuth tokens, initialize remote providers, or write Todo platform data
- **AND** docs SHALL direct users to run diagnostic and auth commands after installation.

#### Scenario: Release operator docs remove local machine assumptions
- **WHEN** a maintainer follows release management docs
- **THEN** command examples SHALL use real repository commands such as `task test`, `task test:integration`, `task release:check`, `task release:local`, and `git tag vX.Y.Z`
- **AND** docs SHALL NOT include local absolute paths, shell aliases, local wrappers, or agent-only command prefixes.

