//go:build unix

package source

import "syscall"

const openNonblock = syscall.O_NONBLOCK
