package releasecontract

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestReleaseDistributionConfigContract(t *testing.T) {
	config := readYAMLFile(t, ".goreleaser.yaml")

	archives := listAt(t, config, "archives")
	if len(archives) != 1 {
		t.Fatalf("archives count = %d, want 1", len(archives))
	}
	archive := asMap(t, archives[0], "archives[0]")
	nameTemplate := stringAt(t, archive, "name_template")
	for _, want := range []string{"{{ .ProjectName }}", "{{- .Version }}", "{{- title .Os }}", "x86_64", "arm64"} {
		if !strings.Contains(nameTemplate, want) {
			t.Fatalf("archive name_template missing %q:\n%s", want, nameTemplate)
		}
	}
	formatOverrides := listAt(t, archive, "format_overrides")
	if len(formatOverrides) != 1 {
		t.Fatalf("format_overrides count = %d, want 1", len(formatOverrides))
	}
	windowsOverride := asMap(t, formatOverrides[0], "format_overrides[0]")
	if got := stringAt(t, windowsOverride, "goos"); got != "windows" {
		t.Fatalf("windows format override goos = %q, want windows", got)
	}
	assertStringListContains(t, listAt(t, windowsOverride, "formats"), "zip", "windows format override")

	source := mapAt(t, config, "source")
	if enabled, ok := source["enabled"].(bool); !ok || !enabled {
		t.Fatalf("source.enabled = %v, want true", source["enabled"])
	}
	if got := stringAt(t, source, "name_template"); got != "{{ .ProjectName }}_{{ .Version }}_source" {
		t.Fatalf("source name_template = %q, want versioned source archive", got)
	}

	checksum := mapAt(t, config, "checksum")
	if got := stringAt(t, checksum, "name_template"); got != "checksums.txt" {
		t.Fatalf("checksum name_template = %q, want checksums.txt", got)
	}
	if got := stringAt(t, checksum, "algorithm"); got != "sha256" {
		t.Fatalf("checksum algorithm = %q, want sha256", got)
	}

	sboms := listAt(t, config, "sboms")
	if len(sboms) != 1 {
		t.Fatalf("sboms count = %d, want 1", len(sboms))
	}
	sbom := asMap(t, sboms[0], "sboms[0]")
	if got := stringAt(t, sbom, "artifacts"); got != "archive" {
		t.Fatalf("sbom artifacts = %q, want archive", got)
	}
	assertStringListContains(t, listAt(t, sbom, "documents"), "${artifact}.spdx.json", "sbom documents")

	nfpms := listAt(t, config, "nfpms")
	if len(nfpms) != 1 {
		t.Fatalf("nfpms count = %d, want 1", len(nfpms))
	}
	nfpm := asMap(t, nfpms[0], "nfpms[0]")
	if got := stringAt(t, nfpm, "bindir"); got != "/usr/bin" {
		t.Fatalf("nfpm bindir = %q, want /usr/bin", got)
	}
	if _, ok := nfpm["scripts"]; ok {
		t.Fatalf("nFPM config must not include install scripts that can create runtime state")
	}
	for _, format := range []string{"deb", "rpm", "apk"} {
		assertStringListContains(t, listAt(t, nfpm, "formats"), format, "nfpm formats")
	}

	homebrewCasks := listAt(t, config, "homebrew_casks")
	if len(homebrewCasks) != 1 {
		t.Fatalf("homebrew_casks count = %d, want 1", len(homebrewCasks))
	}
	cask := asMap(t, homebrewCasks[0], "homebrew_casks[0]")
	if got := stringAt(t, cask, "directory"); got != "Casks" {
		t.Fatalf("homebrew cask directory = %q, want Casks", got)
	}
	if got := stringAt(t, cask, "skip_upload"); got != "{{ .IsSnapshot }}" {
		t.Fatalf("homebrew cask skip_upload = %q, want snapshot guard", got)
	}
	if got := stringAt(t, mapAt(t, cask, "repository"), "token"); got != "{{ .Env.PUBLISHER_TOKEN }}" {
		t.Fatalf("homebrew cask token = %q, want PUBLISHER_TOKEN", got)
	}
	assertStringListContains(t, listAt(t, cask, "binaries"), "taskbridge", "homebrew cask binaries")
	if _, ok := config["brews"]; ok {
		t.Fatalf("release config must not keep Homebrew formula compatibility stanzas")
	}

	scoops := listAt(t, config, "scoops")
	if len(scoops) != 1 {
		t.Fatalf("scoops count = %d, want 1", len(scoops))
	}
	scoop := asMap(t, scoops[0], "scoops[0]")
	if got := stringAt(t, scoop, "directory"); got != "bucket" {
		t.Fatalf("scoop directory = %q, want bucket", got)
	}
	if got := stringAt(t, scoop, "skip_upload"); got != "{{ .IsSnapshot }}" {
		t.Fatalf("scoop skip_upload = %q, want snapshot guard", got)
	}
	if got := stringAt(t, mapAt(t, scoop, "repository"), "token"); got != "{{ .Env.PUBLISHER_TOKEN }}" {
		t.Fatalf("scoop token = %q, want PUBLISHER_TOKEN", got)
	}
}

