package nvidia

import (
	"errors"
	"fmt"
	"nvtuner-go/internal/gpu"
	"strings"
	"unsafe"
)

var _ gpu.Device = (*NvidiaGpu)(nil)

type NvidiaGpu struct {
	handle  Device
	symbols *RawSymbols
	name    string
	uuid    string

	cachedClGpu int
}

func (g *NvidiaGpu) GetName() string { return g.name }
func (g *NvidiaGpu) GetUUID() string { return g.uuid }

func (g *NvidiaGpu) fetchName() string {
	var buf [DEVICE_NAME_BUFFER_SIZE]byte
	if ret := g.symbols.DeviceGetName(g.handle, &buf[0], DEVICE_NAME_BUFFER_SIZE); ret != SUCCESS {
		return "Unknown Nvidia GPU"
	}
	return strings.TrimRight(string(buf[:]), "\x00")
}

func (g *NvidiaGpu) fetchUUID() string {
	var buf [DEVICE_UUID_BUFFER_SIZE]byte
	if ret := g.symbols.DeviceGetUUID(g.handle, &buf[0], DEVICE_UUID_BUFFER_SIZE); ret != SUCCESS {
		return "Unknown UUID"
	}
	return strings.TrimRight(string(buf[:]), "\x00")
}

func (g *NvidiaGpu) GetUtil() (int, int, error) {
	var util Utilization
	if ret := g.symbols.DeviceGetUtilizationRates(g.handle, &util); ret != SUCCESS {
		return 0, 0, fmt.Errorf(g.symbols.StringFromReturn(ret))
	}
	return int(util.Gpu), int(util.Memory), nil
}

func (g *NvidiaGpu) GetClocks() (int, int, error) {
	var gclk, mclk uint32
	if ret := g.symbols.DeviceGetClockInfo(g.handle, CLOCK_GRAPHICS, &gclk); ret != SUCCESS {
		return 0, 0, fmt.Errorf(g.symbols.StringFromReturn(ret))
	}
	if ret := g.symbols.DeviceGetClockInfo(g.handle, CLOCK_MEM, &mclk); ret != SUCCESS {
		return int(gclk), 0, nil
	}
	return int(gclk), int(mclk), nil
}

func (g *NvidiaGpu) GetMemory() (int, int, int, error) {
	var mem Memory
	if ret := g.symbols.DeviceGetMemoryInfo(g.handle, &mem); ret != SUCCESS {
		return 0, 0, 0, fmt.Errorf(g.symbols.StringFromReturn(ret))
	}
	return int(mem.Total), int(mem.Free), int(mem.Used), nil
}

func (g *NvidiaGpu) GetPower() (int, error) {
	var mw uint32
	if ret := g.symbols.DeviceGetPowerUsage(g.handle, &mw); ret == SUCCESS {
		return int(mw), nil
	}
	return g.getPowerViaSample()
}

func (g *NvidiaGpu) getPowerViaSample() (int, error) {
	var (
		sample      Sample
		sampleType  ValueType
		sampleCount uint32 = 1
	)

	ret := g.symbols.DeviceGetSamples(g.handle, TOTAL_POWER_SAMPLES, 0, &sampleType, &sampleCount, &sample)
	if ret != SUCCESS || sampleCount == 0 {
		return 0, fmt.Errorf("sampling failed: %s", g.symbols.StringFromReturn(ret))
	}

	var mw float64
	switch sampleType {
	case VALUE_TYPE_DOUBLE:
		mw = *(*float64)(unsafe.Pointer(&sample.SampleValue.Data))
	case VALUE_TYPE_UNSIGNED_INT:
		mw = float64(*(*uint32)(unsafe.Pointer(&sample.SampleValue.Data)))
	case VALUE_TYPE_UNSIGNED_LONG, VALUE_TYPE_UNSIGNED_LONG_LONG:
		mw = float64(*(*uint64)(unsafe.Pointer(&sample.SampleValue.Data)))
	default:
		mw = float64(sample.SampleValue.AsUint())
	}

	return int(mw), nil
}

