package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Aayush9029/huectl/internal/api"
	"github.com/Aayush9029/huectl/internal/config"
	"github.com/Aayush9029/huectl/internal/tui"
	"github.com/Aayush9029/huectl/internal/ui"
	tea "github.com/charmbracelet/bubbletea"
)

var version = "dev"

type options struct {
	bridgeIP   string
	target     string
	brightness int
	timeout    time.Duration
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		ui.Fatalf("%s", err)
	}
}

func run(args []string) error {
	if err := config.EnsureDir(); err != nil {
		return err
	}

	if len(args) == 0 {
		return cmdUI(nil)
	}

	switch args[0] {
	case "-h", "--help", "help":
		showHelp()
		return nil
	case "-v", "--version":
		fmt.Printf("huectl %s\n", version)
		return nil
	case "auth":
		return cmdAuth(args[1:])
	case "discover":
		return cmdDiscover()
	case "status":
		return cmdStatus(args[1:])
	case "on":
		return cmdPower(args[1:], true)
	case "off":
		return cmdPower(args[1:], false)
	case "toggle":
		return cmdToggle(args[1:])
	case "ui":
		return cmdUI(args[1:])
	case "config":
		return cmdConfig()
	default:
		return fmt.Errorf("unknown command: %s", args[0])
	}
}

func showHelp() {
	fmt.Println()
	ui.Header("huectl")
	ui.Dimf("control local Philips Hue lights")
	fmt.Println()
	fmt.Println("USAGE")
	fmt.Println("    huectl [command] [target] [options]")
	fmt.Println()
	fmt.Println("COMMANDS")
	fmt.Println("    auth                  Pair with the bridge and save an app key")
	fmt.Println("    discover              Find Hue bridges on the local network")
	fmt.Println("    status                Show lights and current state")
	fmt.Println("    on [target]           Turn lights on")
	fmt.Println("    off [target]          Turn lights off")
	fmt.Println("    toggle [target]       Toggle lights")
	fmt.Println("    ui                    Open the interactive light dashboard")
	fmt.Println("    config                Show config path and bridge IP")
	fmt.Println()
	fmt.Println("OPTIONS")
	fmt.Println("    -b, --brightness N    Brightness for on, 1-254 (default: 254)")
	fmt.Println("    --bridge IP           Use a specific bridge IP")
	fmt.Println("    --timeout SECONDS     Pairing timeout for auth (default: 45)")
	fmt.Println("    -h, --help            Show this help")
	fmt.Println("    -v, --version         Show version")
	fmt.Println()
	fmt.Println("EXAMPLES")
	fmt.Println("    huectl")
	fmt.Println("    huectl status")
	fmt.Println("    huectl on")
	fmt.Println("    huectl on 2 -b 180")
	fmt.Println("    huectl off all")
	fmt.Println("    huectl toggle \"lamp 1\"")
	fmt.Println()
}

func cmdDiscover() error {
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()

	bridges, err := api.Discover(ctx)
	if err != nil {
		return err
	}
	if len(bridges) == 0 {
		return errors.New("no Hue bridges found")
	}

	ui.Header("huectl")
	fmt.Println()
	for _, bridge := range bridges {
		ui.Success(fmt.Sprintf("%s %s", bridge.IP, bridge.ID))
	}
	return nil
}

