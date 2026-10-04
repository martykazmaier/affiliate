//go:build windows

package main

import (
	"syscall"

	"golang.org/x/sys/windows"
)

func inheritSocketAttr(s *eleSocket) *syscall.SysProcAttr {
	_ = windows.SetHandleInformation(s.h, windows.HANDLE_FLAG_INHERIT, windows.HANDLE_FLAG_INHERIT)
	return &syscall.SysProcAttr{
		HideWindow:                 true,
		AdditionalInheritedHandles: []syscall.Handle{syscall.Handle(s.h)},
	}
}

func inheritLocalAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{HideWindow: false}
}
