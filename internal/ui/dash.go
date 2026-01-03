package ui

import (
	"fmt"
	"nvtuner-go/internal/gpu"
	"strings"

	lg "github.com/charmbracelet/lipgloss"
)

func (m *Model) dashboardView(width, maxHeight int) string {
	n := len(m.dStates)
	cols := 1
	if !(width <= THIN) {
		cols = (n + 3) / 4
		if width >= WIDE && cols < 2 {
			cols = 2
		}
	}

	colW := width / cols
	rows := (n + cols - 1) / cols
	var out []string

	for r := range rows {
		var row []string
		for c := 0; c < cols; c++ {
			i := r*cols + c
			if i < n {
				row = append(row, barView(&m.dStates[i], colW, i == m.selectedGpu))
			} else if width >= 100 {
				row = append(row, m.fillerView(colW))
			}
		}
		out = append(out, lg.JoinHorizontal(lg.Top, row...))
	}
	return lg.NewStyle().MaxHeight(maxHeight).Render(lg.JoinVertical(lg.Left, out...))
}

func (m *Model) fillerView(width int) string {
	fill := ""
	if w := width - 4; w > 0 {
		fill = strings.Repeat(" -", w/2+1)[:w]
	}
	return lg.NewStyle().Width(width).MaxHeight(1).Foreground(plt.Dim).Render("  [" + fill + "]")
}

func barView(s *gpu.DState, width int, selected bool) string {
	// data
	memUsedG := float64(s.MemUsed) / GIGA
	memTotalG := float64(s.MemTotal) / GIGA
	memPct := int(memUsedG / memTotalG)

	// views
	prefixView := fmt.Sprintf("%2d", s.Index) + "["
	suffixView := "]"
	if selected {
		prefixView = lg.NewStyle().Foreground(plt.Hyper).Render(prefixView)
		suffixView = lg.NewStyle().Foreground(plt.Hyper).Render(suffixView)
	}
	coreView := th.Value.Render(fmt.Sprintf("%3d%% %4dMHz%3d°C%4dW"+" ",
		s.UtilGpu, s.ClockGpu, s.Temp, s.Power/1024))

	wMem := width - lg.Width(prefixView) - lg.Width(coreView) - lg.Width(suffixView)
	memThin := th.Value.Render(fmt.Sprintf("M%3d%%", memPct))
	memRglr := th.Value.Render(fmt.Sprintf("%3s/%-3sG", fm3(memUsedG), fm3(memTotalG)))
	var memWide string
	if lg.Width(memRglr)+lg.Width(" ||") <= wMem {
		gaugeSize := wMem - lg.Width(memRglr) - 1
		gaugeUsed := max(1, int(float64(gaugeSize)*memUsedG/memTotalG))

		barView := gaugeView(gaugeSize, gaugeUsed)
		memWide = lg.JoinHorizontal(lg.Left,
			memRglr,
			" ",
			barView,
		)
	}

	var out string
	switch {
	case wMem <= 0:
		out = coreView
	case wMem < lg.Width(memRglr):
		out = lg.JoinHorizontal(lg.Left, prefixView, coreView, memThin, suffixView)
	case len(memWide) == 0:
		out = lg.JoinHorizontal(lg.Left, prefixView, coreView, memRglr, suffixView)
	default:
		out = lg.JoinHorizontal(lg.Left, prefixView, coreView, memWide, suffixView)
	}
	return lg.NewStyle().Width(width).MaxHeight(1).Render(out)
}

func gaugeView(total int, used int) string {
	bar := strings.Repeat("|", used) + strings.Repeat(".", total-used)
	return lg.NewStyle().Foreground(lg.Color(plt.at(used))).MaxHeight(1).Render(bar)
}

func fm3(f float64) string {
	s := fmt.Sprintf("%.1f", f)
	if len(s) > 3 {
		s = fmt.Sprintf("%.0f", f)
	}
	return s
}
