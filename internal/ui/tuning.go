package ui

import (
	"fmt"
	"strings"
	"time"

	lg "github.com/charmbracelet/lipgloss"
)

func (m *Model) tuningView(width int) string {
	d := &m.dStates[m.selectedGpu]

	type item struct {
		sLbl, lLbl, unit string
		val, min, max    int
	}
	items := []item{
		{"PL:  ", "POWER LIMIT:", "W  ", d.PowerLim, d.Limits.PlMin, d.Limits.PlMax},
		{"G.CO:", "GPU CO:     ", "MHz", d.CoGpu, d.Limits.CoGpuMin, d.Limits.CoGpuMax},
		{"M.CO:", "MEMORY CO:  ", "MHz", d.CoMem, d.Limits.CoMemMin, d.Limits.CoMemMax},
		{"G.CL:", "GPU LIMIT:  ", "MHz", d.ClGpu, d.Limits.ClGpuMin, d.Limits.ClGpuMax},
	}

	var rows []string
	cw := max(0, width-2)                                   // Content width inside box
	rows = append(rows, lg.NewStyle().Width(cw).Render("")) // Top padding

	// Level 1: Short Label + Range: "PL:   [ 250  ] W (100-450)"
	// Level 2: Short Label + Gauge: "PL:   [ 250  ] W 100 [■■□□□] 450"
	// Level 3: Full Label  + Gauge: "POWER LIMIT: [ 250  ] W 100 [■■□□□] 450"
	now := time.Now()
	for i, it := range items {
		sel := m.tuningIndex == i

		// prefix: cursor, label, value
		curView := "  "
		if sel {
			curView = lg.NewStyle().Foreground(plt.Hyper).Bold(true).Render("> ")
		}
		var label string
		if width > THIN {
			label = it.lLbl
		} else {
			label = it.sLbl
		}
		valView := fmt.Sprintf("[ %-5d]", it.val) // Fixed width 8
		if now.Before(m.tuningUpdateUntil[i]) {
			valView = fmt.Sprintf("[  %s  ]", m.spinner.View()) // loading spinner
		}
		if sel {
			if m.isEditing {
				valView = fmt.Sprintf("[%5s_]", m.editBuf) // overrides spinner anyway
			}
			valView = lg.NewStyle().Foreground(plt.Hyper).Render(valView)
		}
		prefixView := lg.JoinHorizontal(lg.Left, curView, label, " ", valView)

		// suffix: range / gauge
		var suffixText string
		rngText := fmt.Sprintf("%5d | %-5d", it.min, it.max)

		wRemain := cw - lg.Width(prefixView) - 1 - 1 // spaces in front & back
		switch {
		case wRemain < lg.Width(rngText):
			suffixText = ""
		case wRemain < len("-9999 [■□□] 9999"):
			suffixText = rngText
		default:
			barW := wRemain - lg.Width("-9999 [] 9999")
			pct := float64(it.val-it.min) / max(1.0, float64(it.max-it.min))
			used := max(0, int(pct*float64(barW)))
			bar := strings.Repeat("■", used) + strings.Repeat("□", max(0, barW-used))
			suffixText = fmt.Sprintf("%5d [%s] %-5d", it.min, bar, it.max)
		}
		suffixView := th.Disabled.Render(suffixText)
		rowView := lg.NewStyle().Width(cw).MaxHeight(1).
			Render(lg.JoinHorizontal(lg.Left, prefixView, " ", suffixView))

		rows = append(rows, rowView)
		continue
	}

	if m.statusIsErr {
		rows = append(rows, th.Focus.MaxWidth(cw).MaxHeight(1).Render(" "+m.statusMsg))
	} else {
		rows = append(rows, lg.NewStyle().Width(cw).Render("")) // Bottom padding
	}

	return RenderBoxWithTitle("TUNING", lg.JoinVertical(lg.Left, rows...))
}
