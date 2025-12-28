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
	GetIndex() int
	GetName() string
	GetUUID() string
	GetUtil() (int, int, error)        // gpu, mem
	GetClocks() (int, int, error)      // gpu, mem; MHz
	GetMemory() (int, int, int, error) // total, free, used; Byte
	GetPower() (int, error)            // mW
	GetTemperature() (int, error)      // celsius
	GetFanSpeed() (int, int, error)    // %, rpm

	GetPl() (int, error)            // mW
	GetCoGpu() (int, error)         // MHz
	GetCoMem() (int, error)         // MHz
	GetClGpu() (int, error)         // MHz
	GetPlLim() (int, int, error)    // min, max
	GetCoLimGpu() (int, int, error) // min, max
	GetCoLimMem() (int, int, error) // min, max
	GetClLimGpu() (int, int, error) // min, max

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

type MState struct {
	ManagerName    string
	ManagerVersion string
	DriverVersion  string
}

type DState struct {
	Index    int
	Name     string
	UUID     string
	UtilGpu  int // %
	UtilMem  int // %
	Temp     int // Celsius
	FanPct   int // %
	FanRPM   int // RPM
	Power    int // mW
	PowerLim int // mW
	ClockGpu int // MHz
	ClockMem int // MHz
	MemTotal int // Byte
	MemUsed  int // Byte
	CoGpu    int // MHz
	CoMem    int // MHz
	ClGpu    int // MHz
	Limits   Limits
}

type Limits struct {
	PlMin, PlMax       int
	CoGpuMin, CoGpuMax int
	CoMemMin, CoMemMax int
	ClGpuMin, ClGpuMax int
}

func (m *MState) FetchOnce(mgr Manager) {
	m.ManagerName = mgr.GetManagerName()
	m.ManagerVersion = mgr.GetManagerVersion()
	m.DriverVersion = mgr.GetDriverVersion()
}

func (d *DState) FetchOnce(dev Device) {
	d.UtilGpu, d.UtilMem, _ = dev.GetUtil()
	d.Temp, _ = dev.GetTemperature()
	d.FanPct, d.FanRPM, _ = dev.GetFanSpeed()
	d.Power, _ = dev.GetPower()
	d.PowerLim, _ = dev.GetPl()
	d.ClockGpu, d.ClockMem, _ = dev.GetClocks()
	d.MemTotal, _, d.MemUsed, _ = dev.GetMemory()
	d.CoGpu, _ = dev.GetCoGpu()
	d.CoMem, _ = dev.GetCoMem()
	d.ClGpu, _ = dev.GetClGpu()
	d.Limits.PlMin, d.Limits.PlMax, _ = dev.GetPlLim()
	d.Limits.CoGpuMin, d.Limits.CoGpuMax, _ = dev.GetCoLimGpu()
	d.Limits.CoMemMin, d.Limits.CoMemMax, _ = dev.GetCoLimMem()
	d.Limits.ClGpuMin, d.Limits.ClGpuMax, _ = dev.GetClLimGpu()
}