func cmdAuth(args []string) error {
	opts, err := parseOptions(args)
	if err != nil {
		return err
	}
	if opts.timeout == 0 {
		opts.timeout = 45 * time.Second
	}

	cfg, _, err := config.Load()
	if err != nil {
		return err
	}

	bridgeIP := firstNonEmpty(opts.bridgeIP, cfg.BridgeIP)
	if bridgeIP == "" {
		ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
		defer cancel()
		bridges, err := api.Discover(ctx)
		if err != nil {
			return err
		}
		if len(bridges) == 0 {
			return errors.New("no Hue bridges found")
		}
		bridgeIP = bridges[0].IP
	}

	ui.Header("huectl")
	fmt.Println()
	ui.Status("Using bridge " + bridgeIP)
	ui.Status("Press the bridge link button now")
	fmt.Println()

	deadline := time.Now().Add(opts.timeout)
	var lastErr error
	attempt := 1
	for time.Now().Before(deadline) {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		appKey, err := api.Auth(ctx, bridgeIP)
		cancel()
		if err == nil {
			cfg.BridgeIP = bridgeIP
			cfg.AppKey = appKey
			if err := config.Save(cfg); err != nil {
				return err
			}
			fmt.Print("\r")
			ui.Success("Paired and saved config")
			ui.Dimf(config.Path())
			return nil
		}
		lastErr = err
		if ui.IsTTY() {
			fmt.Printf("\r%s⏳ %s (%ds/%ds)%s", ui.Yellow, err, attempt, int(opts.timeout.Seconds()), ui.Reset)
		} else {
			fmt.Printf("waiting: %s\n", err)
		}
		attempt++
		time.Sleep(time.Second)
	}
	if ui.IsTTY() {
		fmt.Println()
	}
	return fmt.Errorf("pairing timed out: %w", lastErr)
}

func cmdStatus(args []string) error {
	opts, err := parseOptions(args)
	if err != nil {
		return err
	}
	cfg, client, err := configuredClient(opts.bridgeIP)
	if err != nil {
		return err
	}

	lights, err := fetchAndCache(cfg, client)
	if err != nil {
		return err
	}

	ui.Header("huectl")
	ui.Dimf("bridge %s", client.BridgeIP)
	fmt.Println()
	printLights(lights)
	return nil
}

func cmdPower(args []string, on bool) error {
	opts, err := parseOptions(args)
	if err != nil {
		return err
	}
	if opts.brightness == 0 {
		opts.brightness = 254
	}

	cfg, client, err := configuredClient(opts.bridgeIP)
	if err != nil {
		return err
	}
	lights, err := fetchAndCache(cfg, client)
	if err != nil {
		return err
	}
	targets := matchLights(lights, opts.target)
	if len(targets) == 0 {
		return fmt.Errorf("no lights matched target: %s", opts.target)
	}

	ui.Header("huectl")
	fmt.Println()
	for _, light := range targets {
		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		err := client.SetPower(ctx, light.ID, on, opts.brightness)
		cancel()
		if err != nil {
			return err
		}
		if on {
			ui.Success("On: " + light.Name)
		} else {
			ui.Success("Off: " + light.Name)
		}
	}
	lights, _ = fetchAndCache(cfg, client)
	return nil
}

func cmdToggle(args []string) error {
	opts, err := parseOptions(args)
	if err != nil {
		return err
	}
	if opts.brightness == 0 {
		opts.brightness = 254
	}

	cfg, client, err := configuredClient(opts.bridgeIP)
	if err != nil {
		return err
	}
	lights, err := fetchAndCache(cfg, client)
	if err != nil {
		return err
	}
	targets := matchLights(lights, opts.target)
	if len(targets) == 0 {
		return fmt.Errorf("no lights matched target: %s", opts.target)
	}

	ui.Header("huectl")
	fmt.Println()
	for _, light := range targets {
		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		err := client.SetPower(ctx, light.ID, !light.On, clamp(light.Brightness, 1, 254))
		cancel()
		if err != nil {
			return err
		}
		if light.On {
			ui.Success("Off: " + light.Name)
		} else {
			ui.Success("On: " + light.Name)
		}
	}
	lights, _ = fetchAndCache(cfg, client)
	return nil
}

func cmdUI(args []string) error {
	opts, err := parseOptions(args)
	if err != nil {
		return err
	}
	cfg, client, err := configuredClient(opts.bridgeIP)
	if err != nil {
		return err
	}

	saveCache := func(lights []api.Light) {
		next := config.UpdateLights(cfg, lights)
		_ = config.Save(next)
	}
	program := tea.NewProgram(tui.NewModel(client, saveCache), tea.WithAltScreen())
	_, err = program.Run()
	return err
}

