//go:build linux

package main

import (
	"fmt"
	"io"
	"net"
	"os"
	"sync"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

// eleSocket is a DOOR32 inherited Unix socket/file descriptor.
// Close does NOT close the fd — the BBS owns it and TelnetDoor must inherit
// the same descriptor number from door32.sys.
type eleSocket struct {
	fd     int
	mu     sync.Mutex
	rdl    time.Time
	wdl    time.Time
	local  net.Addr
	remote net.Addr
}

func openEleSocket(handle uintptr) (*eleSocket, error) {
	if handle == 0 {
		return nil, fmt.Errorf("DOOR32.SYS socket handle is 0")
	}
	fd := int(handle)

	flags, err := unix.FcntlInt(uintptr(fd), unix.F_GETFD, 0)
	if err != nil {
		return nil, fmt.Errorf("F_GETFD %d: %w", fd, err)
	}
	if _, err := unix.FcntlInt(uintptr(fd), unix.F_SETFD, flags&^unix.FD_CLOEXEC); err != nil {
		return nil, fmt.Errorf("clear FD_CLOEXEC: %w", err)
	}

	_ = unix.SetsockoptInt(fd, unix.IPPROTO_TCP, unix.TCP_NODELAY, 1)

	s := &eleSocket{fd: fd}
	s.local = linuxAddr(true, fd)
	s.remote = linuxAddr(false, fd)
	if err := unix.SetNonblock(fd, true); err != nil {
		return nil, fmt.Errorf("O_NONBLOCK: %w", err)
	}
	return s, nil
}

func linuxAddr(local bool, fd int) net.Addr {
	var sa unix.Sockaddr
	var err error
	if local {
		sa, err = unix.Getsockname(fd)
	} else {
		sa, err = unix.Getpeername(fd)
	}
	if err != nil {
		if local {
			return dummyAddr("local")
		}
		return dummyAddr("remote")
	}
	switch a := sa.(type) {
	case *unix.SockaddrInet4:
		return &net.TCPAddr{IP: net.IP(a.Addr[:]), Port: a.Port}
	case *unix.SockaddrInet6:
		return &net.TCPAddr{IP: net.IP(a.Addr[:]), Port: a.Port}
	default:
		return dummyAddr("unix")
	}
}

func (s *eleSocket) PrepareChild() {
	_ = unix.SetNonblock(s.fd, false)
	flags, err := unix.FcntlInt(uintptr(s.fd), unix.F_GETFD, 0)
	if err == nil {
		_, _ = unix.FcntlInt(uintptr(s.fd), unix.F_SETFD, flags&^unix.FD_CLOEXEC)
	}
}

func (s *eleSocket) ResumeIO() {
	_ = unix.SetNonblock(s.fd, true)
}

func (s *eleSocket) Read(b []byte) (int, error) {
	if len(b) == 0 {
		return 0, nil
	}
	deadline := s.readDeadline()
	for {
		n, err := s.recv(b)
		if n > 0 {
			return n, nil
		}
		if err == io.EOF {
			return 0, io.EOF
		}
		if err != nil && !isWouldBlock(err) {
			return 0, err
		}
		if lostCarrier(s.fd) {
			return 0, io.EOF
		}
		if !deadline.IsZero() && time.Now().After(deadline) {
			return 0, os.ErrDeadlineExceeded
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func (s *eleSocket) Write(b []byte) (int, error) {
	if len(b) == 0 {
		return 0, nil
	}
	deadline := s.writeDeadline()
	sent := 0
	for sent < len(b) {
		n, err := s.send(b[sent:])
		sent += n
		if err != nil && !isWouldBlock(err) {
			return sent, err
		}
		if sent == len(b) {
			return sent, nil
		}
		if !deadline.IsZero() && time.Now().After(deadline) {
			return sent, os.ErrDeadlineExceeded
		}
		time.Sleep(5 * time.Millisecond)
	}
	return sent, nil
}

func (s *eleSocket) recv(b []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	n, err := unix.Read(s.fd, b)
	if n == 0 && err == nil {
		return 0, io.EOF
	}
	return n, err
}

func (s *eleSocket) send(b []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return unix.Write(s.fd, b)
}

func isWouldBlock(err error) bool {
	return err == unix.EAGAIN || err == unix.EWOULDBLOCK || err == unix.EINTR || err == syscall.EAGAIN
}

func lostCarrier(fd int) bool {
	_, err := unix.Getpeername(fd)
	if err == nil {
		return false
	}
	switch err {
	case unix.ENOTCONN, unix.ECONNRESET, unix.EPIPE, unix.ECONNABORTED, unix.ETIMEDOUT:
		return true
	}
	return false
}

func (s *eleSocket) Close() error {
	return nil
}

func (s *eleSocket) LocalAddr() net.Addr  { return s.local }
func (s *eleSocket) RemoteAddr() net.Addr { return s.remote }

func (s *eleSocket) SetDeadline(t time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rdl = t
	s.wdl = t
	return nil
}

func (s *eleSocket) SetReadDeadline(t time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rdl = t
	return nil
}

func (s *eleSocket) SetWriteDeadline(t time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.wdl = t
	return nil
}

func (s *eleSocket) readDeadline() time.Time {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.rdl
}

func (s *eleSocket) writeDeadline() time.Time {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.wdl
}
