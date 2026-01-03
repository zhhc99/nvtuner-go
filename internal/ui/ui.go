package ui

import (
	"fmt"
	"strconv"
	"time"

	"nvtuner-go/internal/config"
	"nvtuner-go/internal/gpu"
	tinyrb "nvtuner-go/internal/utils"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	lg "github.com/charmbracelet/lipgloss"
)

const GIGA = 1024 * 1024 * 1024

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

	tuningIndex       int
	isEditing         bool
	editBuf           string
	statusMsg         string
	statusIsErr       bool
	spinner           spinner.Model
	tuningUpdateUntil []time.Time

	clockHistory []*tinyrb.RingBuffer[DataPoint]
	powerHistory []*tinyrb.RingBuffer[DataPoint]
	tempHistory  []*tinyrb.RingBuffer[DataPoint]
	memHistory   []*tinyrb.RingBuffer[DataPoint]

	help help.Model
}

type tickMsg time.Time
type statusMsg struct {
	text string
	err  bool
}
type DataPoint struct {
	Time  time.Time
	Value float64
}

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

	histClock := make([]*tinyrb.RingBuffer[DataPoint], len(devs))
	histPower := make([]*tinyrb.RingBuffer[DataPoint], len(devs))
	histTemp := make([]*tinyrb.RingBuffer[DataPoint], len(devs))
	histMem := make([]*tinyrb.RingBuffer[DataPoint], len(devs))
	for i := range devs {
		histClock[i] = tinyrb.New[DataPoint](512)
		histPower[i] = tinyrb.New[DataPoint](512)
		histTemp[i] = tinyrb.New[DataPoint](512)
		histMem[i] = tinyrb.New[DataPoint](512)
	}

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = th.Focus

	return &Model{
		driver:  drv,
		mState:  mState,
		devices: devs,
		dStates: dStates,
		config:  cfg,

		spinner:           s,
		tuningUpdateUntil: make([]time.Time, 4),

		clockHistory: histClock,
		powerHistory: histPower,
		tempHistory:  histTemp,
		memHistory:   histMem,

		help: help.New(),
	}, nil
}

func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		doTick(),
		m.spinner.Tick,
	)
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.isEditing {
			return m.handleEditMode(msg)
		}
		switch {
		case key.Matches(msg, keys.Quit):
			return m, tea.Quit
		case key.Matches(msg, keys.Tab):
			m.statusMsg = ""
			m.selectedGpu = (m.selectedGpu + 1) % len(m.devices)
			m.editBuf = ""
		case key.Matches(msg, keys.Uuid):
			m.statusMsg = ""
			m.showUuid = !m.showUuid
		case key.Matches(msg, keys.Up):
			m.statusMsg = ""
			m.tuningIndex--
			m.tuningIndex = max(0, m.tuningIndex)
		case key.Matches(msg, keys.Down):
			m.statusMsg = ""
			m.tuningIndex++
			m.tuningIndex = min(3, m.tuningIndex)
		case key.Matches(msg, keys.Enter):
			m.statusMsg = ""
			d := &m.dStates[m.selectedGpu]
			vals := []int{d.PowerLim, d.CoGpu, d.CoMem, d.ClGpu}
			m.isEditing, m.editBuf = true, strconv.Itoa(vals[m.tuningIndex])
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case statusMsg:
		m.statusMsg, m.statusIsErr = msg.text, msg.err
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case tickMsg:
		now := time.Now()
		for i, d := range m.devices {
			m.dStates[i].FetchOnce(d)

			m.clockHistory[i].Push(DataPoint{Time: now, Value: float64(m.dStates[i].ClockGpu)})
			m.powerHistory[i].Push(DataPoint{Time: now, Value: float64(m.dStates[i].Power)})
			m.tempHistory[i].Push(DataPoint{Time: now, Value: float64(m.dStates[i].Temp)})
			m.memHistory[i].Push(DataPoint{Time: now, Value: float64(m.dStates[i].MemUsed) / GIGA})

		}
		return m, doTick()
	}
	return m, nil
}