func (g *NvidiaGpu) GetTemperature() (int, error) {
	// fallback
	if g.symbols.DeviceGetTemperatureV == nil {
		var temp uint32
		if ret := g.symbols.DeviceGetTemperature(g.handle, TEMPERATURE_GPU, &temp); ret != SUCCESS {
			return 0, fmt.Errorf(g.symbols.StringFromReturn(ret))
		}
		return int(temp), nil
	}

	var temp Temperature
	temp.Version = VERSION_TEMPERATURE
	temp.SensorType = TEMPERATURE_GPU
	if ret := g.symbols.DeviceGetTemperatureV(g.handle, &temp); ret != SUCCESS {
		return 0, fmt.Errorf(g.symbols.StringFromReturn(ret))
	}
	return int(temp.Temperature), nil

}

func (g *NvidiaGpu) GetFanSpeed() (int, int, error) {
	var percent uint32
	if ret := g.symbols.DeviceGetFanSpeed(g.handle, &percent); ret != SUCCESS {
		return 0, 0, fmt.Errorf(g.symbols.StringFromReturn(ret))
	}

	// error only if fan speed percent is not available
	if g.symbols.DeviceGetFanSpeedRPM != nil {
		var fan FanSpeedInfo
		fan.Version = VERSION_FAN_SPEED
		if g.symbols.DeviceGetFanSpeedRPM(g.handle, &fan) == SUCCESS {
			return int(percent), int(fan.Speed), nil
		}
	}
	return int(percent), 0, nil
}

func (g *NvidiaGpu) GetPl() (int, error) {
	var mw uint32
	if ret := g.symbols.DeviceGetEnforcedPowerLimit(g.handle, &mw); ret != SUCCESS {
		return 0, fmt.Errorf(g.symbols.StringFromReturn(ret))
	}
	return int(mw), nil
}

func (g *NvidiaGpu) GetCoGpu() (int, error) {
	// fallback
	if g.symbols.DeviceGetClockOffsets == nil {
		var co int32
		if ret := g.symbols.DeviceGetGpcClkVfOffset(g.handle, &co); ret != SUCCESS {
			return 0, fmt.Errorf(g.symbols.StringFromReturn(ret))
		}
		return int(co), nil
	}

	var co ClockOffset
	co.Version = VERSION_CLOCK_OFFSET
	co.Type = CLOCK_GRAPHICS
	co.Pstate = PSTATE_0
	if ret := g.symbols.DeviceGetClockOffsets(g.handle, &co); ret != SUCCESS {
		return 0, fmt.Errorf(g.symbols.StringFromReturn(ret))
	}
	return int(co.ClockOffsetMHz), nil
}

func (g *NvidiaGpu) GetCoMem() (int, error) {
	// fallback
	if g.symbols.DeviceGetClockOffsets == nil {
		var co int32
		if ret := g.symbols.DeviceGetMemClkVfOffset(g.handle, &co); ret != SUCCESS {
			return 0, fmt.Errorf(g.symbols.StringFromReturn(ret))
		}
		return int(co), nil
	}

	var co ClockOffset
	co.Version = VERSION_CLOCK_OFFSET
	co.Type = CLOCK_MEM
	co.Pstate = PSTATE_0
	if ret := g.symbols.DeviceGetClockOffsets(g.handle, &co); ret != SUCCESS {
		return 0, fmt.Errorf(g.symbols.StringFromReturn(ret))
	}
	return int(co.ClockOffsetMHz), nil
}

func (g *NvidiaGpu) GetClGpu() (int, error) {
	if g.cachedClGpu > 0 {
		return g.cachedClGpu, nil
	}

	// assume not locked if cache is not set yet. return stocked limit here.
	_, max, err := g.GetClLimGpu()
	if err != nil {
		return 0, err
	}
	return max, nil
}

func (g *NvidiaGpu) GetPlLim() (int, int, error) {
	var min, max uint32
	if ret := g.symbols.DeviceGetPowerManagementLimitConstraints(g.handle, &min, &max); ret != SUCCESS {
		return 0, 0, fmt.Errorf(g.symbols.StringFromReturn(ret))
	}
	return int(min), int(max), nil
}

func (g *NvidiaGpu) GetCoLimGpu() (int, int, error) {
	// fallback
	if g.symbols.DeviceGetClockOffsets == nil {
		var min, max int32
		if ret := g.symbols.DeviceGetGpcClkMinMaxVfOffset(g.handle, &min, &max); ret != SUCCESS {
			return 0, 0, fmt.Errorf(g.symbols.StringFromReturn(ret))
		}
		return int(min), int(max), nil
	}

	var co ClockOffset
	co.Version = VERSION_CLOCK_OFFSET
	co.Type = CLOCK_GRAPHICS
	co.Pstate = PSTATE_0
	if ret := g.symbols.DeviceGetClockOffsets(g.handle, &co); ret != SUCCESS {
		return 0, 0, fmt.Errorf(g.symbols.StringFromReturn(ret))
	}
	return int(co.MinClockOffsetMHz), int(co.MaxClockOffsetMHz), nil
}

