package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m *Model) detailsView(width int) string {
	s := m.dStates[m.selectedGpu]
	// p := s.Limits

	// 1. 定义数据结构
	type row struct {
		f, setL string
		setV    int
		unit    string
		stat    string
		pct     int
		dis     bool
	}

	rows := []row{
		{"Power", "Limit", s.PowerLim, "W", fmt.Sprintf("%d/%dW", s.Power, s.PowerLim), int(float64(s.Power) / float64(s.PowerLim) * 100), false},
		{"Core ", "CO   ", s.CoGpu, "MHz", fmt.Sprintf("%d%% %dMHz", s.UtilGpu, s.ClockGpu), s.UtilGpu, false},
		{"Mem  ", "CO   ", s.CoMem, "MHz", fmt.Sprintf("%d%% %dMHz", s.UtilMem, s.ClockMem), s.UtilMem, false},
		{"Clock", "Limit", s.ClGpu, "MHz", fmt.Sprintf("%dMHz", s.ClockGpu), int(float64(s.ClockGpu) / float64(s.ClGpu) * 100), false},
		{"Fan  ", "Fixed", 0, "%", fmt.Sprintf("%d%% %dRPM", s.FanPct, s.FanRPM), s.FanPct, true},
	}

	lines := make([]string, len(rows))
	for i, r := range rows {
		// 2. 样式准备
		isFoc := i == m.tuningIndex
		pre := "  "
		fStyle, sStyle, vStyle := th.Label, th.Label, th.Value
		if isFoc {
			pre = th.Focus.Render("> ")
			fStyle, sStyle, vStyle = th.Focus, th.Focus, th.Focus
		}
		if r.dis {
			vStyle = th.Disabled
		}

		// 3. 构建 Setter 部分 (固定宽度保证对齐)
		setter := fmt.Sprintf("%s = %s%-3s", sStyle.Render(r.setL), vStyle.Render(fmt.Sprintf("[%4d]", r.setV)), r.unit)
		if r.dis {
			setter = th.Disabled.Render("Unsupported      ")
		}

		// 4. 构建 Status 部分
		status := fmt.Sprintf("[%11s]", r.stat)

		// 5. 动态计算 Gauge 宽度 (重点：减去已知固定宽度，防止换行)
		// pre(2) + field(5) + " > "(3) + setter(20) + " : "(3) + status(13) = 46
		fixedW := 46
		gauge := ""
		if width > fixedW+5 {
			gW := width - fixedW - 4
			fill := max(1, (gW * r.pct / 100))
			bar := lipgloss.NewStyle().Foreground(lipgloss.Color(plt.at(r.pct))).Render(strings.Repeat("|", fill))
			gauge = fmt.Sprintf(" [%s%s]", bar, strings.Repeat(".", gW-fill))
		}

		lines[i] = fmt.Sprintf("%s%s %s : %s%s", pre, fStyle.Render(r.f), setter, status, gauge)
	}

	return lipgloss.NewStyle().Width(width).MaxWidth(width).MaxHeight(5).Render(lipgloss.JoinVertical(lipgloss.Left, lines...))
}
