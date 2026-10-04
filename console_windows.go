//go:build windows

package main

import (
	"os"
	"time"

	"golang.org/x/sys/windows"
)

func hideConsole() {
	kernel32 := windows.NewLazySystemDLL("kernel32.dll")
	user32 := windows.NewLazySystemDLL("user32.dll")
	getConsoleWindow := kernel32.NewProc("GetConsoleWindow")
	showWindow := user32.NewProc("ShowWindow")
	hwnd, _, _ := getConsoleWindow.Call()
	if hwnd != 0 {
		showWindow.Call(hwnd, uintptr(windows.SW_HIDE))
	}
}

func enableConsoleVT() (restore func()) {
	hin := windows.Handle(os.Stdin.Fd())
	hout := windows.Handle(os.Stdout.Fd())
	var inMode, outMode uint32
	_ = windows.GetConsoleMode(hin, &inMode)
	_ = windows.GetConsoleMode(hout, &outMode)

	raw := inMode
	raw &^= windows.ENABLE_ECHO_INPUT | windows.ENABLE_LINE_INPUT | windows.ENABLE_PROCESSED_INPUT | windows.ENABLE_QUICK_EDIT_MODE
	raw |= windows.ENABLE_VIRTUAL_TERMINAL_INPUT | windows.ENABLE_EXTENDED_FLAGS
	_ = windows.SetConsoleMode(hin, raw)

	om := outMode | windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING | windows.ENABLE_PROCESSED_OUTPUT
	_ = windows.SetConsoleMode(hout, om)

	return func() {
		_ = windows.SetConsoleMode(hin, inMode)
		_ = windows.SetConsoleMode(hout, outMode)
	}
}

func (c *consoleConn) Read(b []byte) (int, error) {
	h := windows.Handle(os.Stdin.Fd())
	timeout := uint32(windows.INFINITE)
	if !c.rdl.IsZero() {
		ms := time.Until(c.rdl).Milliseconds()
		if ms <= 0 {
			return 0, os.ErrDeadlineExceeded
		}
		timeout = uint32(ms)
	}
	st, err := windows.WaitForSingleObject(h, timeout)
	if err != nil {
		return 0, err
	}
	if st == uint32(windows.WAIT_TIMEOUT) {
		return 0, os.ErrDeadlineExceeded
	}
	return os.Stdin.Read(b)
}
