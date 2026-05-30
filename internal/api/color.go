package api

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

type XY struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type ColorOptions struct {
	TurnOn     bool
	Brightness int
}

type RGB struct {
	R int
	G int
	B int
}

var namedColors = map[string]RGB{
	"red":     {R: 255, G: 0, B: 0},
	"green":   {R: 0, G: 255, B: 0},
	"blue":    {R: 0, G: 0, B: 255},
	"white":   {R: 255, G: 255, B: 255},
	"warm":    {R: 255, G: 190, B: 120},
	"orange":  {R: 255, G: 136, B: 0},
	"yellow":  {R: 255, G: 255, B: 0},
	"purple":  {R: 128, G: 0, B: 255},
	"pink":    {R: 255, G: 80, B: 180},
	"cyan":    {R: 0, G: 255, B: 255},
	"magenta": {R: 255, G: 0, B: 255},
}

func ParseColor(value string) (XY, error) {
	rgb, err := ParseRGB(value)
	if err != nil {
		return XY{}, err
	}
	return RGBToXY(rgb), nil
}

func ParseRGB(value string) (RGB, error) {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return RGB{}, fmt.Errorf("empty color")
	}
	if rgb, ok := namedColors[value]; ok {
		return rgb, nil
	}
	if strings.HasPrefix(value, "rgb:") {
		return parseCSVColor(strings.TrimPrefix(value, "rgb:"), 255, func(parts []int) RGB {
			return RGB{R: parts[0], G: parts[1], B: parts[2]}
		})
	}
	if strings.HasPrefix(value, "hsv:") {
		return parseCSVColor(strings.TrimPrefix(value, "hsv:"), 360, func(parts []int) RGB {
			return HSVToRGB(parts[0], parts[1], parts[2])
		})
	}
	if strings.HasPrefix(value, "#") {
		value = strings.TrimPrefix(value, "#")
	}
	if len(value) != 6 {
		return RGB{}, fmt.Errorf("invalid color %q: use #rrggbb, rrggbb, a named color, rgb:r,g,b, or hsv:h,s,v", value)
	}
	n, err := strconv.ParseUint(value, 16, 32)
	if err != nil {
		return RGB{}, fmt.Errorf("invalid hex color %q", value)
	}
	return RGB{
		R: int((n >> 16) & 0xff),
		G: int((n >> 8) & 0xff),
		B: int(n & 0xff),
	}, nil
}

func RGBToXY(rgb RGB) XY {
	r := linearize(float64(clamp(rgb.R, 0, 255)) / 255)
	g := linearize(float64(clamp(rgb.G, 0, 255)) / 255)
	b := linearize(float64(clamp(rgb.B, 0, 255)) / 255)

	x := r*0.649926 + g*0.103455 + b*0.197109
	y := r*0.234327 + g*0.743075 + b*0.022598
	z := r*0.000000 + g*0.053077 + b*1.035763
	sum := x + y + z
	if sum == 0 {
		return XY{X: 0.3127, Y: 0.3290}
	}
	return XY{
		X: round4(x / sum),
		Y: round4(y / sum),
	}
}

func HSVToRGB(h, s, v int) RGB {
	h = ((h % 360) + 360) % 360
	sf := float64(clamp(s, 0, 100)) / 100
	vf := float64(clamp(v, 0, 100)) / 100
	c := vf * sf
	x := c * (1 - math.Abs(math.Mod(float64(h)/60, 2)-1))
	m := vf - c

	var r, g, b float64
	switch {
	case h < 60:
		r, g, b = c, x, 0
	case h < 120:
		r, g, b = x, c, 0
	case h < 180:
		r, g, b = 0, c, x
	case h < 240:
		r, g, b = 0, x, c
	case h < 300:
		r, g, b = x, 0, c
	default:
		r, g, b = c, 0, x
	}
	return RGB{
		R: int(math.Round((r + m) * 255)),
		G: int(math.Round((g + m) * 255)),
		B: int(math.Round((b + m) * 255)),
	}
}

func parseCSVColor(value string, max int, build func([]int) RGB) (RGB, error) {
	fields := strings.Split(value, ",")
	if len(fields) != 3 {
		return RGB{}, fmt.Errorf("color requires three comma-separated values")
	}
	parts := make([]int, 3)
	for i, field := range fields {
		n, err := strconv.Atoi(strings.TrimSpace(field))
		if err != nil {
			return RGB{}, fmt.Errorf("invalid color component %q", field)
		}
		parts[i] = n
	}
	if max == 255 {
		for _, n := range parts {
			if n < 0 || n > 255 {
				return RGB{}, fmt.Errorf("rgb components must be 0-255")
			}
		}
	} else if parts[1] < 0 || parts[1] > 100 || parts[2] < 0 || parts[2] > 100 {
		return RGB{}, fmt.Errorf("hsv saturation and value must be 0-100")
	}
	return build(parts), nil
}

func linearize(value float64) float64 {
	if value > 0.04045 {
		return math.Pow((value+0.055)/(1+0.055), 2.4)
	}
	return value / 12.92
}

func round4(value float64) float64 {
	return math.Round(value*10000) / 10000
}

func clamp(value, min, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}