func (m *Model) handleEditMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.isEditing = false
		m.statusMsg = "Cancelled"
		m.statusIsErr = false
	case "enter":
		dev := m.devices[m.selectedGpu]
		sets := []func(int) error{
			func(v int) error { return dev.SetPl(v * 1000) },
			dev.SetCoGpu,
			dev.SetCoMem,
			dev.SetClGpu,
		}

		val, err := strconv.Atoi(m.editBuf)
		m.isEditing = false
		if err != nil {
			m.statusIsErr, m.statusMsg = true, "Invalid input"
			return m, nil
		}
		m.tuningUpdateUntil[m.tuningIndex] = time.Now().Add(time.Second / 2)

		return m, func() tea.Msg {
			if err := sets[m.tuningIndex](val); err != nil {
				return statusMsg{err.Error(), true}
			}
			return statusMsg{"Applied Successfully", false}
		}
	case "backspace":
		if len(m.editBuf) > 0 {
			m.editBuf = m.editBuf[:len(m.editBuf)-1]
		}
	default:
		s := msg.String()
		if (s >= "0" && s <= "9") || s == "-" {
			if len(m.editBuf) < 5 { // max 5 digits
				m.editBuf += s
			}
		}
	}
	return m, nil
}

func (m *Model) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}
	if len(m.devices) == 0 {
		return "GPU not found"
	}

	headerText := fmt.Sprintf("%s %s | %s %s", "nvtuner", "v0.0.1",
		m.mState.ManagerName, m.mState.ManagerVersion)
	headerContent := RenderLineWithTitle(m.width, headerText)
	headerView := lg.NewStyle().Foreground(plt.Border).Render(headerContent)

	m.help.Width = m.width
	helpView := m.help.View(keys)
	helpHeight := lg.Height(helpView)

	boxWidth := m.width
	boxHeight := m.height - helpHeight
	cw := max(0, boxWidth-2)

	content := lg.JoinVertical(lg.Center,
		headerView, "", m.dashboardView(cw, 4))

	hRemain := boxHeight - lg.Height(content)

	ds := m.dStates[m.selectedGpu]
	const chartH = 10
	type chartMeta struct {
		name string
		data *tinyrb.RingBuffer[DataPoint]
		max  float64
	}
	chartDefs := []chartMeta{
		{"Temp (°C)", m.tempHistory[m.selectedGpu], 100.0},
		{"Power (W)", m.powerHistory[m.selectedGpu], float64(ds.Limits.PlMax)}, // 已改 Watt
		{"Clock (MHz)", m.clockHistory[m.selectedGpu], float64(ds.Limits.ClGpuMax)},
		{"Mem (MB)", m.memHistory[m.selectedGpu], float64(ds.MemTotal) / 1024 / 1024},
	}
	const (
		IDX_TEMP  = 0
		IDX_POWER = 1
		IDX_CLOCK = 2
		IDX_MEM   = 3
	)

	if cw <= THIN { // simply place everything in one column
		tunView := m.tuningView(cw)
		if hRemain >= lg.Height(tunView) {
			content = lg.JoinVertical(lg.Center, content, "", tunView)
			hRemain -= (1 + lg.Height(tunView))
		}
		for _, c := range chartDefs {
			if hRemain >= chartH {
				v := m.tsView(cw, chartH, c.name, c.data, 0, c.max)
				content = lg.JoinVertical(lg.Center, content, v)
				hRemain -= chartH
			} else {
				break
			}
		}
	} else {
		// [TUNING] [    TEMP    ]
		// [POWER] [CLOCK] [ MEM ]

		wTun := int(float64(cw) * 0.4)
		wTemp := cw - wTun

		tunView := m.tuningView(wTun)
		row1H := lg.Height(tunView)

		if hRemain >= row1H {
			c := chartDefs[IDX_TEMP]
			tempView := m.tsView(wTemp, row1H, c.name, c.data, 0, c.max)

			row1 := lg.JoinHorizontal(lg.Top, tunView, tempView)
			content = lg.JoinVertical(lg.Center, content, "", row1)
			hRemain -= (1 + row1H)
		}

		if hRemain >= chartH {
			wCol := cw / 3
			wColLast := cw - (wCol * 2)

			bottomRowCharts := []struct {
				def   chartMeta
				width int
			}{
				{chartDefs[IDX_POWER], wCol},
				{chartDefs[IDX_CLOCK], wCol},
				{chartDefs[IDX_MEM], wColLast},
			}

			var views []string
			for _, item := range bottomRowCharts {
				v := m.tsView(item.width, hRemain, item.def.name, item.def.data, 0, item.def.max)
				views = append(views, v)
			}

			row2 := lg.JoinHorizontal(lg.Top, views...)
			content = lg.JoinVertical(lg.Center, content, row2)
		}
	}

	contentView := lg.NewStyle().
		Width(boxWidth).
		Height(boxHeight).
		Render(content)

	return lg.JoinVertical(lg.Left,
		contentView,
		helpView,
	)
}

func doTick() tea.Cmd {
	return tea.Tick(time.Second/2, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}
