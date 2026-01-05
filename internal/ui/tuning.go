package ui

import (
	"fmt"
	"nvtuner-go/internal/config"
	"nvtuner-go/internal/gpu"
	"strings"

	lg "github.com/charmbracelet/lipgloss"
)

type TuningParam struct {
	ID         string
	Label      string // like "POWER LIMIT:"
	ShortLabel string // like "PL"
	Unit       string // like "W"

	GetConfig  func(c config.GpuSettings) int
	GetCurrent func(d gpu.DState) int
	GetLimits  func(d gpu.DState) (int, int) // min, max
	GetDefault func(d gpu.DState) int

	SetConfig func(c *config.GpuSettings, val int)
	Apply     func(d gpu.Device, val int) error
}

func getDefaultTuningParams() []TuningParam {
	return []TuningParam{
		{
			ID: "pl", Label: "POWER LIMIT:", ShortLabel: "PL", Unit: "W",
			GetConfig:  func(c config.GpuSettings) int { return c.PowerLimit },
			GetCurrent: func(d gpu.DState) int { return d.PowerLim },
			GetLimits:  func(d gpu.DState) (int, int) { return d.Limits.PlMin, d.Limits.PlMax },
			GetDefault: func(d gpu.DState) int { return d.Defaults.Pl },
			SetConfig:  func(c *config.GpuSettings, v int) { c.PowerLimit = v },
			Apply:      func(d gpu.Device, v int) error { return d.SetPl(v) },
		},
		{
			ID: "gpu_co", Label: "GPU CO:     ", ShortLabel: "G.CO", Unit: "MHz",
			GetConfig:  func(c config.GpuSettings) int { return c.GpuCO },
			GetCurrent: func(d gpu.DState) int { return d.CoGpu },
			GetLimits:  func(d gpu.DState) (int, int) { return d.Limits.CoGpuMin, d.Limits.CoGpuMax },
			GetDefault: func(d gpu.DState) int { return d.Defaults.CoGpu },
			SetConfig:  func(c *config.GpuSettings, v int) { c.GpuCO = v },
			Apply:      func(d gpu.Device, v int) error { return d.SetCoGpu(v) },
		},
		{
			ID: "mem_co", Label: "MEMORY CO:  ", ShortLabel: "M.CO", Unit: "MHz",
			GetConfig:  func(c config.GpuSettings) int { return c.MemCO },
			GetCurrent: func(d gpu.DState) int { return d.CoMem },
			GetLimits:  func(d gpu.DState) (int, int) { return d.Limits.CoMemMin, d.Limits.CoMemMax },
			GetDefault: func(d gpu.DState) int { return d.Defaults.CoMem },
			SetConfig:  func(c *config.GpuSettings, v int) { c.MemCO = v },
			Apply:      func(d gpu.Device, v int) error { return d.SetCoMem(v) },
		},
		{
			ID: "gpu_cl", Label: "GPU LIMIT:  ", ShortLabel: "G.CL", Unit: "MHz",
			GetConfig:  func(c config.GpuSettings) int { return c.GpuCL },
			GetCurrent: func(d gpu.DState) int { return d.ClGpu },
			GetLimits:  func(d gpu.DState) (int, int) { return d.Limits.ClGpuMin, d.Limits.ClGpuMax },
			GetDefault: func(d gpu.DState) int { return d.Defaults.ClGpu },
			SetConfig:  func(c *config.GpuSettings, v int) { c.GpuCL = v },
			Apply: func(d gpu.Device, v int) error {
				_, maxCl, _ := d.GetClLimGpu()
				if v == gpu.NO_VALUE || v < 0 || v >= maxCl {
					return d.ResetClGpu()
				}
				return d.SetClGpu(v)
			},
		},
	}
}

func (m *Model) tuningView(width int) string {
	d := &m.dStates[m.selectedGpu]
	cfg, _ := m.config.Get(d.UUID)

	var rows []string
	cw := max(0, width-2)

	// header
	rows = append(rows, th.PrimaryBold.Width(cw).Render(fmt.Sprintf("%2d: %s", d.Index, d.Name)))

	// tuning params
	// Level 1: Short Label + Range: "PL:   [ 250  ] W (100-450)"
	// Level 2: Short Label + Gauge: "PL:   [ 250  ] W 100 [■■□□□] 450"
	// Level 3: Full Label  + Gauge: "POWER LIMIT: [ 250  ] W 100 [■■□□□] 450"
	for i, p := range m.tuningParams {
		sel := m.tuningIndex == i

		// prefix: cursor, label, config input, current value
		var cursorView string
		var labelView string
		var inputView string
		var valView string
		if sel {
			cursorView = th.Focus.Render("> ")
		} else {
			cursorView = "  "
		}
		if width <= THIN {
			labelView = fmt.Sprintf("%-4s", p.ShortLabel)
		} else {
			labelView = p.Label
		}
		if sel && m.isEditing {
			inputView = fmt.Sprintf("[%s]", m.tuningInput.View())
		} else {
			inputView = fmt.Sprintf("[ %-5d]", p.GetConfig(cfg))
		}
		currVal := p.GetCurrent(*d)
		if currVal == gpu.NO_VALUE {
			valView = th.Disabled.Render("  N/A")
		} else {
			valView = fmt.Sprintf("%5d", currVal)
		}
		prefixView := lg.JoinHorizontal(lg.Left, cursorView, labelView, " ", inputView, " ", valView)

		// suffix: range / gauge
		var suffixText string
		minVal, maxVal := p.GetLimits(*d)
		rngText := fmt.Sprintf("%5d | %-5d", minVal, maxVal)

		wRemain := cw - lg.Width(prefixView) - 1 - 1 // spaces in front & back
		switch {
		case wRemain < lg.Width(rngText):
			suffixText = ""
		case wRemain < len("-9999 [■□□] 9999"):
			suffixText = rngText
		default:
			barW := wRemain - lg.Width("-9999 [] 9999")
			pct := float64(currVal-minVal) / max(1.0, float64(maxVal-minVal))
			used := max(0, int(pct*float64(barW)))
			bar := strings.Repeat("■", used) + strings.Repeat("□", max(0, barW-used))
			suffixText = fmt.Sprintf("%5d [%s] %-5d", minVal, bar, maxVal)
		}
		suffixView := th.Disabled.Render(suffixText)

		// row
		rowView := lg.NewStyle().Width(cw).MaxHeight(1).
			Render(lg.JoinHorizontal(lg.Left, prefixView, " ", suffixView))
		rows = append(rows, rowView)
	}

	if m.statusMsg != "" {
		rows = append(rows, th.Focus.MaxWidth(cw).MaxHeight(1).Render(" "+m.statusMsg))
	} else {
		rows = append(rows, lg.NewStyle().Width(cw).Render("")) // Bottom padding
	}

	return RenderBoxWithTitle("TUNING", lg.JoinVertical(lg.Left, rows...))
}
