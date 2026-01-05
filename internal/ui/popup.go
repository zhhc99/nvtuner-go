package ui

import (
	"fmt"
	"nvtuner-go/internal/gpu"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	lg "github.com/charmbracelet/lipgloss"
)

type PopupType int

const (
	PopupNone PopupType = iota
	PopupApply
	PopupReset
)

type PopupState struct {
	Type PopupType
}

func (m *Model) popupView(width, height int) string {
	pt := m.popup.Type
	if pt == PopupNone {
		return ""
	}

	d := m.dStates[m.selectedGpu]
	var body strings.Builder
	var title, desc string

	nameStyle := th.Focus.Align(lg.Center).MarginBottom(1)
	keyStyle := lg.NewStyle().Foreground(plt.Dim).Width(6).Align(lg.Right).MarginRight(1)
	descStyle := th.Primary.Italic(true).MarginTop(1)

	if pt == PopupApply {
		title = "APPLY SETTINGS"
		cfg, _ := m.config.Get(d.UUID)
		body.WriteString(nameStyle.Render(fmt.Sprintf("%s (ID: %d)", d.Name, d.Index)) + "\n")
		for _, p := range m.tuningParams {
			val := "  N/A"
			if v := p.GetConfig(cfg); v != gpu.NO_VALUE {
				val = fmt.Sprintf("%5d", v)
			}
			body.WriteString(lg.JoinHorizontal(lg.Left, keyStyle.Render(p.ShortLabel),
				th.Focus.Render(val), th.Value.Render(fmt.Sprintf(" %-5s", p.Unit))) + "\n")
		}
		desc = "Apply this profile?"
	} else {
		title = "RESET TO DEFAULT"
		body.WriteString(nameStyle.Render(fmt.Sprintf("GPU %d: %s", d.Index, d.Name)) + "\n")
		for _, p := range m.tuningParams {
			val := "  N/A"
			if v := p.GetDefault(d); v != gpu.NO_VALUE {
				val = fmt.Sprintf("%5d", v)
			}
			body.WriteString(lg.JoinHorizontal(lg.Left, keyStyle.Render(p.ShortLabel),
				th.Focus.Render(val), th.Value.Render(fmt.Sprintf(" %-5s", p.Unit))) + "\n")
		}
		desc = "Restore to preset defaults?"
	}

	text := body.String() + descStyle.Render(desc) + "\n\n" + th.Value.Render("(Enter to Confirm / q to Cancel)")
	box := RenderBoxWithTitle(title, lg.NewStyle().Align(lg.Center).Padding(1, 2).Render(text))
	return lg.Place(width, height, lg.Center, lg.Center, box,
		lg.WithWhitespaceChars(" "), lg.WithWhitespaceForeground(plt.Dim))
}

func (m *Model) handlePopupConfirm() (tea.Model, tea.Cmd) {
	pt, ds := m.popup.Type, m.dStates[m.selectedGpu]
	m.popup.Type = PopupNone

	errs := make(map[string][]string)
	cfg, _ := m.config.Get(ds.UUID)

	for _, p := range m.tuningParams {
		val := p.GetConfig(cfg)
		if pt == PopupReset {
			val = p.GetDefault(ds)
			p.SetConfig(&cfg, val)
		}
		if err := p.Apply(m.devices[m.selectedGpu], val); err != nil {
			msg := err.Error()
			errs[msg] = append(errs[msg], p.ShortLabel)
		}
	}

	if pt == PopupReset {
		m.config.Set(ds.UUID, cfg)
		m.config.Save()
	}

	if len(errs) == 0 {
		m.statusIsErr, m.statusMsg = false, "Action completed successfully"
		return m, nil
	}

	var msgs []string
	for msg, labels := range errs {
		msgs = append(msgs, fmt.Sprintf("[%s]: %s", strings.Join(labels, ","), msg))
	}
	m.statusIsErr, m.statusMsg = true, strings.Join(msgs, " | ")
	return m, nil
}
