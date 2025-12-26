//go:build windows

package libloader

import (
	"syscall"

	"github.com/ebitengine/purego"
)

func Load(lib string) (uintptr, error) {
	h, err := syscall.LoadLibrary(lib)
	return uintptr(h), err
}

func Bind(lib uintptr, ptr any, name string) {
	addr, _ := syscall.GetProcAddress(syscall.Handle(lib), name)
	if addr != 0 {
		purego.RegisterFunc(ptr, addr)
	}
}
