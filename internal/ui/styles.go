package ui

import "github.com/charmbracelet/lipgloss"

const (
	THIN = 80
	WIDE = 140
)

type Palette struct {
	Border lipgloss.AdaptiveColor
	Dim    lipgloss.AdaptiveColor
	Major  lipgloss.AdaptiveColor
	Hyper  lipgloss.AdaptiveColor

	Success lipgloss.Color
	Warning lipgloss.Color
	Error   lipgloss.Color
}

type Theme struct {
	PrimaryBorder lipgloss.Style
	Primary       lipgloss.Style
	PrimaryBold   lipgloss.Style
	Focus         lipgloss.Style
	Label         lipgloss.Style
	Value         lipgloss.Style
	Disabled      lipgloss.Style
}

var plt = Palette{
	Border: lipgloss.AdaptiveColor{Light: "69", Dark: "69"},
	Dim:    lipgloss.AdaptiveColor{Light: "244", Dark: "242"},
	Major:  lipgloss.AdaptiveColor{Light: "86", Dark: "86"},
	Hyper:  lipgloss.AdaptiveColor{Light: "203", Dark: "203"},

	Success: lipgloss.Color("42"),
	Warning: lipgloss.Color("196"),
	Error:   lipgloss.Color("208"),
}

var th = Theme{
	PrimaryBorder: lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(plt.Border),
	Primary: lipgloss.NewStyle().
		Foreground(plt.Border),
	PrimaryBold: lipgloss.NewStyle().
		Foreground(plt.Border).
		Bold(true),
	Focus: lipgloss.NewStyle().
		Foreground(plt.Hyper).
		Bold(true),
	Label: lipgloss.NewStyle(),
	Value: lipgloss.NewStyle().
		Foreground(plt.Dim).
		Bold(true),
	Disabled: lipgloss.NewStyle().
		Foreground(plt.Dim),
}

func (tc *Palette) at(val int) lipgloss.Color {
	if val < 60 {
		return tc.Success
	} else if val < 80 {
		return tc.Warning
	}
	return tc.Error
}
