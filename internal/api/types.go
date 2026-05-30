package api

import "time"

type Bridge struct {
	ID string
	IP string
}

type Light struct {
	ID         string
	Name       string
	Type       string
	ModelID    string
	On         bool
	Brightness int
	Reachable  bool
	ColorMode  string
	XY         XY
	HasColor   bool
}

type CachedLight struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	Type       string    `json:"type,omitempty"`
	ModelID    string    `json:"model_id,omitempty"`
	On         bool      `json:"on"`
	Brightness int       `json:"brightness"`
	Reachable  bool      `json:"reachable"`
	ColorMode  string    `json:"color_mode,omitempty"`
	XY         XY        `json:"xy,omitempty"`
	HasColor   bool      `json:"has_color,omitempty"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type rawLight struct {
	State struct {
		On        bool      `json:"on"`
		Bri       int       `json:"bri"`
		Reachable bool      `json:"reachable"`
		ColorMode string    `json:"colormode"`
		XY        []float64 `json:"xy"`
	} `json:"state"`
	Type    string `json:"type"`
	Name    string `json:"name"`
	ModelID string `json:"modelid"`
}

func (l Light) CacheEntry(now time.Time) CachedLight {
	return CachedLight{
		ID:         l.ID,
		Name:       l.Name,
		Type:       l.Type,
		ModelID:    l.ModelID,
		On:         l.On,
		Brightness: l.Brightness,
		Reachable:  l.Reachable,
		ColorMode:  l.ColorMode,
		XY:         l.XY,
		HasColor:   l.HasColor,
		UpdatedAt:  now,
	}
}
