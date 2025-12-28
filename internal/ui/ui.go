package ui

import (
	"fmt"
	"time"

	"nvtuner-go/internal/config"
	"nvtuner-go/internal/gpu"

	"github.com/charmbracelet/bubbles/help"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	outerBoxStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("69"))
	innerBoxStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("203"))
	// BorderBackground(lipgloss.Color("63"))
)

var _ tea.Model = (*Model)(nil)

type Model struct {
	driver  gpu.Manager
	mState  gpu.MState
	devices []gpu.Device
	dStates []gpu.DState
	config  *config.Manager

	width, height int

	help help.Model
}

type tickMsg time.Time

func New(drv gpu.Manager, cfg *config.Manager) (*Model, error) {
	devs, err := drv.Devices()
	if err != nil {
		return nil, err
	}
	mState := gpu.MState{}
	mState.FetchOnce(drv)
	dStates := make([]gpu.DState, len(devs))
	for i, d := range devs {
		dStates[i].FetchOnce(d)
	}

	return &Model{
		driver:  drv,
		mState:  mState,
		devices: devs,
		dStates: dStates,
		config:  cfg,
		help:    help.New(),
	}, nil
}

func (m *Model) Init() tea.Cmd { return doTick() }

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		if msg.String() == "q" {
			return m, tea.Quit
		}
	case tickMsg:
		for i, d := range m.devices {
			m.dStates[i].FetchOnce(d)
		}
		return m, doTick()
	}
	return m, nil
}

func (m *Model) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}

	m.help.Width = m.width
	helpView := m.help.View(keys)
	helpHeight := lipgloss.Height(helpView)

	contentWidth := m.width - 2
	contentHeight := m.height - helpHeight - 2

	headerContent := lipgloss.NewStyle().
		Inline(true).
		Render(fmt.Sprintf(" %s %s | %s %s", "nvtuner", "v0.0.1",
			m.mState.ManagerName, m.mState.ManagerVersion),
		)

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		headerContent,
		"",
		m.dashboardView(contentWidth),
	)

	boxedView := outerBoxStyle.
		Width(contentWidth).
		Height(contentHeight).
		Render(content)

	return lipgloss.JoinVertical(lipgloss.Left, boxedView, helpView)
}

func (m *Model) dashboardView(width int) string {
	var rows []string
	for _, s := range m.dStates {
		rows = append(rows, barView(&s, width))
	}
	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

func doTick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}
