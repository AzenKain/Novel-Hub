// go:build windows
//go:build windows
// +build windows

package database

import (
	"syscall"
	"unsafe"
)

type memoryStatusEx struct {
	Length               uint32
	MemoryLoad           uint32
	TotalPhys            uint64
	AvailPhys            uint64
	TotalPageFile        uint64
	AvailPageFile        uint64
	TotalVirtual         uint64
	AvailVirtual         uint64
	AvailExtendedVirtual uint64
}

var (
	kernel32           = syscall.NewLazyDLL("kernel32.dll")
	globalMemoryStatus = kernel32.NewProc("GlobalMemoryStatusEx")
)

func systemMemoryBytes() int64 {
	var ms memoryStatusEx
	ms.Length = uint32(unsafe.Sizeof(ms))
	r, _, _ := globalMemoryStatus.Call(uintptr(unsafe.Pointer(&ms)))
	if r == 0 {
		return 0
	}
	return int64(ms.TotalPhys)
}
