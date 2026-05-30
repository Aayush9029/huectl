package tui

import (
	"testing"

	"github.com/Aayush9029/huectl/internal/api"
)

func TestOpenColorPickerTargetsAllColorLights(t *testing.T) {
	model := Model{
		lights: []api.Light{
			{ID: "1", Name: "Desk", HasColor: true},
			{ID: "2", Name: "Lamp", HasColor: true},
			{ID: "3", Name: "Hall", HasColor: false},
		},
	}

	next, cmd := model.openColorPicker()
	if cmd != nil {
		t.Fatal("openColorPicker returned a command")
	}

	got := next.(Model)
	if got.mode != modeColorPicker {
		t.Fatalf("mode = %v, want modeColorPicker", got.mode)
	}
	if len(got.colorTargets) != 2 {
		t.Fatalf("colorTargets length = %d, want 2", len(got.colorTargets))
	}
	if got.colorTargets[0].Name != "Desk" || got.colorTargets[1].Name != "Lamp" {
		t.Fatalf("colorTargets = %#v, want Desk and Lamp", got.colorTargets)
	}
}
