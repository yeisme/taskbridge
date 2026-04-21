package loader

import (
	"context"
	"testing"
)

// TestLoadProvidersResultStructure verifies that LoadProviders returns a
// properly structured ProviderLoadResult with non-nil maps even when no
// providers are configured.
func TestLoadProvidersResultStructure(t *testing.T) {
	result := LoadProviders(context.Background())

	if result == nil {
		t.Fatal("LoadProviders returned nil result")
	}
	if result.Providers == nil {
		t.Error("Providers map is nil")
	}
	if result.Statuses == nil {
		t.Error("Statuses map is nil")
	}
}

// TestLoadProvidersStatusFields verifies that every entry in Statuses has
// the Name field set and a non-empty status.
func TestLoadProvidersStatusFields(t *testing.T) {
	result := LoadProviders(context.Background())

	for name, status := range result.Statuses {
		if status == nil {
			t.Fatalf("status entry with key %q is nil", name)
		}
		if status.Name == "" {
			t.Errorf("status entry with key %q has empty Name", name)
		}
		if status.Name != name {
			t.Errorf("status.Name=%q but map key=%q, mismatch", status.Name, name)
		}
	}
}

// TestLoadProvidersTargetFilter verifies that specifying a target limits
// loading to only that provider.
func TestLoadProvidersTargetFilter(t *testing.T) {
	result := LoadProviders(context.Background(), WithTarget("google"))

	if len(result.Statuses) != 1 {
		t.Errorf("expected 1 status entry for target=google, got %d", len(result.Statuses))
	}
	if _, ok := result.Statuses["google"]; !ok {
		t.Error("expected 'google' in statuses")
	}
}

// TestLoadProvidersAllProvidersCovered verifies that loading all providers
// produces status entries for every known provider.
func TestLoadProvidersAllProvidersCovered(t *testing.T) {
	expectedProviders := []string{"google", "microsoft", "todoist", "feishu", "ticktick", "dida"}

	result := LoadProviders(context.Background())

	for _, name := range expectedProviders {
		if _, ok := result.Statuses[name]; !ok {
			t.Errorf("expected status entry for provider %q", name)
		}
	}
}

// TestLoadProvidersFailedProviderNotInProvidersMap verifies that providers
// which fail to load or authenticate do not appear in the Providers map
// but do appear in Statuses with an error.
func TestLoadProvidersFailedProviderNotInProvidersMap(t *testing.T) {
	result := LoadProviders(context.Background())

	for name, status := range result.Statuses {
		if !status.Loaded || !status.Authenticated {
			if _, inProviders := result.Providers[name]; inProviders {
				t.Errorf("provider %q is failed/unauthenticated but present in Providers map", name)
			}
			if status.Error == "" {
				t.Errorf("provider %q is not loaded but has empty Error field", name)
			}
		} else {
			if _, inProviders := result.Providers[name]; !inProviders {
				t.Errorf("provider %q is loaded+authenticated but missing from Providers map", name)
			}
		}
	}
}

// TestLoadProvidersAuthenticatedProviderInMap verifies that an authenticated
// provider appears both in Providers and Statuses with correct fields.
func TestLoadProvidersAuthenticatedProviderInMap(t *testing.T) {
	result := LoadProviders(context.Background())

	for name, p := range result.Providers {
		status, ok := result.Statuses[name]
		if !ok {
			t.Errorf("provider %q in Providers map but missing from Statuses", name)
			continue
		}
		if !status.Loaded {
			t.Errorf("provider %q in Providers map but status.Loaded=false", name)
		}
		if !status.Authenticated {
			t.Errorf("provider %q in Providers map but status.Authenticated=false", name)
		}
		if status.Error != "" {
			t.Errorf("provider %q in Providers map but has error: %s", name, status.Error)
		}
		if p == nil {
			t.Errorf("provider %q in Providers map is nil", name)
		}
	}
}