func TestReleaseWorkflowPermissionAndSmokeContract(t *testing.T) {
	releaseWorkflow := readYAMLFile(t, ".github/workflows/release.yml")
	jobs := mapAt(t, releaseWorkflow, "jobs")
	snapshotJob := mapAt(t, jobs, "snapshot")
	snapshotSyft := findStep(t, listAt(t, snapshotJob, "steps"), "Set up Syft")
	if got := stringAt(t, snapshotSyft, "uses"); got != "anchore/sbom-action/download-syft@v0.20.6" {
		t.Fatalf("snapshot syft setup action = %q, want pinned download-syft action", got)
	}

	releaseJob := mapAt(t, jobs, "release")
	permissions := mapAt(t, releaseJob, "permissions")
	if got := stringAt(t, permissions, "contents"); got != "write" {
		t.Fatalf("release job contents permission = %q, want write", got)
	}
	if got := stringAt(t, permissions, "id-token"); got != "write" {
		t.Fatalf("release job id-token permission = %q, want write for release provenance support", got)
	}
	steps := listAt(t, releaseJob, "steps")
	publish := findStep(t, steps, "Publish release")
	publishEnv := mapAt(t, publish, "env")
	if got := stringAt(t, publishEnv, "GITHUB_TOKEN"); got != "${{ secrets.GITHUB_TOKEN }}" {
		t.Fatalf("publish GITHUB_TOKEN = %q, want repository token", got)
	}
	if got := stringAt(t, publishEnv, "PUBLISHER_TOKEN"); !strings.Contains(got, "publisher-token") {
		t.Fatalf("publish PUBLISHER_TOKEN = %q, want resolved publisher token", got)
	}
	publisherToken := findStep(t, steps, "Create package publisher GitHub App token")
	if got := stringAt(t, publisherToken, "uses"); got != "actions/create-github-app-token@v3" {
		t.Fatalf("publisher token action = %q, want create-github-app-token v3", got)
	}
	releaseSyft := findStep(t, steps, "Set up Syft")
	if got := stringAt(t, releaseSyft, "uses"); got != "anchore/sbom-action/download-syft@v0.20.6" {
		t.Fatalf("release syft setup action = %q, want pinned download-syft action", got)
	}

	postReleaseWorkflow := readYAMLFile(t, ".github/workflows/post-release.yml")
	postJobs := mapAt(t, postReleaseWorkflow, "jobs")
	for _, jobName := range []string{"archive-smoke-unix", "archive-smoke-windows", "homebrew-cask-smoke", "scoop-smoke"} {
		if _, ok := postJobs[jobName]; !ok {
			t.Fatalf("post-release workflow missing job %q", jobName)
		}
	}
	postWorkflowText := readTextFile(t, ".github/workflows/post-release.yml")
	for _, want := range []string{"gh release download", "checksums.txt", "sha256sum --check", "Get-FileHash -Algorithm SHA256", "brew trust yeisme/tap", "brew install --cask taskbridge", "scoop install taskbridge", "RUN_SCOOP_SMOKE"} {
		if !strings.Contains(postWorkflowText, want) {
			t.Fatalf("post-release workflow missing %q", want)
		}
	}
}

