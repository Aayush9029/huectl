package tui

import (
	"fmt"
	"strings"

	"github.com/Aayush9029/huectl/internal/api"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type ColorChoice struct {
	Name  string
	Value string
	RGB   api.RGB
}

func (c ColorChoice) Hex() string {
	return fmt.Sprintf("#%02x%02x%02x", c.RGB.R, c.RGB.G, c.RGB.B)
}

type colorPalette struct {
	Name    string
	Choices []ColorChoice
}

type colorPicker struct {
	title  string
	target string
	page   int
	cursor int
}

type ColorPickerModel struct {
	picker    colorPicker
	width     int
	selected  ColorChoice
	submitted bool
	cancelled bool
}

var (
	pickerTabStyle       = lipgloss.NewStyle().Padding(0, 1).Foreground(lipgloss.Color("8"))
	pickerActiveTabStyle = lipgloss.NewStyle().Padding(0, 1).Foreground(lipgloss.Color("0")).Background(lipgloss.Color("6")).Bold(true)
)

var colorPalettes = []colorPalette{
	palette("Essentials", []string{
		"warm",
		"soft-white",
		"white",
		"daylight",
		"red",
		"orange",
		"yellow",
		"green",
		"cyan",
		"blue",
		"purple",
		"pink",
	}),
	palette("Mood", []string{
		"candle",
		"sunset",
		"peach",
		"rose",
		"lavender",
		"sky",
		"ocean",
		"forest",
		"mint",
		"ice",
		"night",
		"amber",
	}),
	palette("Seasons", []string{
		"spring",
		"blossom",
		"meadow",
		"summer",
		"golden",
		"coral",
		"autumn",
		"copper",
		"wine",
		"winter",
		"arctic",
		"frost",
	}),
}

func NewColorPickerModel(target string) ColorPickerModel {
	return ColorPickerModel{
		picker: newColorPicker("choose a color", target),
	}
}

func (m ColorPickerModel) Init() tea.Cmd {
	return nil
}

func (m ColorPickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			m.cancelled = true
			return m, tea.Quit
		case " ", "enter":
			m.selected = m.picker.Selected()
			m.submitted = true
			return m, tea.Quit
		default:
			m.picker = m.picker.HandleKey(msg.String(), m.width)
			return m, nil
		}
	}
	return m, nil
}

func (m ColorPickerModel) View() string {
	return m.picker.View(m.width, "h/j/k/l or arrows move  tab/[ ] palette  enter apply  esc cancel")
}

func (m ColorPickerModel) Selection() (ColorChoice, bool) {
	if !m.submitted || m.cancelled {
		return ColorChoice{}, false
	}
	return m.selected, true
}

func newColorPicker(title, target string) colorPicker {
	return colorPicker{
		title:  title,
		target: target,
	}
}

func (p colorPicker) HandleKey(key string, width int) colorPicker {
	switch key {
	case "left", "h":
		return p.moveHorizontal(-1)
	case "right", "l":
		return p.moveHorizontal(1)
	case "up", "k":
		return p.moveVertical(-1, width)
	case "down", "j":
		return p.moveVertical(1, width)
	case "tab", "]":
		return p.movePage(1)
	case "shift+tab", "[":
		return p.movePage(-1)
	case "home":
		p.cursor = 0
		return p
	case "end":
		p.cursor = len(p.current().Choices) - 1
		return p
	}
	return p
}

func (p colorPicker) Selected() ColorChoice {
	choices := p.current().Choices
	if len(choices) == 0 {
		return ColorChoice{}
	}
	return choices[clamp(p.cursor, 0, len(choices)-1)]
}

