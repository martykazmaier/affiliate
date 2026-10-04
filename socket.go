//go:build windows

package main

import (
	"fmt"
	"io"
	"net"
	"os"
	"sync"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

const fionbio = 0x8004667e

var (
	ws2             = windows.NewLazySystemDLL("ws2_32.dll")
	procRecv        = ws2.NewProc("recv")
	procSend        = ws2.NewProc("send")
	procIoctlsocket = ws2.NewProc("ioctlsocket")
)

// eleSocket is an EleBBS DOOR32 inherited WinSock SOCKET.
// Close does NOT call closesocket — EleBBS owns the handle and TelnetDoor
// must inherit the same descriptor value from door32.sys.
type eleSocket struct {
	h      windows.Handle
	mu     sync.Mutex
	rdl    time.Time
	wdl    time.Time
	local  net.Addr
	remote net.Addr
}

func initWinsock() error {
	var d windows.WSAData
	return windows.WSAStartup(uint32(0x0202), &d)
}

func openEleSocket(handle uintptr) (*eleSocket, error) {
	if handle == 0 {
		return nil, fmt.Errorf("DOOR32.SYS socket handle is 0")
	}
	if err := initWinsock(); err != nil {
		return nil, fmt.Errorf("WSAStartup: %w", err)
	}

	h := windows.Handle(handle)
	s := &eleSocket{h: h}
	// TCP_NODELAY is shared on the SOCKET. Leave it blocking for TelnetDoor;
	// only our lightbar loop switches in FIONBIO.
	_ = windows.SetsockoptInt(h, windows.IPPROTO_TCP, windows.TCP_NODELAY, 1)
	_ = windows.SetHandleInformation(h, windows.HANDLE_FLAG_INHERIT, windows.HANDLE_FLAG_INHERIT)
	if err := s.setNonBlock(true); err != nil {
		return nil, fmt.Errorf("FIONBIO: %w", err)
	}
	s.local = sockAddr(windows.Getsockname, h, "local")
	s.remote = sockAddr(windows.Getpeername, h, "remote")
	return s, nil
}

func (s *eleSocket) setNonBlock(nb bool) error {
	var v uint32
	if nb {
		v = 1
	}
	return ioctlSocket(s.h, fionbio, &v)
}

// PrepareChild puts the EleBBS SOCKET back to blocking so TelnetDoor/RMLib
// can inherit the same door32.sys handle value.
func (s *eleSocket) PrepareChild() {
	_ = s.setNonBlock(false)
	_ = windows.SetHandleInformation(s.h, windows.HANDLE_FLAG_INHERIT, windows.HANDLE_FLAG_INHERIT)
}

func (s *eleSocket) ResumeIO() {
	_ = s.setNonBlock(true)
}

func sockAddr(fn func(windows.Handle) (windows.Sockaddr, error), h windows.Handle, which string) net.Addr {
	sa, err := fn(h)
	if err != nil {
		return dummyAddr(which)
	}
	switch a := sa.(type) {
	case *windows.SockaddrInet4:
		return &net.TCPAddr{IP: net.IP(a.Addr[:]), Port: a.Port}
	case *windows.SockaddrInet6:
		return &net.TCPAddr{IP: net.IP(a.Addr[:]), Port: a.Port}
	default:
		return dummyAddr(which)
	}
}

func ioctlSocket(s windows.Handle, cmd uint32, arg *uint32) error {
	r1, _, err := procIoctlsocket.Call(uintptr(s), uintptr(cmd), uintptr(unsafe.Pointer(arg)))
	if int32(r1) == -1 {
		return err
	}
	return nil
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
		if lostCarrier(s.h) {
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
	r1, _, err := procRecv.Call(uintptr(s.h), uintptr(unsafe.Pointer(&b[0])), uintptr(len(b)), 0)
	n := int32(r1)
	if n == 0 {
		return 0, io.EOF
	}
	if n < 0 {
		return 0, err
	}
	return int(n), nil
}

func (s *eleSocket) send(b []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r1, _, err := procSend.Call(uintptr(s.h), uintptr(unsafe.Pointer(&b[0])), uintptr(len(b)), 0)
	n := int32(r1)
	if n < 0 {
		return 0, err
	}
	return int(n), nil
}

func isWouldBlock(err error) bool {
	if err == nil {
		return false
	}
	if errno, ok := err.(windows.Errno); ok {
		return errno == windows.WSAEWOULDBLOCK || errno == windows.WSAEINTR
	}
	return false
}

func lostCarrier(h windows.Handle) bool {
	_, err := windows.Getpeername(h)
	if err == nil {
		return false
	}
	if errno, ok := err.(windows.Errno); ok {
		switch errno {
		case windows.WSAENOTCONN, windows.WSAECONNRESET, windows.WSAECONNABORTED, windows.WSAENETRESET:
			return true
		}
	}
	return false
}

func (s *eleSocket) Close() error {
	// EleBBS still owns this SOCKET. Closing it would drop the user.
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
