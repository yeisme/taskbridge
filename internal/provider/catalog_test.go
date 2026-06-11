package provider

import (
	"testing"
)

// TestGetAllProviderNames_Consistency verifies catalog order, dedup, and inclusion.
func TestGetAllProviderNames_Consistency(t *testing.T) {
	names := GetAllProviderNames()

	// Must have exactly 6 providers
	if len(names) != 6 {
		t.Fatalf("len(GetAllProviderNames()) = %d, want 6", len(names))
	}

	// Must be in canonical order
	want := []string{"google", "microsoft", "feishu", "ticktick", "dida", "todoist"}
	for i, w := range want {
		if names[i] != w {
			t.Errorf("names[%d] = %q, want %q", i, names[i], w)
		}
	}

	// No duplicates
	seen := make(map[string]bool, len(names))
	for _, n := range names {
		if seen[n] {
			t.Errorf("duplicate provider name: %q", n)
		}
		seen[n] = true
	}
}

// TestGetAllProviders_MatchesProviderNames verifies GetAllProviders returns
// definitions in the same order as GetAllProviderNames.
func TestGetAllProviders_MatchesProviderNames(t *testing.T) {
	names := GetAllProviderNames()
	defs := GetAllProviders()

	if len(defs) != len(names) {
		t.Fatalf("GetAllProviders() len = %d, GetAllProviderNames() len = %d", len(defs), len(names))
	}
	for i, d := range defs {
		if d.Name != names[i] {
			t.Errorf("defs[%d].Name = %q, want %q", i, d.Name, names[i])
		}
	}
}

// TestIsValidProvider_AllCatalogNames verifies every catalog name is valid.
func TestIsValidProvider_AllCatalogNames(t *testing.T) {
	for _, name := range GetAllProviderNames() {
		if !IsValidProvider(name) {
			t.Errorf("IsValidProvider(%q) = false, want true", name)
		}
	}
}

// TestIsValidProvider_UnknownName verifies unknown names are rejected.
func TestIsValidProvider_UnknownName(t *testing.T) {
	if IsValidProvider("nonexistent") {
		t.Error("IsValidProvider(\"nonexistent\") = true, want false")
	}
}

// TestGetProviderDefinition_AllProviders verifies each provider has a definition.
func TestGetProviderDefinition_AllProviders(t *testing.T) {
	for _, name := range GetAllProviderNames() {
		def, ok := GetProviderDefinition(name)
		if !ok {
			t.Errorf("GetProviderDefinition(%q) returned false", name)
			continue
		}
		if def.Name == "" {
			t.Errorf("GetProviderDefinition(%q).Name is empty", name)
		}
		if def.DisplayName == "" {
			t.Errorf("GetProviderDefinition(%q).DisplayName is empty", name)
		}
		if def.ShortName == "" {
			t.Errorf("GetProviderDefinition(%q).ShortName is empty", name)
		}
	}
}

// TestResolveProviderName_CaseInsensitive verifies case-insensitive resolution.
func TestResolveProviderName_CaseInsensitive(t *testing.T) {
	cases := map[string]string{
		"Google":    "google",
		"MICROSOFT": "microsoft",
		"Feishu":    "feishu",
		"TODOIST":   "todoist",
	}
	for input, want := range cases {
		if got := ResolveProviderName(input); got != want {
			t.Errorf("ResolveProviderName(%q) = %q, want %q", input, got, want)
		}
	}
}