func TestReleaseDocumentationContract(t *testing.T) {
	for _, path := range []string{"README.md", "docs/release-management.md", "docs/release-packaging.md"} {
		text := readTextFile(t, path)
		if regexp.MustCompile(`/(Users|home|workspaces)/`).MatchString(text) {
			t.Fatalf("%s contains a local absolute path", path)
		}
		for _, forbidden := range []string{"brew install taskbridge", "scoop install yeisme/taskbridge", "apt repository", "yum repository"} {
			if strings.Contains(text, forbidden) {
				t.Fatalf("%s contains stale release guidance %q", path, forbidden)
			}
		}
	}

	readme := readTextFile(t, "README.md")
	for _, want := range []string{"brew install --cask taskbridge", "scoop install taskbridge", "GitHub Release archives", "go install github.com/yeisme/taskbridge@latest", "Direct Linux package assets"} {
		if !strings.Contains(readme, want) {
			t.Fatalf("README missing release install guidance %q", want)
		}
	}

	releaseDocs := readTextFile(t, "docs/release-management.md")
	for _, want := range []string{"task release:check", "task release:local", "git tag vX.Y.Z", "secrets.GITHUB_TOKEN", "PUBLISHER_TOKEN", "id-token: write", "checksums.txt", "SPDX JSON SBOMs"} {
		if !strings.Contains(releaseDocs, want) {
			t.Fatalf("release management docs missing %q", want)
		}
	}
}

func readTextFile(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(filepath.Join("..", "..", path))
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(content)
}

func readYAMLFile(t *testing.T, path string) map[string]any {
	t.Helper()
	var document map[string]any
	if err := yaml.Unmarshal([]byte(readTextFile(t, path)), &document); err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	return document
}

func mapAt(t *testing.T, document map[string]any, key string) map[string]any {
	t.Helper()
	value, ok := document[key]
	if !ok {
		t.Fatalf("missing key %q", key)
	}
	return asMap(t, value, key)
}

func listAt(t *testing.T, document map[string]any, key string) []any {
	t.Helper()
	value, ok := document[key]
	if !ok {
		t.Fatalf("missing key %q", key)
	}
	items, ok := value.([]any)
	if !ok {
		t.Fatalf("%s has type %T, want list", key, value)
	}
	return items
}

func stringAt(t *testing.T, document map[string]any, key string) string {
	t.Helper()
	value, ok := document[key]
	if !ok {
		t.Fatalf("missing key %q", key)
	}
	text, ok := value.(string)
	if !ok {
		t.Fatalf("%s has type %T, want string", key, value)
	}
	return text
}

func asMap(t *testing.T, value any, name string) map[string]any {
	t.Helper()
	document, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("%s has type %T, want map", name, value)
	}
	return document
}

func assertStringListContains(t *testing.T, list []any, want, name string) {
	t.Helper()
	for _, item := range list {
		if item == want {
			return
		}
	}
	t.Fatalf("%s = %#v, missing %q", name, list, want)
}

func findStep(t *testing.T, steps []any, name string) map[string]any {
	t.Helper()
	for _, item := range steps {
		step := asMap(t, item, "step")
		if step["name"] == name {
			return step
		}
	}
	t.Fatalf("missing workflow step %q", name)
	return nil
}
