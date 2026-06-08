package provider

import "testing"

func TestGetAllProviderNamesUsesCanonicalOrder(t *testing.T) {
	names := GetAllProviderNames()
	want := []string{"google", "microsoft", "feishu", "ticktick", "dida", "todoist"}

	if len(names) != len(want) {
		t.Fatalf("len(GetAllProviderNames())=%d, want %d", len(names), len(want))
	}
	for i := range want {
		if names[i] != want[i] {
			t.Fatalf("GetAllProviderNames()[%d]=%q, want %q", i, names[i], want[i])
		}
	}
}

func TestResolveProviderNameAliases(t *testing.T) {
	cases := map[string]string{
		"g":           "google",
		"ms":          "microsoft",
		"tick":        "ticktick",
		"tick-cn":     "dida",
		"ticktick_cn": "dida",
		"todo":        "todoist",
	}

	for input, want := range cases {
		if got := ResolveProviderName(input); got != want {
			t.Fatalf("ResolveProviderName(%q)=%q, want %q", input, got, want)
		}
	}
}