func cmdConfig() error {
	cfg, loadedLegacy, err := config.Load()
	if err != nil {
		return err
	}

	ui.Header("huectl")
	fmt.Println()
	ui.Status("Config: " + config.Path())
	if loadedLegacy {
		ui.Status("Legacy config: " + config.LegacyPath())
	}
	if cfg.BridgeIP == "" {
		ui.Status("Bridge: not set")
	} else {
		ui.Status("Bridge: " + cfg.BridgeIP)
	}
	if cfg.AppKey == "" {
		ui.Status("App key: missing")
	} else {
		ui.Status("App key: saved")
	}
	if len(cfg.Lights) > 0 {
		ui.Status(fmt.Sprintf("Cached lights: %d", len(cfg.Lights)))
	}
	return nil
}

func configuredClient(bridgeOverride string) (config.Config, *api.Client, error) {
	cfg, loadedLegacy, err := config.Load()
	if err != nil {
		return config.Config{}, nil, err
	}
	if bridgeOverride != "" {
		cfg.BridgeIP = bridgeOverride
	}
	if cfg.BridgeIP == "" || cfg.AppKey == "" {
		return config.Config{}, nil, errors.New("Hue bridge is not paired; run: huectl auth")
	}
	if loadedLegacy {
		_ = config.Save(cfg)
	}
	return cfg, api.NewClient(cfg.BridgeIP, cfg.AppKey), nil
}

func fetchAndCache(cfg config.Config, client *api.Client) ([]api.Light, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	lights, err := client.Lights(ctx)
	if err != nil {
		return nil, err
	}
	if err := config.Save(config.UpdateLights(cfg, lights)); err != nil {
		return nil, err
	}
	return lights, nil
}

func printLights(lights []api.Light) {
	for _, light := range lights {
		state := "off"
		if light.On {
			state = "on"
		}
		reachable := "reachable"
		if !light.Reachable {
			reachable = "unreachable"
		}
		fmt.Printf("%-3s %-24s %-4s bri=%-3d %-11s %s\n", light.ID, truncate(light.Name, 24), state, light.Brightness, reachable, light.ModelID)
	}
}

func matchLights(lights []api.Light, target string) []api.Light {
	target = strings.TrimSpace(target)
	if target == "" || target == "all" {
		return lights
	}

	var matches []api.Light
	for _, light := range lights {
		if light.ID == target || strings.Contains(strings.ToLower(light.Name), strings.ToLower(target)) {
			matches = append(matches, light)
		}
	}
	return matches
}

func parseOptions(args []string) (options, error) {
	opts := options{
		target:     "all",
		brightness: 0,
	}

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-b", "--brightness":
			if i+1 >= len(args) {
				return opts, errors.New("--brightness requires a value")
			}
			i++
			n, err := strconv.Atoi(args[i])
			if err != nil {
				return opts, fmt.Errorf("invalid brightness: %s", args[i])
			}
			opts.brightness = clamp(n, 1, 254)
		case "--bridge":
			if i+1 >= len(args) {
				return opts, errors.New("--bridge requires an IP address")
			}
			i++
			opts.bridgeIP = args[i]
		case "--timeout":
			if i+1 >= len(args) {
				return opts, errors.New("--timeout requires seconds")
			}
			i++
			n, err := strconv.Atoi(args[i])
			if err != nil {
				return opts, fmt.Errorf("invalid timeout: %s", args[i])
			}
			opts.timeout = time.Duration(n) * time.Second
		default:
			if strings.HasPrefix(args[i], "-") {
				return opts, fmt.Errorf("unknown option: %s", args[i])
			}
			if opts.target != "all" {
				return opts, fmt.Errorf("unexpected argument: %s", args[i])
			}
			opts.target = args[i]
		}
	}
	return opts, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
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

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	if max <= 1 {
		return s[:max]
	}
	return s[:max-1] + "…"
}
