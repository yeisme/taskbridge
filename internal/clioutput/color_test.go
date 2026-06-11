package clioutput

import "testing"

func TestShouldColorNeverDisablesColor(t *testing.T) {
	if ShouldColor(ColorNever, false, "") {
		t.Fatal("ColorNever should disable color")
	}
}

func TestShouldColorNoColorEnvDisablesAuto(t *testing.T) {
	if ShouldColor(ColorAuto, true, "1") {
		t.Fatal("NO_COLOR should disable auto color")
	}
}

func TestShouldColorAlwaysIgnoresNoColor(t *testing.T) {
	if !ShouldColor(ColorAlways, false, "1") {
		t.Fatal("ColorAlways should force color")
	}
}
