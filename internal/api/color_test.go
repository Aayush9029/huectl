package api

import "testing"

func TestParseRGB(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want RGB
	}{
		{name: "hex with hash", in: "#ff8800", want: RGB{R: 255, G: 136, B: 0}},
		{name: "hex without hash", in: "ff8800", want: RGB{R: 255, G: 136, B: 0}},
		{name: "named", in: "blue", want: RGB{R: 0, G: 0, B: 255}},
		{name: "named mood", in: "sunset", want: RGB{R: 255, G: 94, B: 58}},
		{name: "named multi word", in: "soft-white", want: RGB{R: 255, G: 220, B: 170}},
		{name: "rgb", in: "rgb:1,2,3", want: RGB{R: 1, G: 2, B: 3}},
		{name: "hsv", in: "hsv:30,100,100", want: RGB{R: 255, G: 128, B: 0}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseRGB(tt.in)
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Fatalf("ParseRGB(%q) = %#v, want %#v", tt.in, got, tt.want)
			}
		})
	}
}

func TestParseColor(t *testing.T) {
	xy, err := ParseColor("#ff0000")
	if err != nil {
		t.Fatal(err)
	}
	if xy.X < 0.73 || xy.X > 0.74 || xy.Y < 0.26 || xy.Y > 0.27 {
		t.Fatalf("red xy = %#v, want approximately 0.7350,0.2650", xy)
	}
}
