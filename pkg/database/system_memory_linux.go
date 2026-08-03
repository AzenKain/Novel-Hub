// go:build linux
// +build linux

package database

import (
	"os"
	"strconv"
	"strings"
)

func cgroupMemoryLimit() int64 {
	// Check cgroups v2
	if data, err := os.ReadFile("/sys/fs/cgroup/memory.max"); err == nil {
		str := strings.TrimSpace(string(data))
		if str != "max" && str != "" {
			if limit, err := strconv.ParseInt(str, 10, 64); err == nil && limit > 0 {
				return limit
			}
		}
	}

	// Check cgroups v1
	if data, err := os.ReadFile("/sys/fs/cgroup/memory/memory.limit_in_bytes"); err == nil {
		str := strings.TrimSpace(string(data))
		if str != "" {
			if limit, err := strconv.ParseInt(str, 10, 64); err == nil && limit > 0 {
				// cgroups v1 sets a huge number (~9EB) when unlimited
				if limit < (1 << 60) {
					return limit
				}
			}
		}
	}

	return 0
}

func systemMemoryBytes() int64 {
	var total int64

	data, err := os.ReadFile("/proc/meminfo")
	if err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			fields := strings.Fields(line)
			if len(fields) < 2 || fields[0] != "MemTotal:" {
				continue
			}
			if kib, err := strconv.ParseInt(fields[1], 10, 64); err == nil {
				total = kib * 1024
			}
			break
		}
	}

	cgroupLimit := cgroupMemoryLimit()
	if cgroupLimit > 0 && (total == 0 || cgroupLimit < total) {
		return cgroupLimit
	}

	return total
}