func (p colorPicker) View(width int, help string) string {
	if width <= 0 {
		width = 80
	}

	var b strings.Builder
	b.WriteString(titleStyle.Render(p.title))
	if p.target != "" {
		b.WriteString(dimStyle.Render("  " + p.target))
	}
	b.WriteString("\n\n")
	b.WriteString(p.renderTabs())
	b.WriteString("\n\n")

	choices := p.current().Choices
	columns := p.columns(width)
	rows := (len(choices) + columns - 1) / columns
	for row := 0; row < rows; row++ {
		var cells []string
		for col := 0; col < columns; col++ {
			index := row*columns + col
			if index >= len(choices) {
				continue
			}
			cells = append(cells, p.renderChoice(choices[index], index == p.cursor, width, columns))
		}
		b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, cells...))
		b.WriteString("\n")
	}

	selected := p.Selected()
	b.WriteString("\n")
	b.WriteString(dimStyle.Render(fmt.Sprintf("selected %s (%s)", selected.Name, selected.Value)))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render(help))
	b.WriteString("\n")
	return b.String()
}

func (p colorPicker) renderTabs() string {
	tabs := make([]string, 0, len(colorPalettes))
	for i, palette := range colorPalettes {
		style := pickerTabStyle
		if i == p.page {
			style = pickerActiveTabStyle
		}
		tabs = append(tabs, style.Render(palette.Name))
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, tabs...)
}

func (p colorPicker) renderChoice(choice ColorChoice, selected bool, width, columns int) string {
	cellWidth := 16
	if columns == 1 {
		cellWidth = clamp(width-4, 16, 28)
	}

	label := choice.Name
	if selected {
		label = "> " + label
	}
	return lipgloss.NewStyle().
		Width(cellWidth).
		MarginRight(1).
		Align(lipgloss.Center).
		Bold(selected).
		Foreground(contrastColor(choice.RGB)).
		Background(lipgloss.Color(choice.Hex())).
		Render(truncate(label, cellWidth-1))
}

func (p colorPicker) current() colorPalette {
	if len(colorPalettes) == 0 {
		return colorPalette{}
	}
	return colorPalettes[clamp(p.page, 0, len(colorPalettes)-1)]
}

func (p colorPicker) columns(width int) int {
	if width <= 0 {
		width = 80
	}
	if width < 40 {
		return 1
	}
	if width < 76 {
		return 2
	}
	return 4
}

func (p colorPicker) moveHorizontal(delta int) colorPicker {
	choices := p.current().Choices
	if len(choices) == 0 {
		return p
	}
	p.cursor = clamp(p.cursor+delta, 0, len(choices)-1)
	return p
}

func (p colorPicker) moveVertical(delta int, width int) colorPicker {
	choices := p.current().Choices
	if len(choices) == 0 {
		return p
	}
	columns := p.columns(width)
	next := p.cursor + delta*columns
	p.cursor = clamp(next, 0, len(choices)-1)
	return p
}

func (p colorPicker) movePage(delta int) colorPicker {
	if len(colorPalettes) == 0 {
		return p
	}
	p.page = (p.page + delta + len(colorPalettes)) % len(colorPalettes)
	p.cursor = clamp(p.cursor, 0, len(p.current().Choices)-1)
	return p
}

func palette(name string, values []string) colorPalette {
	choices := make([]ColorChoice, 0, len(values))
	for _, value := range values {
		rgb, err := api.ParseRGB(value)
		if err != nil {
			continue
		}
		choices = append(choices, ColorChoice{
			Name:  displayColorName(value),
			Value: value,
			RGB:   rgb,
		})
	}
	return colorPalette{Name: name, Choices: choices}
}

func displayColorName(value string) string {
	parts := strings.FieldsFunc(value, func(r rune) bool {
		return r == '-' || r == '_'
	})
	for i, part := range parts {
		if part == "" {
			continue
		}
		parts[i] = strings.ToUpper(part[:1]) + part[1:]
	}
	return strings.Join(parts, " ")
}

func contrastColor(rgb api.RGB) lipgloss.Color {
	luminance := 0.299*float64(rgb.R) + 0.587*float64(rgb.G) + 0.114*float64(rgb.B)
	if luminance > 150 {
		return lipgloss.Color("0")
	}
	return lipgloss.Color("15")
}
