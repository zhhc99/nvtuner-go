package ui

import (
	"fmt"
	"nvtuner-go/internal/gpu"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const GIGA = 1024 * 1024 * 1024

func barView(s *gpu.DState, width int) string {
	memUsedG := float64(s.MemUsed) / GIGA
	memTotalG := float64(s.MemTotal) / GIGA

	core := fmt.Sprintf("%2d[%3d%% %4dMHz%3d°C%4dW]",
		s.Index, s.UtilGpu, s.ClockGpu, s.Temp, s.Power/1024)
	szRegular := lipgloss.Width(core) + 10 // "[xxx/yyyG]"

	style := lipgloss.NewStyle().MaxWidth(width)
	var mem string
	memPct := int(memUsedG / memTotalG)
	if width < szRegular {
		// short: "16[100% 2210MHz 49°C 151W][M 45%]"
		mem = fmt.Sprintf("[M%3d%%]", memPct)
	} else if width-szRegular < 3 {
		// "[xxx/yyyG␣||]"
		// regular: "16[100% 2210MHz 49°C 151W][2.0/8.0G]"
		mem = fmt.Sprintf("[%3s/%-3sG]", fm3(memUsedG), fm3(memTotalG))
	} else {
		// large: "16[100% 2210MHz 49°C 151W][2.0/8.0G |||  ]"
		gaugeSz := width - szRegular - 1
		usedUnits := int(float64(gaugeSz) * memUsedG / memTotalG)
		usedUnits = max(1, usedUnits)
		bar := lipgloss.NewStyle().
			Foreground(lipgloss.Color(barColor(memPct))).
			Render(strings.Repeat("|", usedUnits)) +
			strings.Repeat(" ", gaugeSz-usedUnits)
		mem = fmt.Sprintf("[%3s/%-3sG %s]", fm3(memUsedG), fm3(memTotalG), bar)
	}

	return style.Render(fmt.Sprintf("%s%s", core, mem))
}

func barColor(util int) string {
	if util > 80 {
		return "196"
	}
	if util > 50 {
		return "208"
	}
	return "42"
}

func fm3(f float64) string {
	s := fmt.Sprintf("%.1f", f)
	if len(s) > 3 {
		s = fmt.Sprintf("%.0f", f)
	}
	return s
}
