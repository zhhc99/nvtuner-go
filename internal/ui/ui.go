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
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	lg "github.com/charmbracelet/lipgloss"
)

const GIGA = 1024 * 1024 * 1024

var _ tea.Model = (*Model)(nil)

type Model struct {
	width, height int

	config  *config.Manager
	driver  gpu.Manager
	mState  gpu.MState
	devices []gpu.Device
	dStates []gpu.DState

	selectedGpu int
	showUuid    bool

	tuningIndex  int
	isEditing    bool
	tuningParams []TuningParam
	tuningInput  textinput.Model

	statusMsg   string
	statusIsErr bool

	clockHistory []*tinyrb.RingBuffer[DataPoint]
	powerHistory []*tinyrb.RingBuffer[DataPoint]
	tempHistory  []*tinyrb.RingBuffer[DataPoint]
	memHistory   []*tinyrb.RingBuffer[DataPoint]

	help  help.Model
	popup PopupState
}
type tickMsg time.Time
type statusMsg struct { // TODO: why do we need this?
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

	// init manager & device states
	mState := gpu.MState{}
	mState.FetchOnce(drv)
	dStates := make([]gpu.DState, len(devs))
	for i, d := range devs {
		dStates[i].FetchOnce(d)
	}

	// load / create configs
	for _, d := range devs {
		uuid := d.GetUUID()
		if _, ok := cfg.Get(uuid); !ok {
			defPl, _ := d.GetPlDefault()
			_, defCl, _ := d.GetClLimGpu()
			initSetting := config.GpuSettings{
				PowerLimit: defPl,
				GpuCO:      0,
				MemCO:      0,
				GpuCL:      defCl,
			}
			cfg.Set(uuid, initSetting)
		}
	}
	cfg.Save()

	// init tuning params
	params := []TuningParam{
		{
			ID: "pl", Label: "POWER LIMIT:", ShortLabel: "PL", Unit: "W",
			GetConfig:  func(c config.GpuSettings) int { return c.PowerLimit },
			GetCurrent: func(d gpu.DState) int { return d.PowerLim },
			GetLimits:  func(d gpu.DState) (int, int) { return d.Limits.PlMin, d.Limits.PlMax },
			SetConfig:  func(c *config.GpuSettings, v int) { c.PowerLimit = v },
			Apply:      func(d gpu.Device, v int) error { return d.SetPl(v) },
		},
		{
			ID: "gpu_co", Label: "GPU CO:     ", ShortLabel: "G.CO", Unit: "MHz",
			GetConfig:  func(c config.GpuSettings) int { return c.GpuCO },
			GetCurrent: func(d gpu.DState) int { return d.CoGpu },
			GetLimits:  func(d gpu.DState) (int, int) { return d.Limits.CoGpuMin, d.Limits.CoGpuMax },
			SetConfig:  func(c *config.GpuSettings, v int) { c.GpuCO = v },
			Apply:      func(d gpu.Device, v int) error { return d.SetCoGpu(v) },
		},
		{
			ID: "mem_co", Label: "MEMORY CO:  ", ShortLabel: "M.CO", Unit: "MHz",
			GetConfig:  func(c config.GpuSettings) int { return c.MemCO },
			GetCurrent: func(d gpu.DState) int { return d.CoMem },
			GetLimits:  func(d gpu.DState) (int, int) { return d.Limits.CoMemMin, d.Limits.CoMemMax },
			SetConfig:  func(c *config.GpuSettings, v int) { c.MemCO = v },
			Apply:      func(d gpu.Device, v int) error { return d.SetCoMem(v) },
		},
		{
			ID: "gpu_cl", Label: "GPU LIMIT:  ", ShortLabel: "G.CL", Unit: "MHz",
			GetConfig:  func(c config.GpuSettings) int { return c.GpuCL },
			GetCurrent: func(d gpu.DState) int { return d.ClGpu },
			GetLimits:  func(d gpu.DState) (int, int) { return d.Limits.ClGpuMin, d.Limits.ClGpuMax },
			SetConfig:  func(c *config.GpuSettings, v int) { c.GpuCL = v },
			Apply: func(d gpu.Device, v int) error {
				if v == gpu.NO_VALUE || v < 0 {
					return d.ResetClGpu()
				}
				return d.SetClGpu(v)
			},
		},
	}

	ti := textinput.New()
	ti.Prompt = ""
	ti.CharLimit = 5
	ti.Width = 5
	ti.TextStyle = th.Focus

	// init history data
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

	return &Model{
		config:  cfg,
		driver:  drv,
		mState:  mState,
		devices: devs,
		dStates: dStates,

		tuningParams: params,
		tuningInput:  ti,

		clockHistory: histClock,
		powerHistory: histPower,
		tempHistory:  histTemp,
		memHistory:   histMem,

		help:  help.New(),
		popup: PopupState{Type: PopupNone},
	}, nil
}

