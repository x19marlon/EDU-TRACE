package handler

import (
	"syscall"
	"unsafe"
)

var getDiskFreeSpaceEx = syscall.NewLazyDLL("kernel32.dll").NewProc("GetDiskFreeSpaceExW")

func freePercent(dir string) (float64, error) {
	path, err := syscall.UTF16PtrFromString(dir)
	if err != nil {
		return 0, err
	}
	var available, total, free uint64
	ok, _, callErr := getDiskFreeSpaceEx.Call(
		uintptr(unsafe.Pointer(path)),
		uintptr(unsafe.Pointer(&available)),
		uintptr(unsafe.Pointer(&total)),
		uintptr(unsafe.Pointer(&free)),
	)
	if ok == 0 {
		return 0, callErr
	}
	if total == 0 {
		return 0, nil
	}
	return 100 * float64(available) / float64(total), nil
}
