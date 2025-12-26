package nvidia

import (
	"bytes"
	"fmt"
	"nvtuner-go/internal/libloader"
	"unsafe"
)

var (
	VERSION_CLOCK_OFFSET = nvmlStructVersion(unsafe.Sizeof(ClockOffset{}), 1)
	VERSION_FAN_SPEED    = nvmlStructVersion(unsafe.Sizeof(FanSpeedInfo{}), 1)
	VERSION_TEMPERATURE  = nvmlStructVersion(unsafe.Sizeof(Temperature{}), 1)
)

type RawSymbols struct {
	// systems
	Init_v2                    func() Return
	SystemGetDriverVersion     func(buffer *byte, length uint32) Return // NVML_SYSTEM_DRIVER_VERSION_BUFFER_SIZE=80
	SystemGetNVMLVersion       func(buffer *byte, length uint32) Return // NVML_SYSTEM_NVML_VERSION_BUFFER_SIZE=80
	SystemGetCudaDriverVersion func(version *int32) Return
	Shutdown                   func() Return
	ErrorString                func(result Return) uintptr

	// find devices
	DeviceGetCount_v2         func(count *uint32) Return
	DeviceGetHandleByIndex_v2 func(index uint32, device *Device) Return
	DeviceGetUUID             func(device Device, buffer *byte, length uint32) Return // NVML_DEVICE_UUID_BUFFER_SIZE=80
	DeviceGetName             func(device Device, buffer *byte, length uint32) Return // NVML_DEVICE_NAME_BUFFER_SIZE=64

	// monitor
	DeviceGetUtilizationRates          func(device Device, util *Utilization) Return
	DeviceGetMemoryInfo                func(device Device, memory *Memory) Return
	DeviceGetClockInfo                 func(device Device, clockType ClockType, clock *uint32) Return
	DeviceGetPowerUsage                func(device Device, power *uint32) Return
	DeviceGetEnforcedPowerLimit        func(device Device, limit *uint32) Return
	DeviceGetTemperature               func(device Device, sensor TemperatureSensors, temp *uint32) Return
	DeviceGetTemperatureV              func(device Device, info *Temperature) Return
	DeviceGetFanSpeed                  func(device Device, speed *uint32) Return
	DeviceGetFanSpeedRPM               func(device Device, info *FanSpeedInfo) Return
	DeviceGetSamples                   func(device Device, samplingType SamplingType, lastSeen uint64, valType *ValueType, count *uint32, samples *Sample) Return
	DeviceGetCurrentClocksEventReasons func(device Device, reasons *uint64) Return

	// oc: power limits
	DeviceGetPowerManagementLimitConstraints func(device Device, min *uint32, max *uint32) Return
	DeviceGetPowerManagementDefaultLimit     func(device Device, limit *uint32) Return
	DeviceGetPowerManagementLimit            func(device Device, limit *uint32) Return
	DeviceSetPowerManagementLimit            func(device Device, limit uint32) Return

	// oc: clock offsets
	DeviceGetClockOffsets         func(device Device, info *ClockOffset) Return
	DeviceGetGpcClkMinMaxVfOffset func(device Device, min *int32, max *int32) Return // GetClockOffsetsLegacy
	DeviceGetMaxClockInfo         func(device Device, clockType ClockType, clock *uint32) Return
	DeviceSetClockOffsets         func(device Device, info *ClockOffset) Return
	DeviceSetGpcClkVfOffset       func(device Device, offset int32) Return // SetClockOffsetsLegacy

	// oc: locked clocks
	DeviceSetGpuLockedClocks   func(device Device, min uint32, max uint32) Return
	DeviceResetGpuLockedClocks func(device Device) Return
}

