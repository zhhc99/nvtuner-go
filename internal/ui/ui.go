package ui

import (
	"fmt"
	"strings"
	"time"

	"nvtuner-go/internal/config"
	"nvtuner-go/internal/gpu"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
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
	sepStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("69"))
	activeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("205")).
			Bold(true)
	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("244"))
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
	selectedGpu   int
	showUuid      bool

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
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Quit):
			return m, tea.Quit
		case key.Matches(msg, keys.Tab):
			m.selectedGpu = (m.selectedGpu + 1) % len(m.devices)
		case key.Matches(msg, keys.Uuid):
			m.showUuid = !m.showUuid
		}
		return m, nil
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
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

	headerContent := lipgloss.NewStyle().
		Inline(true).
		Render(fmt.Sprintf(" %s %s | %s %s", "nvtuner", "v0.0.1",
			m.mState.ManagerName, m.mState.ManagerVersion),
		)

	m.help.Width = m.width
	helpView := m.help.View(keys)
	helpHeight := lipgloss.Height(helpView)

	contentWidth := m.width - 2
	contentHeight := m.height - helpHeight - 2

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		headerContent,
		"",
		m.dashboardView(contentWidth),
		m.selectedGpuView(contentWidth),
	)

	contentView := outerBoxStyle.
		Width(contentWidth).
		Height(contentHeight).
		Render(content)

	return lipgloss.JoinVertical(lipgloss.Left,
		contentView,
		helpView,
	)
}

func (m *Model) dashboardView(width int) string {
	var rows []string
	for _, s := range m.dStates {
		rows = append(rows, barView(&s, width))
	}
	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

func (m *Model) selectedGpuView(width int) string {
	topSep := sepStyle.Width(width).Render(m.sepH(width, 0))
	botSep := sepStyle.Width(width).Render(m.sepH(width, 2))

	dState := m.dStates[m.selectedGpu]
	prefix := activeStyle.Render("> ")
	model := fmt.Sprintf("GPU %d: %s ", dState.Index, dState.Name)
	var uuid string
	if m.showUuid {
		uuid = dimStyle.Render(fmt.Sprintf("(%s)", dState.UUID))
	} else {
		uuid = dimStyle.Render("")
	}
	contentRow := lipgloss.NewStyle().
		PaddingLeft(1).
		Width(width).
		MaxWidth(width).
		MaxHeight(1).
		Render(prefix + model + uuid)

	return lipgloss.JoinVertical(lipgloss.Left,
		topSep,
		contentRow,
		botSep,
	)
}

func (m *Model) sepH(width int, spaces int) string {
	width = max(0, width)
	spaces = min(width/2, spaces)
	rep := strings.Repeat
	return rep(" ", spaces) + rep("─", width-2*spaces) + rep(" ", spaces)
}

func doTick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}