func (g *NvidiaGpu) GetCoLimMem() (int, int, error) {
	// fallback
	if g.symbols.DeviceGetClockOffsets == nil {
		var min, max int32
		if ret := g.symbols.DeviceGetMemClkMinMaxVfOffset(g.handle, &min, &max); ret != SUCCESS {
			return 0, 0, fmt.Errorf(g.symbols.StringFromReturn(ret))
		}
		return int(min), int(max), nil
	}

	var co ClockOffset
	co.Version = VERSION_CLOCK_OFFSET
	co.Type = CLOCK_MEM
	co.Pstate = PSTATE_0
	if ret := g.symbols.DeviceGetClockOffsets(g.handle, &co); ret != SUCCESS {
		return 0, 0, fmt.Errorf(g.symbols.StringFromReturn(ret))
	}
	return int(co.MinClockOffsetMHz), int(co.MaxClockOffsetMHz), nil
}

func (g *NvidiaGpu) GetClLimGpu() (int, int, error) {
	return g.getClLimGpuV2()
}

func (g *NvidiaGpu) CanSetPl() bool {
	var limit uint32

	// pl are controlled by vbios/hardware on some laptop GPUs (and getter always fails)
	// setter still succeeds but does nothing
	return g.symbols.DeviceGetPowerManagementLimit != nil &&
		g.symbols.DeviceSetPowerManagementLimit != nil &&
		g.symbols.DeviceGetPowerManagementLimit(g.handle, &limit) == SUCCESS
}

func (g *NvidiaGpu) SetPl(mw int) error {
	if !g.CanSetPl() {
		return errors.New("controlled by vbios/hardware")
	}
	if ret := g.symbols.DeviceSetPowerManagementLimit(g.handle, uint32(mw)); ret != SUCCESS {
		return fmt.Errorf(g.symbols.StringFromReturn(ret))
	}
	return nil
}

func (g *NvidiaGpu) SetCoGpu(mhz int) error {
	// fallback
	if g.symbols.DeviceSetClockOffsets == nil {
		if ret := g.symbols.DeviceSetGpcClkVfOffset(g.handle, int32(mhz)); ret != SUCCESS {
			return fmt.Errorf(g.symbols.StringFromReturn(ret))
		}
		return nil
	}

	var co ClockOffset
	co.Version = VERSION_CLOCK_OFFSET
	co.Type = CLOCK_GRAPHICS
	co.Pstate = PSTATE_0
	co.ClockOffsetMHz = int32(mhz)
	if ret := g.symbols.DeviceSetClockOffsets(g.handle, &co); ret != SUCCESS {
		return fmt.Errorf(g.symbols.StringFromReturn(ret))
	}
	return nil
}

func (g *NvidiaGpu) SetCoMem(mhz int) error {
	// fallback
	if g.symbols.DeviceSetClockOffsets == nil {
		if ret := g.symbols.DeviceSetMemClkVfOffset(g.handle, int32(mhz)); ret != SUCCESS {
			return fmt.Errorf(g.symbols.StringFromReturn(ret))
		}
		return nil
	}

	var co ClockOffset
	co.Version = VERSION_CLOCK_OFFSET
	co.Type = CLOCK_MEM
	co.Pstate = PSTATE_0
	co.ClockOffsetMHz = int32(mhz)
	if ret := g.symbols.DeviceSetClockOffsets(g.handle, &co); ret != SUCCESS {
		return fmt.Errorf(g.symbols.StringFromReturn(ret))
	}
	return nil
}

func (g *NvidiaGpu) SetClGpu(mhz int) error {
	if ret := g.symbols.DeviceSetGpuLockedClocks(g.handle, 0, uint32(mhz)); ret != SUCCESS {
		return fmt.Errorf(g.symbols.StringFromReturn(ret))
	}
	return nil
}

func (g *NvidiaGpu) ResetPl() error {
	if !g.CanSetPl() {
		return errors.New("controlled by vbios/hardware")
	}
	defaultPl, err := g.GetPl()
	if err != nil {
		return err
	}
	return g.SetPl(defaultPl)
}

