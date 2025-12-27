package ui

import (
	"fmt"
	"nvtuner-go/internal/gpu"
	"strconv"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type Tab int

const (
	TabDashboard Tab = iota
	TabOverclock
)

type InputField int

const (
	InputNone InputField = iota
	InputPower
	InputCoreOffset
	InputMemOffset
	InputClockLock
)

type tickMsg time.Time
type statusMsg struct {
	text string
	err  bool
}

type Model struct {
	driver  gpu.Manager
	devices []gpu.Device
	states  []gpu.State

	// Navigation
	activeTab       Tab
	activeDeviceIdx int
	width, height   int
	err             error

	// Overclock Editing
	cursor    InputField
	inputs    map[InputField]string
	editing   bool
	statusMsg string
	statusErr bool
}

func NewModel(drv gpu.Manager) (*Model, error) {
	devs, err := drv.Devices()
	if err != nil {
		return nil, err
	}

	states := make([]gpu.State, len(devs))
	for i, d := range devs {
		states[i] = fetchState(i, d)
	}

	return &Model{
		driver:          drv,
		devices:         devs,
		states:          states,
		activeTab:       TabDashboard,
		activeDeviceIdx: 0,
		cursor:          InputPower,
		inputs:          make(map[InputField]string),
	}, nil
}

func (m *Model) Init() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		// Global Keys
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "tab":
			// Switch Device
			if len(m.devices) > 1 {
				m.activeDeviceIdx = (m.activeDeviceIdx + 1) % len(m.devices)
				m.resetEditState()
			}
			return m, nil
		case "shift+tab":
			// Switch Tab (Dashboard <-> Overclock)
			if m.activeTab == TabDashboard {
				m.activeTab = TabOverclock
			} else {
				m.activeTab = TabDashboard
			}
			m.resetEditState()
			return m, nil
		}

		// Tab specific handling
		if m.activeTab == TabOverclock {
			return m.updateOverclock(msg)
		}

	case tickMsg:
		// Refresh stats
		m.states[m.activeDeviceIdx] = fetchState(m.activeDeviceIdx, m.devices[m.activeDeviceIdx])
		return m, tea.Tick(time.Second, func(t time.Time) tea.Msg {
			return tickMsg(t)
		})

	case statusMsg:
		m.statusMsg = msg.text
		m.statusErr = msg.err
		m.editing = false
	}

	return m, nil
}

func (m *Model) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error: %v", m.err)
	}
	if len(m.devices) == 0 {
		return "No NVIDIA devices found."
	}

	switch m.activeTab {
	case TabDashboard:
		return m.viewDashboard()
	case TabOverclock:
		return m.viewOverclock()
	default:
		return "Unknown Tab"
	}
}

// --- Helpers ---

func (m *Model) resetEditState() {
	m.editing = false
	m.statusMsg = ""
	m.inputs = make(map[InputField]string)
}

func fetchState(idx int, d gpu.Device) gpu.State {
	s := gpu.State{
		Index: idx,
		Name:  d.GetName(),
		UUID:  d.GetUUID(),
	}

	s.GpuUtil, s.MemUtil, _ = d.GetUtil()
	s.Temp, _ = d.GetTemperature()
	s.FanPct, s.FanRPM, _ = d.GetFanSpeed()
	s.Power, _ = d.GetPower()
	s.PowerLim, _ = d.GetPl()
	s.ClockGpu, s.ClockMem, _ = d.GetClocks()

	tMem, _, uMem, _ := d.GetMemory()
	s.MemTotal = float64(tMem) / 1024 / 1024 / 1024
	s.MemUsed = float64(uMem) / 1024 / 1024 / 1024

	s.CoGpu, _ = d.GetCoGpu()
	s.CoMem, _ = d.GetCoMem()

	s.Limits.PlMin, s.Limits.PlMax, _ = d.GetPlLim()
	s.Limits.CoGpuMin, s.Limits.CoGpuMax, _ = d.GetCoLimGpu()
	s.Limits.CoMemMin, s.Limits.CoMemMax, _ = d.GetCoLimMem()

	return s
}

// Helper to check for editing commands in Overclock tab
func (m *Model) updateOverclock(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Navigation
	if !m.editing {
		switch msg.String() {
		case "up", "k":
			if m.cursor > InputPower {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < InputClockLock {
				m.cursor++
			}
		case "enter":
			m.startEditing()
		case "r":
			// Quick Reset for current field
			return m, m.resetCurrentField()
		}
		return m, nil
	}

	// Editing
	switch msg.String() {
	case "esc":
		m.editing = false
		m.statusMsg = "Cancelled"
		m.statusErr = false
	case "enter":
		return m, m.applyChanges()
	case "backspace":
		val := m.inputs[m.cursor]
		if len(val) > 0 {
			m.inputs[m.cursor] = val[:len(val)-1]
		}
	default:
		// Simple numeric filter
		if len(msg.String()) == 1 {
			r := msg.Runes[0]
			if (r >= '0' && r <= '9') || r == '-' {
				m.inputs[m.cursor] += string(r)
			}
		}
	}
	return m, nil
}

func (m *Model) startEditing() {
	m.editing = true
	s := m.states[m.activeDeviceIdx]

	// Pre-fill
	val := ""
	switch m.cursor {
	case InputPower:
		val = strconv.Itoa(s.PowerLim / 1000)
	case InputCoreOffset:
		val = strconv.Itoa(s.CoGpu)
	case InputMemOffset:
		val = strconv.Itoa(s.CoMem)
	case InputClockLock:
		val = "0" // Default to 0 for locked clock UI
	}
	m.inputs[m.cursor] = val
}

func (m *Model) applyChanges() tea.Cmd {
	dev := m.devices[m.activeDeviceIdx]
	val, err := strconv.Atoi(m.inputs[m.cursor])
	if err != nil {
		return func() tea.Msg { return statusMsg{"Invalid number", true} }
	}

	return func() tea.Msg {
		var err error
		switch m.cursor {
		case InputPower:
			err = dev.SetPl(val * 1000)
		case InputCoreOffset:
			err = dev.SetCoGpu(val)
		case InputMemOffset:
			err = dev.SetCoMem(val)
		case InputClockLock:
			err = dev.SetClGpu(val)
		}

		if err != nil {
			return statusMsg{fmt.Sprintf("Error: %v", err), true}
		}
		return statusMsg{"Applied successfully", false}
	}
}

func (m *Model) resetCurrentField() tea.Cmd {
	dev := m.devices[m.activeDeviceIdx]
	return func() tea.Msg {
		var err error
		switch m.cursor {
		case InputPower:
			err = dev.ResetPl()
		case InputCoreOffset:
			err = dev.ResetCoGpu()
		case InputMemOffset:
			err = dev.ResetCoMem()
		case InputClockLock:
			err = dev.ResetClGpu()
		}
		if err != nil {
			return statusMsg{fmt.Sprintf("Reset Error: %v", err), true}
		}
		return statusMsg{"Reset successfully", false}
	}
}
