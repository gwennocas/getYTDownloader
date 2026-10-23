//go:build windows

package main

import "syscall"

func updaterSysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{CreationFlags: 0x08000000}
}
