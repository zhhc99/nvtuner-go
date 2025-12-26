package nvidia

import (
	"fmt"
	"nvtuner-go/internal/gpu"
	"strings"
)

var _ gpu.Manager = (*NvidiaDriver)(nil)

type NvidiaDriver struct {
	s *RawSymbols
}

func New() (*NvidiaDriver, error) {
	s, err := NewRawSymbols()
	if err != nil {
		return nil, err
	}
	return &NvidiaDriver{s: s}, nil
}

func (d *NvidiaDriver) Init() error {
	if ret := d.s.Init_v2(); ret != SUCCESS {
		return fmt.Errorf("nvml init failed: %s", d.s.StringFromReturn(ret))
	}
	return nil
}

func (d *NvidiaDriver) Shutdown() error {
	if d.s.Shutdown != nil {
		d.s.Shutdown()
	}
	return nil
}

func (d *NvidiaDriver) GetManagerName() string {
	return "NVML"
}

func (d *NvidiaDriver) GetManagerVersion() string {
	var buf [SYSTEM_NVML_VERSION_BUFFER_SIZE]byte
	if ret := d.s.SystemGetNVMLVersion(&buf[0], SYSTEM_NVML_VERSION_BUFFER_SIZE); ret != SUCCESS {
		return "Unknown"
	}
	return string(buf[:len(strings.TrimRight(string(buf[:]), "\x00"))])
}

func (d *NvidiaDriver) GetDriverVersion() string {
	var buf [SYSTEM_DRIVER_VERSION_BUFFER_SIZE]byte
	if ret := d.s.SystemGetDriverVersion(&buf[0], SYSTEM_DRIVER_VERSION_BUFFER_SIZE); ret != SUCCESS {
		return "Unknown"
	}
	return string(buf[:len(strings.TrimRight(string(buf[:]), "\x00"))])
}

func (d *NvidiaDriver) Devices() ([]gpu.Device, error) {
	var count uint32
	if ret := d.s.DeviceGetCount_v2(&count); ret != SUCCESS {
		return nil, fmt.Errorf("get device count failed: %s", d.s.StringFromReturn(ret))
	}

	var res []gpu.Device
	for i := uint32(0); i < count; i++ {
		var handle Device
		if ret := d.s.DeviceGetHandleByIndex_v2(i, &handle); ret != SUCCESS {
			continue
		}

		g := &NvidiaGpu{handle: handle, symbols: d.s}
		g.name = g.fetchName()
		g.uuid = g.fetchUUID()
		res = append(res, g)
	}
	return res, nil
}
