//go:build !linux

package main

const (
	SOL_IP          = 0
	IP_TRANSPARENT  = 1337 // TODO
	SO_ORIGINAL_DST = 80
)
