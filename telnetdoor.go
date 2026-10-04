package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func launchTelnetDoor(s *session, telnetPath, address, dropPath string, sock *eleSocket) error {
	if strings.TrimSpace(telnetPath) == "" {
		s.print(crlf + colRed + "  SysOp: set -T to the telnetdoor path." + colReset + crlf)
		return s.pause("Hit any key")
	}
	if strings.TrimSpace(address) == "" {
		s.print(crlf + colRed + "  That entry has no telnet address." + colReset + crlf)
		return s.pause("Hit any key")
	}
	if _, err := os.Stat(telnetPath); err != nil {
		s.print(crlf + colRed + "  Cannot find telnetdoor: " + telnetPath + colReset + crlf)
		return s.pause("Hit any key")
	}

	s.cls()
	s.print(colYel + crlf + "  Connecting to " + colBWhi + address + colYel + " ..." + colReset + crlf)
	time.Sleep(200 * time.Millisecond)

	args := []string{
		"-S" + strings.TrimSpace(address),
		"-D" + dropPath,
		"-W0",
	}
	cmd := exec.Command(telnetPath, args...)
	cmd.Dir = filepath.Dir(telnetPath)
	if sock != nil {
		sock.PrepareChild()
		defer sock.ResumeIO()
		cmd.SysProcAttr = inheritSocketAttr(sock)
	} else {
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.SysProcAttr = inheritLocalAttr()
	}

	s.resetInput()
	s.print("\x1b[?25h")
	err := cmd.Run()
	s.print("\x1b[?25l")
	s.resetInput()
	if err != nil {
		s.print(crlf + colRed + "  TelnetDoor ended: " + err.Error() + colReset + crlf)
		_ = s.pause("Hit any key")
	}
	return nil
}
