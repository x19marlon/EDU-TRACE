//go:build !windows

package handler

import "syscall"

func freePercent(dir string) (float64, error) {
	var st syscall.Statfs_t
	if err := syscall.Statfs(dir, &st); err != nil {
		return 0, err
	}
	if st.Blocks == 0 {
		return 0, nil
	}
	return 100 * float64(st.Bavail) / float64(st.Blocks), nil
}