func NewRawSymbols() (*RawSymbols, error) {
	lib, err := libloader.Load(libloader.NVML)
	if err != nil {
		return nil, err
	}

	nvml := &RawSymbols{}

	libloader.Bind(lib, &nvml.Init_v2, "nvmlInit_v2")
	libloader.Bind(lib, &nvml.SystemGetDriverVersion, "nvmlSystemGetDriverVersion")
	libloader.Bind(lib, &nvml.SystemGetNVMLVersion, "nvmlSystemGetNVMLVersion")
	libloader.Bind(lib, &nvml.SystemGetCudaDriverVersion, "nvmlSystemGetCudaDriverVersion")
	libloader.Bind(lib, &nvml.Shutdown, "nvmlShutdown")
	libloader.Bind(lib, &nvml.ErrorString, "nvmlErrorString")

	libloader.Bind(lib, &nvml.DeviceGetCount_v2, "nvmlDeviceGetCount_v2")
	libloader.Bind(lib, &nvml.DeviceGetHandleByIndex_v2, "nvmlDeviceGetHandleByIndex_v2")
	libloader.Bind(lib, &nvml.DeviceGetUUID, "nvmlDeviceGetUUID")
	libloader.Bind(lib, &nvml.DeviceGetName, "nvmlDeviceGetName")

	libloader.Bind(lib, &nvml.DeviceGetPowerManagementLimitConstraints, "nvmlDeviceGetPowerManagementLimitConstraints")
	libloader.Bind(lib, &nvml.DeviceGetPowerManagementDefaultLimit, "nvmlDeviceGetPowerManagementDefaultLimit")
	libloader.Bind(lib, &nvml.DeviceGetPowerManagementLimit, "nvmlDeviceGetPowerManagementLimit")
	libloader.Bind(lib, &nvml.DeviceGetEnforcedPowerLimit, "nvmlDeviceGetEnforcedPowerLimit")
	libloader.Bind(lib, &nvml.DeviceSetPowerManagementLimit, "nvmlDeviceSetPowerManagementLimit")
	libloader.Bind(lib, &nvml.DeviceGetPowerUsage, "nvmlDeviceGetPowerUsage")

	libloader.Bind(lib, &nvml.DeviceGetMaxClockInfo, "nvmlDeviceGetMaxClockInfo")
	libloader.Bind(lib, &nvml.DeviceGetClockInfo, "nvmlDeviceGetClockInfo")
	libloader.Bind(lib, &nvml.DeviceSetGpuLockedClocks, "nvmlDeviceSetGpuLockedClocks")
	libloader.Bind(lib, &nvml.DeviceResetGpuLockedClocks, "nvmlDeviceResetGpuLockedClocks")

	libloader.Bind(lib, &nvml.DeviceGetClockOffsets, "nvmlDeviceGetClockOffsets")
	libloader.Bind(lib, &nvml.DeviceSetClockOffsets, "nvmlDeviceSetClockOffsets")
	libloader.Bind(lib, &nvml.DeviceGetGpcClkMinMaxVfOffset, "nvmlDeviceGetGpcClkMinMaxVfOffset")
	libloader.Bind(lib, &nvml.DeviceSetGpcClkVfOffset, "nvmlDeviceSetGpcClkVfOffset")

	libloader.Bind(lib, &nvml.DeviceGetFanSpeed, "nvmlDeviceGetFanSpeed")
	libloader.Bind(lib, &nvml.DeviceGetFanSpeedRPM, "nvmlDeviceGetFanSpeedRPM")
	libloader.Bind(lib, &nvml.DeviceGetTemperature, "nvmlDeviceGetTemperature")
	libloader.Bind(lib, &nvml.DeviceGetTemperatureV, "nvmlDeviceGetTemperatureV")
	libloader.Bind(lib, &nvml.DeviceGetMemoryInfo, "nvmlDeviceGetMemoryInfo")
	libloader.Bind(lib, &nvml.DeviceGetUtilizationRates, "nvmlDeviceGetUtilizationRates")
	libloader.Bind(lib, &nvml.DeviceGetSamples, "nvmlDeviceGetSamples")
	libloader.Bind(lib, &nvml.DeviceGetCurrentClocksEventReasons, "nvmlDeviceGetCurrentClocksEventReasons")

	return nvml, nil
}

func (s *RawSymbols) StringFromReturn(r Return) string {
	if s.ErrorString == nil {
		return fmt.Sprintf("NVML Error %d", r)
	}
	ptr := s.ErrorString(r)
	if ptr == 0 {
		return "Unknown NVML Error"
	}

	p := (*byte)(unsafe.Pointer(ptr))

	const maxLen = 1024

	str := unsafe.Slice(p, maxLen)
	n := bytes.IndexByte(str, 0)
	if n == -1 {
		n = maxLen
	}

	return string(str[:n])
}
