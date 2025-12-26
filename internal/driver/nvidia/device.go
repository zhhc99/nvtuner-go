package nvidia

// TODO: fall back (temperature / clkvf, etc. see nvtuner.cpp)

import (
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
	var temp uint32
	if ret := g.symbols.DeviceGetTemperature(g.handle, TEMPERATURE_GPU, &temp); ret != SUCCESS {
		return 0, fmt.Errorf(g.symbols.StringFromReturn(ret))
	}
	return int(temp), nil
}

func (g *NvidiaGpu) GetFanSpeed() (int, int, error) {
	var percent uint32
	var fan FanSpeedInfo
	fan.Version = VERSION_FAN_SPEED
	_ = g.symbols.DeviceGetFanSpeed(g.handle, &percent)
	_ = g.symbols.DeviceGetFanSpeedRPM(g.handle, &fan)
	return int(percent), int(fan.Speed), nil
}

func (g *NvidiaGpu) GetCo() (int, int, error) {
	var co ClockOffset
	co.Version = VERSION_CLOCK_OFFSET
	co.Type = CLOCK_GRAPHICS
	co.Pstate = PSTATE_0
	if ret := g.symbols.DeviceGetClockOffsets(g.handle, &co); ret != SUCCESS {
		return 0, 0, fmt.Errorf(g.symbols.StringFromReturn(ret))
	}
	return int(co.ClockOffsetMHz), 0, nil // TODO: memory co
}

func (g *NvidiaGpu) GetPl() (int, error) {
	var mw uint32
	if ret := g.symbols.DeviceGetEnforcedPowerLimit(g.handle, &mw); ret != SUCCESS {
		return 0, fmt.Errorf(g.symbols.StringFromReturn(ret))
	}
	return int(mw), nil
}

func (g *NvidiaGpu) GetCoLimGpu() (int, int, error) {
	var min, max int32
	if ret := g.symbols.DeviceGetGpcClkMinMaxVfOffset(g.handle, &min, &max); ret != SUCCESS {
		return 0, 0, fmt.Errorf(g.symbols.StringFromReturn(ret))
	}
	return int(min), int(max), nil
}

func (g *NvidiaGpu) GetCoLimMem() (int, int, error) {
	return 0, 0, fmt.Errorf("not implemented for memory via this api")
}

func (g *NvidiaGpu) GetPlLim() (int, int, error) {
	var min, max uint32
	if ret := g.symbols.DeviceGetPowerManagementLimitConstraints(g.handle, &min, &max); ret != SUCCESS {
		return 0, 0, fmt.Errorf(g.symbols.StringFromReturn(ret))
	}
	return int(min), int(max), nil
}

func (g *NvidiaGpu) SetPl(mw int) error {
	if ret := g.symbols.DeviceSetPowerManagementLimit(g.handle, uint32(mw)); ret != SUCCESS {
		return fmt.Errorf(g.symbols.StringFromReturn(ret))
	}
	return nil
}

func (g *NvidiaGpu) SetCo(mhz int) error {
	if ret := g.symbols.DeviceSetGpcClkVfOffset(g.handle, int32(mhz)); ret != SUCCESS {
		return fmt.Errorf(g.symbols.StringFromReturn(ret))
	}
	return nil
}

func (g *NvidiaGpu) SetCl(mhz int) error {
	if ret := g.symbols.DeviceSetGpuLockedClocks(g.handle, uint32(mhz), uint32(mhz)); ret != SUCCESS {
		return fmt.Errorf(g.symbols.StringFromReturn(ret))
	}
	return nil
}
