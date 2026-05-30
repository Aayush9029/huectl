package tui

import "testing"

func TestColorPickerSelectionAndMovement(t *testing.T) {
	picker := newColorPicker("choose a color", "desk")

	if got := picker.Selected().Value; got != "warm" {
		t.Fatalf("initial selection = %q, want warm", got)
	}

	picker = picker.HandleKey("l", 80)
	if got := picker.Selected().Value; got != "soft-white" {
		t.Fatalf("selection after right = %q, want soft-white", got)
	}

	picker = picker.HandleKey("j", 80)
	if got := picker.Selected().Value; got != "orange" {
		t.Fatalf("selection after down = %q, want orange", got)
	}

	picker = picker.HandleKey("]", 80)
	if got := picker.Selected().Value; got != "sky" {
		t.Fatalf("selection after next palette = %q, want sky", got)
	}
}
