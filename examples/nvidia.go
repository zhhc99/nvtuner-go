package main

import (
	"fmt"
	"log"

	"nvtuner-go/internal/driver/nvidia"
	"nvtuner-go/internal/gpu"
)

func main() {
	drv, err := nvidia.New()
	if err != nil {
		log.Fatalf("Failed to load NVML: %v", err)
	}
	if err := drv.Init(); err != nil {
		log.Fatalf("Failed to initialize NVML: %v", err)
	}
	defer drv.Shutdown()
	fmt.Printf("%s %s | Driver %s\n", drv.GetManagerName(), drv.GetManagerVersion(), drv.GetDriverVersion())

	devices, err := drv.Devices()
	if err != nil {
		log.Fatalf("Failed to enumerate devices: %v", err)
	}
	for i, dev := range devices {
		printDeviceInfo(i, dev)
	}
}

func printDeviceInfo(index int, dev gpu.Device) {
	gUtil, mUtil, _ := dev.GetUtil()
	gClk, mClk, _ := dev.GetClocks()
	temp, _ := dev.GetTemperature()

	tMem, _, uMem, _ := dev.GetMemory()
	mTotalGi := float64(tMem) / 1024 / 1024 / 1024
	mUsedGi := float64(uMem) / 1024 / 1024 / 1024

	fanPct, fanRpm, _ := dev.GetFanSpeed()

	pCurr, _ := dev.GetPower()
	pLim, _ := dev.GetPl()
	pMin, pMax, _ := dev.GetPlLim()

	coCurr, _ := dev.GetCoGpu()
	coMin, coMax, _ := dev.GetCoLimGpu()

	// 格式说明:
	// %d: 索引
	// %4.1f%%: 占用率 (如  0.9%)
	// %4dMHz: 频率 (如 1892MHz)
	// %2d°C: 温度
	// %4.1fG: 显存 (如  1.2G)
	// %2d%%: 风扇百分比
	// %4d: 风扇转速
	// %3dW: 功耗
	// %+4dMHz: 频率偏移 (带符号, 如 +150MHz)
	fmt.Printf("%d[%4.1f%% %4dMHz %2d°C][%4.1f%% %4dMHz %4.1fG/%4.1fG][%2d%% %4dRPM][%3dW/%3dW|(%3dW,%3dW)][%+4dMHz|(%+4d,%+4d)]\n",
		index,
		float64(gUtil), gClk, temp,
		float64(mUtil), mClk, mUsedGi, mTotalGi,
		fanPct, fanRpm,
		pCurr/1000, pLim/1000, pMin/1000, pMax/1000,
		coCurr, coMin, coMax,
	)
}
