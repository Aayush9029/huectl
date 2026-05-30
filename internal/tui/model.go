package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Aayush9029/huectl/internal/api"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type SaveCacheFunc func([]api.Light)

type Model struct {
	client    *api.Client
	saveCache SaveCacheFunc
	lights    []api.Light
	cursor    int
	width     int
	height    int
	loading   bool
	message   string
	err       error
}

type lightsMsg []api.Light
type errMsg struct{ err error }
type actionMsg string
type colorPreset struct {
	key   string
	name  string
	value string
}

var (
	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))
	dimStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	onStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true)
	offStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	errStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	rowStyle   = lipgloss.NewStyle().PaddingLeft(1)
	activeRow  = lipgloss.NewStyle().Foreground(lipgloss.Color("0")).Background(lipgloss.Color("6")).PaddingLeft(1).PaddingRight(1)
	presets    = []colorPreset{
		{key: "1", name: "warm", value: "warm"},
		{key: "2", name: "white", value: "white"},
		{key: "3", name: "red", value: "red"},
		{key: "4", name: "orange", value: "orange"},
		{key: "5", name: "blue", value: "blue"},
		{key: "6", name: "purple", value: "purple"},
	}
)

func NewModel(client *api.Client, saveCache SaveCacheFunc) Model {
	return Model{
		client:    client,
		saveCache: saveCache,
		loading:   true,
	}
}

func (m Model) Init() tea.Cmd {
	return m.refreshCmd()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
			return m, nil
		case "down", "j":
			if m.cursor < len(m.lights)-1 {
				m.cursor++
			}
			return m, nil
		case "r":
			m.loading = true
			m.message = "refreshing"
			return m, m.refreshCmd()
		case " ", "enter":
			return m, m.toggleSelectedCmd()
		case "o":
			return m, m.powerSelectedCmd(true)
		case "f":
			return m, m.powerSelectedCmd(false)
		case "a":
			return m, m.toggleAllCmd()
		case "+", "=":
			return m, m.adjustBrightnessCmd(25)
		case "-", "_":
			return m, m.adjustBrightnessCmd(-25)
		case "1", "2", "3", "4", "5", "6":
			return m, m.colorPresetCmd(msg.String())
		}
	case lightsMsg:
		m.loading = false
		m.err = nil
		m.lights = []api.Light(msg)
		if m.cursor >= len(m.lights) {
			m.cursor = len(m.lights) - 1
		}
		if m.cursor < 0 {
			m.cursor = 0
		}
		if m.saveCache != nil {
			m.saveCache(m.lights)
		}
		if m.message == "" {
			m.message = "ready"
		}
		return m, nil
	case errMsg:
		m.loading = false
		m.err = msg.err
		return m, nil
	case actionMsg:
		m.message = string(msg)
		m.loading = true
		return m, m.refreshCmd()
	}
	return m, nil
}

func (m Model) View() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("huectl"))
	if m.client.BridgeIP != "" {
		b.WriteString(dimStyle.Render("  bridge " + m.client.BridgeIP))
	}
	b.WriteString("\n\n")

	if m.err != nil {
		b.WriteString(errStyle.Render(m.err.Error()))
		b.WriteString("\n\n")
	}

	if m.loading && len(m.lights) == 0 {
		b.WriteString(dimStyle.Render("loading lights..."))
		b.WriteString("\n")
	} else if len(m.lights) == 0 {
		b.WriteString(dimStyle.Render("no lights found"))
		b.WriteString("\n")
	} else {
		for i, light := range m.lights {
			b.WriteString(m.renderRow(i, light))
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")
	if m.loading {
		b.WriteString(dimStyle.Render("working..."))
	} else {
		b.WriteString(dimStyle.Render(m.message))
	}
	b.WriteString("\n")
	b.WriteString(dimStyle.Render("j/k select  space toggle  o on  f off  +/- brightness  1-6 color  a all  r refresh  q quit"))
	b.WriteString("\n")
	return b.String()
}

func (m Model) renderRow(index int, light api.Light) string {
	state := offStyle.Render("off")
	if light.On {
		state = onStyle.Render("on ")
	}
	reachable := "reachable"
	if !light.Reachable {
		reachable = "unreachable"
	}
	color := "white-only"
	if light.HasColor {
		color = fmt.Sprintf("xy=%.3f,%.3f", light.XY.X, light.XY.Y)
	}
	line := fmt.Sprintf("%-2s %-24s %s  bri=%-3d  %-11s  %s", light.ID, truncate(light.Name, 24), state, light.Brightness, reachable, color)
	if index == m.cursor {
		return activeRow.Render(line)
	}
	return rowStyle.Render(line)
}

func (m Model) refreshCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()
		lights, err := m.client.Lights(ctx)
		if err != nil {
			return errMsg{err: err}
		}
		return lightsMsg(lights)
	}
}

func (m Model) toggleSelectedCmd() tea.Cmd {
	if len(m.lights) == 0 {
		return nil
	}
	light := m.lights[m.cursor]
	return m.powerCmd(light.ID, !light.On, light.Brightness, "toggled "+light.Name)
}

func (m Model) powerSelectedCmd(on bool) tea.Cmd {
	if len(m.lights) == 0 {
		return nil
	}
	light := m.lights[m.cursor]
	return m.powerCmd(light.ID, on, light.Brightness, stateVerb(on)+" "+light.Name)
}

func (m Model) toggleAllCmd() tea.Cmd {
	lights := append([]api.Light(nil), m.lights...)
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
		defer cancel()
		anyOff := false
		for _, light := range lights {
			if !light.On {
				anyOff = true
				break
			}
		}
		for _, light := range lights {
			if err := m.client.SetPower(ctx, light.ID, anyOff, clamp(light.Brightness, 1, 254)); err != nil {
				return errMsg{err: err}
			}
		}
		if anyOff {
			return actionMsg("turned all on")
		}
		return actionMsg("turned all off")
	}
}

func (m Model) adjustBrightnessCmd(delta int) tea.Cmd {
	if len(m.lights) == 0 {
		return nil
	}
	light := m.lights[m.cursor]
	next := clamp(light.Brightness+delta, 1, 254)
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()
		if err := m.client.SetBrightness(ctx, light.ID, next); err != nil {
			return errMsg{err: err}
		}
		return actionMsg(fmt.Sprintf("%s brightness %d", light.Name, next))
	}
}

func (m Model) colorPresetCmd(key string) tea.Cmd {
	if len(m.lights) == 0 {
		return nil
	}
	light := m.lights[m.cursor]
	if !light.HasColor {
		m.message = light.Name + " does not support color"
		return nil
	}
	var selected colorPreset
	for _, preset := range presets {
		if preset.key == key {
			selected = preset
			break
		}
	}
	if selected.key == "" {
		return nil
	}
	return func() tea.Msg {
		xy, err := api.ParseColor(selected.value)
		if err != nil {
			return errMsg{err: err}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()
		if err := m.client.SetColor(ctx, light.ID, xy, api.ColorOptions{TurnOn: true}); err != nil {
			return errMsg{err: err}
		}
		return actionMsg(fmt.Sprintf("%s color %s", light.Name, selected.name))
	}
}

func (m Model) powerCmd(id string, on bool, brightness int, message string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()
		if err := m.client.SetPower(ctx, id, on, clamp(brightness, 1, 254)); err != nil {
			return errMsg{err: err}
		}
		return actionMsg(message)
	}
}

func stateVerb(on bool) string {
	if on {
		return "turned on"
	}
	return "turned off"
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
