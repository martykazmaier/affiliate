//go:build linux

package main

import (
	"os"
	"time"

	"golang.org/x/sys/unix"
)

func hideConsole() {}

func enableConsoleVT() (restore func()) {
	fd := int(os.Stdin.Fd())
	old, err := unix.IoctlGetTermios(fd, unix.TCGETS)
	if err != nil {
		return func() {}
	}
	raw := *old
	raw.Iflag &^= unix.IGNBRK | unix.BRKINT | unix.PARMRK | unix.ISTRIP | unix.INLCR | unix.IGNCR | unix.ICRNL | unix.IXON
	raw.Oflag &^= unix.OPOST
	raw.Lflag &^= unix.ECHO | unix.ECHONL | unix.ICANON | unix.ISIG | unix.IEXTEN
	raw.Cflag &^= unix.CSIZE | unix.PARENB
	raw.Cflag |= unix.CS8
	raw.Cc[unix.VMIN] = 1
	raw.Cc[unix.VTIME] = 0
	if err := unix.IoctlSetTermios(fd, unix.TCSETS, &raw); err != nil {
		return func() {}
	}
	return func() {
		_ = unix.IoctlSetTermios(fd, unix.TCSETS, old)
	}
}

func (c *consoleConn) Read(b []byte) (int, error) {
	fd := int(os.Stdin.Fd())
	timeout := -1
	if !c.rdl.IsZero() {
		ms := time.Until(c.rdl).Milliseconds()
		if ms <= 0 {
			return 0, os.ErrDeadlineExceeded
		}
		timeout = int(ms)
	}
	pfd := []unix.PollFd{{Fd: int32(fd), Events: unix.POLLIN}}
	n, err := unix.Poll(pfd, timeout)
	if err != nil {
		if err == unix.EINTR {
			return 0, nil
		}
		return 0, err
	}
	if n == 0 {
		return 0, os.ErrDeadlineExceeded
	}
	return unix.Read(fd, b)
}
