package main

import "testing"

func TestParseColorOptions(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantTarget string
		wantColor  string
	}{
		{
			name:       "no args opens picker for all",
			args:       nil,
			wantTarget: "all",
		},
		{
			name:       "single known color keeps direct command behavior",
			args:       []string{"sunset"},
			wantTarget: "all",
			wantColor:  "sunset",
		},
		{
			name:       "single unknown value is treated as target for picker",
			args:       []string{"desk"},
			wantTarget: "desk",
		},
		{
			name:       "target and color",
			args:       []string{"desk", "soft-white"},
			wantTarget: "desk",
			wantColor:  "soft-white",
		},
		{
			name:       "hex color",
			args:       []string{"#ff8800"},
			wantTarget: "all",
			wantColor:  "#ff8800",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseColorOptions(tt.args)
			if err != nil {
				t.Fatal(err)
			}
			if got.target != tt.wantTarget {
				t.Fatalf("target = %q, want %q", got.target, tt.wantTarget)
			}
			if got.colorValue != tt.wantColor {
				t.Fatalf("colorValue = %q, want %q", got.colorValue, tt.wantColor)
			}
		})
	}
}

func TestParseColorOptionsFlags(t *testing.T) {
	got, err := parseColorOptions([]string{"desk", "blue", "-b", "180", "--no-on", "--bridge", "192.168.1.2"})
	if err != nil {
		t.Fatal(err)
	}
	if got.target != "desk" || got.colorValue != "blue" {
		t.Fatalf("target/color = %q/%q, want desk/blue", got.target, got.colorValue)
	}
	if !got.brightnessSet || got.brightness != 180 {
		t.Fatalf("brightness = %d set=%v, want 180 true", got.brightness, got.brightnessSet)
	}
	if !got.noOn {
		t.Fatal("noOn = false, want true")
	}
	if got.bridgeIP != "192.168.1.2" {
		t.Fatalf("bridgeIP = %q, want 192.168.1.2", got.bridgeIP)
	}
}
