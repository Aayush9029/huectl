package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"
)

const discoveryURL = "https://discovery.meethue.com/"

type Client struct {
	BridgeIP   string
	AppKey     string
	HTTPClient *http.Client
}

func NewClient(bridgeIP, appKey string) *Client {
	return &Client{
		BridgeIP: bridgeIP,
		AppKey:   appKey,
		HTTPClient: &http.Client{
			Timeout: 6 * time.Second,
		},
	}
}

func Discover(ctx context.Context) ([]Bridge, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, discoveryURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("discovery failed: %s", resp.Status)
	}

	var raw []struct {
		ID string `json:"id"`
		IP string `json:"internalipaddress"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}

	bridges := make([]Bridge, 0, len(raw))
	for _, b := range raw {
		if b.IP == "" {
			continue
		}
		bridges = append(bridges, Bridge{ID: b.ID, IP: b.IP})
	}
	return bridges, nil
}

func Auth(ctx context.Context, bridgeIP string) (string, error) {
	body := []byte(`{"devicetype":"huectl#mac"}`)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://"+bridgeIP+"/api", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var parsed []map[string]map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		return "", fmt.Errorf("unexpected auth response: %s", strings.TrimSpace(string(data)))
	}
	if len(parsed) == 0 {
		return "", fmt.Errorf("empty auth response")
	}
	if success, ok := parsed[0]["success"]; ok {
		if username, ok := success["username"].(string); ok && username != "" {
			return username, nil
		}
	}
	if apiErr, ok := parsed[0]["error"]; ok {
		if desc, ok := apiErr["description"].(string); ok && desc != "" {
			return "", errors.New(desc)
		}
	}
	return "", fmt.Errorf("pairing failed")
}

func (c *Client) Lights(ctx context.Context) ([]Light, error) {
	var raw map[string]rawLight
	if err := c.get(ctx, "lights", &raw); err != nil {
		return nil, err
	}

	lights := make([]Light, 0, len(raw))
	for id, light := range raw {
		lights = append(lights, Light{
			ID:         id,
			Name:       light.Name,
			Type:       light.Type,
			ModelID:    light.ModelID,
			On:         light.State.On,
			Brightness: light.State.Bri,
			Reachable:  light.State.Reachable,
			ColorMode:  light.State.ColorMode,
		})
	}
	sort.Slice(lights, func(i, j int) bool {
		return naturalLess(lights[i].ID, lights[j].ID)
	})
	return lights, nil
}

func (c *Client) SetPower(ctx context.Context, id string, on bool, brightness int) error {
	body := map[string]any{"on": on}
	if on {
		body["bri"] = brightness
	}
	return c.putState(ctx, id, body)
}

func (c *Client) SetBrightness(ctx context.Context, id string, brightness int) error {
	return c.putState(ctx, id, map[string]any{
		"on":  true,
		"bri": brightness,
	})
}

func (c *Client) putState(ctx context.Context, id string, body map[string]any) error {
	var parsed []map[string]map[string]any
	if err := c.put(ctx, "lights/"+id+"/state", body, &parsed); err != nil {
		return err
	}
	for _, item := range parsed {
		if apiErr, ok := item["error"]; ok {
			if desc, ok := apiErr["description"].(string); ok && desc != "" {
				return errors.New(desc)
			}
			return fmt.Errorf("bridge returned an error")
		}
	}
	return nil
}

func (c *Client) get(ctx context.Context, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url(path), nil)
	if err != nil {
		return err
	}
	return c.do(req, out)
}

func (c *Client) put(ctx context.Context, path string, body any, out any) error {
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, c.url(path), bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	return c.do(req, out)
}

func (c *Client) do(req *http.Request, out any) error {
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 300 {
		return fmt.Errorf("bridge request failed: %s", resp.Status)
	}
	if err := json.Unmarshal(data, out); err != nil {
		return fmt.Errorf("unexpected bridge response: %s", strings.TrimSpace(string(data)))
	}
	return nil
}

func (c *Client) url(path string) string {
	return "http://" + c.BridgeIP + "/api/" + c.AppKey + "/" + strings.TrimPrefix(path, "/")
}

func naturalLess(a, b string) bool {
	if len(a) != len(b) {
		return len(a) < len(b)
	}
	return a < b
}
