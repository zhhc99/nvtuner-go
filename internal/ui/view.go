package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m *Model) renderHeader() string {
	// Top Bar: [ Dashboard ] [ Overclock ]
	var tabs []string

	dTitle := " Dashboard "
	oTitle := " Overclock "

	if m.activeTab == TabDashboard {
		tabs = append(tabs, activeTab.Render(dTitle))
		tabs = append(tabs, tab.Render(oTitle))
	} else {
		tabs = append(tabs, tab.Render(dTitle))
		tabs = append(tabs, activeTab.Render(oTitle))
	}

	// Device Info
	s := m.states[m.activeDeviceIdx]
	devInfo := fmt.Sprintf(" GPU %d: %s | %s ", s.Index, s.Name, s.UUID)

	header := lipgloss.JoinHorizontal(lipgloss.Bottom, tabs...)
	header += "\n" + lipgloss.NewStyle().Foreground(subtle).Render(devInfo)
	return header
}

func (m *Model) viewDashboard() string {
	s := m.states[m.activeDeviceIdx]
	doc := strings.Builder{}

	doc.WriteString(m.renderHeader())
	doc.WriteString("\n\n")

	// Left Column: Usage & Temp
	col1 := lipgloss.JoinVertical(lipgloss.Left,
		listHeader("Status"),
		fmt.Sprintf("GPU Util : %d%%", s.GpuUtil),
		fmt.Sprintf("Mem Util : %d%%", s.MemUtil),
		fmt.Sprintf("Memory   : %.2f / %.2f GB", s.MemUsed, s.MemTotal),
		fmt.Sprintf("Temp     : %d°C", s.Temp),
		fmt.Sprintf("Fan      : %d%% (%d RPM)", s.FanPct, s.FanRPM),
	)

	// Right Column: Power & Clocks
	col2 := lipgloss.JoinVertical(lipgloss.Left,
		listHeader("Sensors"),
		fmt.Sprintf("Power    : %dW / %dW", s.Power/1000, s.PowerLim/1000),
		fmt.Sprintf("Core Clk : %d MHz", s.ClockGpu),
		fmt.Sprintf("Mem Clk  : %d MHz", s.ClockMem),
		fmt.Sprintf("Offset C : %+d MHz", s.CoGpu),
		fmt.Sprintf("Offset M : %+d MHz", s.CoMem),
	)

	content := lipgloss.JoinHorizontal(lipgloss.Top,
		listStyle.Render(col1),
		listStyle.Render(col2),
	)

	doc.WriteString(docStyle.Render(content))
	doc.WriteString("\n[Shift+Tab] Mode | [Tab] GPU | [Q] Quit")
	return doc.String()
}

func (m *Model) viewOverclock() string {
	s := m.states[m.activeDeviceIdx]
	doc := strings.Builder{}

	doc.WriteString(m.renderHeader())
	doc.WriteString("\n\n")

	controls := lipgloss.JoinVertical(lipgloss.Left,
		listHeader("Controls (Enter to Edit, R to Reset)"),

		m.renderControl(InputPower, "Power Limit (W)", s.PowerLim/1000,
			fmt.Sprintf("%d to %d", s.Limits.PlMin/1000, s.Limits.PlMax/1000)),

		m.renderControl(InputCoreOffset, "Core CO (MHz)", s.CoGpu,
			fmt.Sprintf("%d to %d", s.Limits.CoGpuMin, s.Limits.CoGpuMax)),

		m.renderControl(InputMemOffset, "Mem CO (MHz)", s.CoMem,
			fmt.Sprintf("%d to %d", s.Limits.CoMemMin, s.Limits.CoMemMax)),

		m.renderControl(InputClockLock, "Core CL (MHz)", s.ClGpu,
			fmt.Sprintf("%d to %d", s.Limits.ClGpuMin, s.Limits.ClGpuMax)),

		"\n"+m.renderStatus(),
	)

	doc.WriteString(docStyle.Render(controls))
	doc.WriteString("\n[Shift+Tab] Mode | [Up/Down] Select | [Enter] Apply | [R] Reset | [S] Save | [Q] Quit")
	return doc.String()
}

func (m *Model) renderControl(id InputField, label string, currentVal int, hint string) string {
	cursor := " "
	labelStyle := lipgloss.NewStyle()

	if m.cursor == id {
		cursor = ">"
		labelStyle = labelStyle.Foreground(special)
	}

	valStr := fmt.Sprintf("%d", currentVal)
	if m.editing && m.cursor == id {
		valStr = m.inputs[id] + "_"
		labelStyle = labelStyle.Bold(true)
	}

	l := lipgloss.PlaceHorizontal(18, lipgloss.Left, labelStyle.Render(label))
	v := inputStyle.Render(valStr)
	h := lipgloss.NewStyle().Foreground(subtle).Render("(" + hint + ")")

	return fmt.Sprintf("%s %s: %s  %s", cursor, l, v, h)
}

func (m *Model) renderStatus() string {
	if m.statusMsg == "" {
		return ""
	}
	s := statusStyle
	if m.statusErr {
		s = s.Foreground(warning)
	} else {
		s = s.Foreground(special)
	}
	return s.Render(m.statusMsg)
}
