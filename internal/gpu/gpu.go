package gpu

type Manager interface {
	Init() error
	Shutdown() error
	Devices() ([]Device, error)
	GetManagerName() string // e.g. "NVML", "OneAPI"
	GetManagerVersion() string
	GetDriverVersion() string
}

type Device interface {
	GetName() string
	GetUUID() string
	GetUtil() (int, int, error)        // gpu, mem
	GetClocks() (int, int, error)      // gpu, mem; MHz
	GetMemory() (int, int, int, error) // total, free, used
	GetPower() (int, error)            // mW
	GetTemperature() (int, error)      // celsius
	GetFanSpeed() (int, int, error)    // %, rpm

	GetPl() (int, error)            // mW
	GetCoGpu() (int, error)         // MHz
	GetCoMem() (int, error)         // MHz
	GetPlLim() (int, int, error)    // min, max
	GetCoLimGpu() (int, int, error) // min, max
	GetCoLimMem() (int, int, error) // min, max

	CanSetPl() bool
	SetPl(int) error    // mW
	SetCoGpu(int) error // MHz
	SetCoMem(int) error // MHz
	SetClGpu(int) error // set clock limit; MHz
	ResetPl() error
	ResetCoGpu() error
	ResetCoMem() error
	ResetClGpu() error
}

type State struct {
	Index    int
	Name     string
	UUID     string
	GpuUtil  int     // %
	MemUtil  int     // %
	Temp     int     // Celsius
	FanPct   int     // %
	FanRPM   int     // RPM
	Power    int     // mW
	PowerLim int     // mW
	ClockGpu int     // MHz
	ClockMem int     // MHz
	MemTotal float64 // GiB
	MemUsed  float64 // GiB
	CoGpu    int     // MHz
	CoMem    int     // MHz
	Limits   Limits
}

type Limits struct {
	PlMin, PlMax       int
	CoGpuMin, CoGpuMax int
	CoMemMin, CoMemMax int
}
