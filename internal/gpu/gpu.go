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
	GetFanSpeed() (int, int, error)    // percent, rpm

	GetCoGpu() (int, error)         // MHz
	GetCoMem() (int, error)         // MHz
	GetPl() (int, error)            // mW
	GetCoLimGpu() (int, int, error) // min, max
	GetCoLimMem() (int, int, error) // min, max
	GetPlLim() (int, int, error)    // min, max

	CanSetPl() bool
	SetPl(int) error    // mW
	SetCoGpu(int) error // MHz
	SetCoMem(int) error // MHz
	SetClGpu(int) error // set clock limit; MHz
}
