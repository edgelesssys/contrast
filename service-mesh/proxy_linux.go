//go:build linux

package main

import "syscall"

const (
	SOL_IP          = syscall.SOL_IP
	IP_TRANSPARENT  = syscall.IP_TRANSPARENT
	SO_ORIGINAL_DST = 80
)
