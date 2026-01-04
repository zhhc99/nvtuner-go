package ui

import (
	"fmt"
	"nvtuner-go/internal/gpu"
	"strings"

	lg "github.com/charmbracelet/lipgloss"
)

type PopupType int

const (
	PopupNone PopupType = iota
	PopupApply
)

type PopupState struct {
	Type    PopupType
	Title   string
	Message string
}

func (m *Model) popupView(width, height int) string {
	if m.popup.Type == PopupNone {
		return ""
	}

	contentStyle := lg.NewStyle().
		Align(lg.Center).
		Padding(1, 2)

	text := m.popup.Message + "\n\n" + th.Value.Render("(Enter to Confirm / Esc to Cancel)")

	box := RenderBoxWithTitle(m.popup.Title, contentStyle.Render(text))

	return lg.Place(width, height, lg.Center, lg.Center, box,
		lg.WithWhitespaceChars(" "), lg.WithWhitespaceForeground(plt.Dim))
}

func (m *Model) makeApplyPopup() PopupState {
	d := m.dStates[m.selectedGpu]
	cfg, _ := m.config.Get(d.UUID)

	var body strings.Builder

	nameStyle := lg.NewStyle().Foreground(plt.Major).Bold(true).Align(lg.Center).MarginBottom(1)
	keyStyle := lg.NewStyle().Foreground(plt.Dim).Width(6).Align(lg.Right).MarginRight(1)
	valStyle := lg.NewStyle().Foreground(plt.Hyper).Bold(true)

	body.WriteString(nameStyle.Render(fmt.Sprintf("%s (ID: %d)", d.Name, d.Index)) + "\n")
	for _, p := range m.tuningParams {
		val := "  N/A"
		if p.GetConfig(cfg) != gpu.NO_VALUE {
			val = fmt.Sprintf("%5d", p.GetConfig(cfg))
		}
		line := lg.JoinHorizontal(lg.Left,
			keyStyle.Render(p.ShortLabel),
			valStyle.Render(val),
		)
		body.WriteString(line + "\n")
	}

	return PopupState{
		Type:    PopupApply,
		Title:   "APPLY SETTINGS",
		Message: body.String(),
	}
}
