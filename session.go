package main

import (
	"errors"
	"io"
	"net"
	"os"
	"time"
	"unicode/utf8"
)

const (
	keyUp    = 0x1001
	keyDown  = 0x1002
	keyEnter = 0x1003
	keyEsc   = 0x1004
	keyPgUp  = 0x1005
	keyPgDn  = 0x1006
	keyHome  = 0x1007
	keyEnd   = 0x1008
)

type session struct {
	rw      io.ReadWriter
	sock    *eleSocket
	iac     int
	iacCmd  byte
	app     []byte
	restore func()
}

func newSession(rw io.ReadWriter, sock *eleSocket, local bool) *session {
	s := &session{rw: rw, sock: sock}
	if local {
		s.restore = enableConsoleVT()
	} else {
		hideConsole()
		s.takeEcho()
	}
	return s
}

func (s *session) Close() {
	if s.restore != nil {
		s.restore()
	}
}

func (s *session) resetInput() {
	s.app = nil
	s.iac = 0
	s.iacCmd = 0
}

func (s *session) takeEcho() {
	// Server WILL ECHO: client must turn off local echo so lightbar keys
	// and form input are not printed twice.
	_, _ = s.Write([]byte{
		255, 251, 1, // IAC WILL ECHO
		255, 251, 3, // IAC WILL SGA
		255, 253, 3, // IAC DO SGA
		255, 251, 0, // IAC WILL BINARY
		255, 253, 0, // IAC DO BINARY
	})
}

func (s *session) Write(p []byte) (int, error) {
	return s.rw.Write(p)
}

func (s *session) print(str string) {
	_, _ = s.Write([]byte(str))
}

func (s *session) readByte(d time.Duration) (byte, error) {
	if len(s.app) > 0 {
		b := s.app[0]
		s.app = s.app[1:]
		return b, nil
	}
	if setter, ok := s.rw.(interface{ SetReadDeadline(time.Time) error }); ok {
		if d > 0 {
			_ = setter.SetReadDeadline(time.Now().Add(d))
		} else {
			_ = setter.SetReadDeadline(time.Time{})
		}
		defer setter.SetReadDeadline(time.Time{})
	}

	buf := make([]byte, 64)
	for {
		n, err := s.rw.Read(buf)
		if n > 0 {
			s.pushRaw(buf[:n])
			if len(s.app) > 0 {
				b := s.app[0]
				s.app = s.app[1:]
				return b, nil
			}
			continue
		}
		if err != nil {
			if d > 0 && errors.Is(err, os.ErrDeadlineExceeded) {
				return 0, err
			}
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				return 0, os.ErrDeadlineExceeded
			}
			return 0, err
		}
	}
}

func (s *session) pushRaw(p []byte) {
	for _, b := range p {
		s.feedTelnet(b)
	}
}

func (s *session) feedTelnet(b byte) {
	const (
		iac  = 255
		will = 251
		wont = 252
		do   = 253
		dont = 254
		sb   = 250
		se   = 240
		echo = 1
		sga  = 3
		bin  = 0
	)
	switch s.iac {
	case 0:
		if b == iac {
			s.iac = 1
			return
		}
		s.app = append(s.app, b)
	case 1:
		switch b {
		case iac:
			s.app = append(s.app, iac)
			s.iac = 0
		case will, wont, do, dont:
			s.iacCmd = b
			s.iac = 2
		case sb:
			s.iac = 3
		default:
			s.iac = 0
		}
	case 2:
		cmd, opt := s.iacCmd, b
		s.iac = 0
		switch cmd {
		case do:
			if opt == echo || opt == sga || opt == bin {
				_, _ = s.Write([]byte{iac, will, opt})
			} else {
				_, _ = s.Write([]byte{iac, wont, opt})
			}
		case will:
			if opt == sga || opt == bin {
				_, _ = s.Write([]byte{iac, do, opt})
			} else {
				// DONT ECHO — the door owns echoing (readLine only).
				_, _ = s.Write([]byte{iac, dont, opt})
			}
		}
	case 3:
		if b == iac {
			s.iac = 4
		}
	case 4:
		if b == se {
			s.iac = 0
		} else if b != iac {
			s.iac = 3
		}
	}
}

func (s *session) readKey() (int, error) {
	b, err := s.readByte(0)
	if err != nil {
		return 0, err
	}
	switch b {
	case 13, 10:
		return keyEnter, nil
	case 27:
		n1, err := s.readByte(180 * time.Millisecond)
		if err != nil {
			return keyEsc, nil
		}
		if n1 == '[' || n1 == 'O' {
			n2, err := s.readByte(180 * time.Millisecond)
			if err != nil {
				return keyEsc, nil
			}
			switch n2 {
			case 'A':
				return keyUp, nil
			case 'B':
				return keyDown, nil
			case 'H':
				return keyHome, nil
			case 'F':
				return keyEnd, nil
			case '5':
				_, _ = s.readByte(80 * time.Millisecond)
				return keyPgUp, nil
			case '6':
				_, _ = s.readByte(80 * time.Millisecond)
				return keyPgDn, nil
			}
			return 0, nil
		}
		return keyEsc, nil
	case 8, 127:
		return 8, nil
	}
	return int(b), nil
}

func (s *session) readLine(max int) (string, bool, error) {
	var buf []byte
	s.print("\x1b[?25h")
	defer s.print("\x1b[?25l")
	for {
		k, err := s.readKey()
		if err != nil {
			return "", false, err
		}
		switch k {
		case keyEsc:
			return "", false, nil
		case keyEnter:
			s.print("\r\n")
			return string(buf), true, nil
		case 8:
			if len(buf) == 0 {
				continue
			}
			_, size := utf8.DecodeLastRune(buf)
			buf = buf[:len(buf)-size]
			s.print("\x08 \x08")
		default:
			if k < 32 || k > 0x10ff {
				continue
			}
			if len(buf) >= max {
				continue
			}
			r := rune(k)
			ch := string(r)
			buf = append(buf, ch...)
			s.print(ch)
		}
	}
}

func (s *session) cls() {
	s.print("\x1b[0;37;44m\x1b[?7l\x1b[2J\x1b[H\x1b[?25l")
}
