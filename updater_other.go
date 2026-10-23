//go:build !windows

package main

import "syscall"

func updaterSysProcAttr() *syscall.SysProcAttr { return nil }
