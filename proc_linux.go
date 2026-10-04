//go:build linux

package main

import "syscall"

func inheritSocketAttr(s *eleSocket) *syscall.SysProcAttr {
	return &syscall.SysProcAttr{}
}

func inheritLocalAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{}
}
