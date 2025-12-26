package nvidia

import "unsafe"

// nvmlStructVersion implements NVML_STRUCT_VERSION
func nvmlStructVersion(size uintptr, ver uint32) uint32 {
	return uint32(size) | (ver << 24)
}

func (v Value) AsUint() uint32 {
	return *(*uint32)(unsafe.Pointer(&v.Data))
}