func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		doTick(),
		textinput.Blink,
	)
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// popup handling
	if m.popup.Type != PopupNone {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "esc":
				m.popup.Type = PopupNone
				m.statusMsg = "Cancelled"
				m.statusIsErr = false
			case "enter":
				return m.handlePopupConfirm()
			}
		}
		return m, nil
	}

	// editing handling
	if m.isEditing {
		var cmd tea.Cmd
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.Type {
			case tea.KeyEsc: // cancel tuning edit
				m.isEditing = false
				m.tuningInput.Blur()
				m.statusMsg = "Edit Cancelled"
				m.statusIsErr = false
				return m, nil
			case tea.KeyEnter: // save tuning profiles to config
				valStr := m.tuningInput.Value()
				val, err := strconv.Atoi(valStr)
				if err != nil {
					m.isEditing = false
					m.tuningInput.Blur()
					m.statusIsErr, m.statusMsg = true, "Invalid Input"
					return m, nil
				}

				d := &m.dStates[m.selectedGpu]
				cfg, _ := m.config.Get(d.UUID)
				param := m.tuningParams[m.tuningIndex]
				minVal, maxVal := param.GetLimits(*d)
				if val < minVal || val > maxVal {
					m.isEditing = false
					m.tuningInput.Blur()
					m.statusIsErr, m.statusMsg = true, "Value out of range"
					return m, nil
				}

				param.SetConfig(&cfg, val)
				m.config.Set(d.UUID, cfg)

				if err := m.config.Save(); err != nil {
					m.statusIsErr, m.statusMsg = true, fmt.Sprintf("Save Failed: %v", err)
				} else {
					m.statusIsErr, m.statusMsg = false, "Config Saved. Press 'a' to Apply."
				}

				m.isEditing = false
				m.tuningInput.Blur()
				return m, nil
			}
		}
		m.tuningInput, cmd = m.tuningInput.Update(msg)
		return m, cmd
	}

	// navigation
	switch msg := msg.(type) {
	case tea.KeyMsg:
		m.statusMsg = ""
		switch {
		case key.Matches(msg, keys.Quit):
			return m, tea.Quit
		case key.Matches(msg, keys.Tab):
			m.selectedGpu = (m.selectedGpu + 1) % len(m.devices)
		case key.Matches(msg, keys.Uuid):
			m.showUuid = !m.showUuid
		case key.Matches(msg, keys.Up):
			m.tuningIndex = max(0, m.tuningIndex-1)
		case key.Matches(msg, keys.Down):
			m.tuningIndex = min(3, m.tuningIndex+1)
		case key.Matches(msg, keys.Enter):
			m.isEditing = true
			cfg, _ := m.config.Get(m.dStates[m.selectedGpu].UUID)
			val := m.tuningParams[m.tuningIndex].GetConfig(cfg)
			m.tuningInput.SetValue(strconv.Itoa(val))
			m.tuningInput.Focus()
		case key.Matches(msg, keys.Apply):
			m.popup = m.makeApplyPopup()
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case statusMsg:
		m.statusMsg, m.statusIsErr = msg.text, msg.err
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

func (m *Model) handlePopupConfirm() (tea.Model, tea.Cmd) {
	if m.popup.Type == PopupApply {
		m.popup.Type = PopupNone

		dev := m.devices[m.selectedGpu]
		cfg, _ := m.config.Get(m.dStates[m.selectedGpu].UUID)
		hasFailure := false
		failureMsg := ""
		for _, p := range m.tuningParams {
			if err := p.Apply(dev, p.GetConfig(cfg)); err != nil {
				hasFailure = true
				failureMsg += fmt.Sprintf("%s: %v ", p.ShortLabel, err)
			}
		}

		if hasFailure {
			m.statusIsErr = true
			m.statusMsg = failureMsg
			return m, nil
		}

		m.statusIsErr = false
		m.statusMsg = "Settings Applied Successfully"
		return m, nil
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

	if m.popup.Type != PopupNone {
		return m.popupView(m.width, m.height)
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

	dash := m.dashboardView(cw, 4)
	uuidLine := ""
	if m.showUuid {
		uuidLine = th.Disabled.Width(cw).Render(m.dStates[m.selectedGpu].UUID)
	}
	content := lg.JoinVertical(lg.Center,
		headerView, "", dash, uuidLine)

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
			content = lg.JoinVertical(lg.Center, content, tunView)
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
			content = lg.JoinVertical(lg.Center, content, row1)
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
