// go:build !linux && !windows
// +build !linux,!windows

package database

func systemMemoryBytes() int64 {
	return 0
}