func (g *NvidiaGpu) ResetCoGpu() error {
	return g.SetCoGpu(0)
}

func (g *NvidiaGpu) ResetCoMem() error {
	return g.SetCoMem(0)
}

func (g *NvidiaGpu) ResetClGpu() error {
	if ret := g.symbols.DeviceResetGpuLockedClocks(g.handle); ret != SUCCESS {
		return fmt.Errorf(g.symbols.StringFromReturn(ret))
	}
	return nil
}

func (g *NvidiaGpu) getSupportedMemClocks() ([]int, error) {
	var count uint32
	ret := g.symbols.DeviceGetSupportedMemoryClocks(g.handle, &count, nil)
	if ret != SUCCESS && ret != ERROR_INSUFFICIENT_SIZE {
		return nil, fmt.Errorf("failed to get supported mem clock count: %s", g.symbols.StringFromReturn(ret))
	}

	if count == 0 {
		return []int{}, nil
	}

	clocks := make([]uint32, count)
	ret = g.symbols.DeviceGetSupportedMemoryClocks(g.handle, &count, &clocks[0])
	if ret != SUCCESS {
		return nil, fmt.Errorf("failed to fetch supported mem clocks: %s", g.symbols.StringFromReturn(ret))
	}

	res := make([]int, count)
	for i, v := range clocks {
		res[i] = int(v)
	}
	return res, nil
}

func (g *NvidiaGpu) getSupportedGpuClocks(memClockMHz int) ([]int, error) {
	var count uint32
	ret := g.symbols.DeviceGetSupportedGraphicsClocks(g.handle, uint32(memClockMHz), &count, nil)
	if ret != SUCCESS && ret != ERROR_INSUFFICIENT_SIZE {
		return nil, fmt.Errorf("failed to get gpu clock count: %s", g.symbols.StringFromReturn(ret))
	}

	if count == 0 {
		return []int{}, nil
	}

	clocks := make([]uint32, count)
	ret = g.symbols.DeviceGetSupportedGraphicsClocks(g.handle, uint32(memClockMHz), &count, &clocks[0])
	if ret != SUCCESS {
		return nil, fmt.Errorf("failed to fetch gpu clocks: %s", g.symbols.StringFromReturn(ret))
	}

	res := make([]int, count)
	for i, v := range clocks {
		res[i] = int(v)
	}
	return res, nil
}

func (g *NvidiaGpu) getClLimGpuV1() (int, int, error) {
	// a faster way, which ignores min cl lim

	var max uint32
	if ret := g.symbols.DeviceGetMaxClockInfo(g.handle, CLOCK_GRAPHICS, &max); ret != SUCCESS {
		return 0, 0, fmt.Errorf("failed to get max clock: %s", g.symbols.StringFromReturn(ret))
	}
	co, err := g.GetCoGpu()
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get current co: %w", err)
	}
	return 0, int(max) - co, nil
}

func (g *NvidiaGpu) getClLimGpuV2() (int, int, error) {
	// see also: nvidia-smi -q -d SUPPORTED_CLOCKS

	memClocks, err := g.getSupportedMemClocks()
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get supported mem clocks: %w", err)
	}
	if len(memClocks) == 0 {
		return 0, 0, errors.New("no supported mem clocks")
	}

	minMem := memClocks[0]
	maxMem := memClocks[0]
	for _, m := range memClocks {
		if m < minMem {
			minMem = m
		}
		if m > maxMem {
			maxMem = m
		}
	}

	minGpuClocks, err := g.getSupportedGpuClocks(minMem)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get gpu clocks for min mem %d: %v", minMem, err)
	}
	if len(minGpuClocks) == 0 {
		return 0, 0, fmt.Errorf("no gpu clocks found for min mem %d", minMem)
	}
	maxGpuClocks, err := g.getSupportedGpuClocks(maxMem)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get gpu clocks for max mem %d: %v", maxMem, err)
	}
	if len(maxGpuClocks) == 0 {
		return 0, 0, fmt.Errorf("no gpu clocks found for max mem %d", maxMem)
	}

	minGpu := minGpuClocks[0]
	for _, g := range minGpuClocks {
		if g < minGpu {
			minGpu = g
		}
	}
	maxGpu := maxGpuClocks[0]
	for _, g := range maxGpuClocks {
		if g > maxGpu {
			maxGpu = g
		}
	}

	return minGpu, maxGpu, nil
}
