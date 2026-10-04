package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/gorilla/websocket"
)

// wsBridge converts raw EleBBS TCP bytes to/from gorilla/websocket frames.
// Outbound ANSI is sent as binary frames so CP437 / 8-bit color stays intact.
type wsBridge struct {
	raw    net.Conn
	server bool
	rbuf   []byte
	carry  []byte
}

func newWSBridge(raw net.Conn, server bool) *wsBridge {
	return &wsBridge{raw: raw, server: server}
}

func (w *wsBridge) Read(p []byte) (int, error) {
	for len(w.rbuf) == 0 {
		payload, opcode, err := w.readFrame()
		if err != nil {
			return 0, err
		}
		switch opcode {
		case websocket.TextMessage, websocket.BinaryMessage, 0:
			w.rbuf = payload
		case websocket.PingMessage:
			_ = w.writeFrame(websocket.PongMessage, payload)
		case websocket.PongMessage:
			continue
		case websocket.CloseMessage:
			return 0, io.EOF
		default:
			continue
		}
	}
	n := copy(p, w.rbuf)
	w.rbuf = w.rbuf[n:]
	return n, nil
}

func (w *wsBridge) Write(p []byte) (int, error) {
	if err := w.writeFrame(websocket.BinaryMessage, p); err != nil {
		return 0, err
	}
	return len(p), nil
}

func (w *wsBridge) Close() error                       { return w.raw.Close() }
func (w *wsBridge) LocalAddr() net.Addr                { return w.raw.LocalAddr() }
func (w *wsBridge) RemoteAddr() net.Addr               { return w.raw.RemoteAddr() }
func (w *wsBridge) SetDeadline(t time.Time) error      { return w.raw.SetDeadline(t) }
func (w *wsBridge) SetReadDeadline(t time.Time) error  { return w.raw.SetReadDeadline(t) }
func (w *wsBridge) SetWriteDeadline(t time.Time) error { return w.raw.SetWriteDeadline(t) }

func (w *wsBridge) readFull(n int) ([]byte, error) {
	buf := make([]byte, n)
	off := 0
	if len(w.carry) > 0 {
		c := copy(buf, w.carry)
		w.carry = w.carry[c:]
		off += c
	}
	for off < n {
		got, err := w.raw.Read(buf[off:])
		off += got
		if err != nil {
			return buf[:off], err
		}
	}
	return buf, nil
}

func (w *wsBridge) readFrame() ([]byte, int, error) {
	hdr, err := w.readFull(2)
	if err != nil {
		return nil, 0, err
	}
	opcode := int(hdr[0] & 0x0f)
	masked := hdr[1]&0x80 != 0
	n := int(hdr[1] & 0x7f)
	switch n {
	case 126:
		ext, err := w.readFull(2)
		if err != nil {
			return nil, 0, err
		}
		n = int(binary.BigEndian.Uint16(ext))
	case 127:
		ext, err := w.readFull(8)
		if err != nil {
			return nil, 0, err
		}
		n64 := binary.BigEndian.Uint64(ext)
		if n64 > 1<<20 {
			return nil, 0, fmt.Errorf("websocket frame too large")
		}
		n = int(n64)
	}
	var mask [4]byte
	if masked {
		m, err := w.readFull(4)
		if err != nil {
			return nil, 0, err
		}
		copy(mask[:], m)
	}
	payload, err := w.readFull(n)
	if err != nil {
		return nil, 0, err
	}
	if masked {
		for i := range payload {
			payload[i] ^= mask[i%4]
		}
	}
	return payload, opcode, nil
}

func (w *wsBridge) writeFrame(opcode int, payload []byte) error {
	n := len(payload)
	hdr := []byte{byte(0x80 | opcode)}
	switch {
	case n < 126:
		hdr = append(hdr, byte(n))
	case n <= 0xffff:
		var ext [2]byte
		binary.BigEndian.PutUint16(ext[:], uint16(n))
		hdr = append(hdr, 126)
		hdr = append(hdr, ext[:]...)
	default:
		var ext [8]byte
		binary.BigEndian.PutUint64(ext[:], uint64(n))
		hdr = append(hdr, 127)
		hdr = append(hdr, ext[:]...)
	}
	if !w.server {
		hdr[1] |= 0x80
		mask := []byte{0x37, 0xfa, 0x21, 0x3d}
		hdr = append(hdr, mask...)
		masked := make([]byte, n)
		for i := range payload {
			masked[i] = payload[i] ^ mask[i%4]
		}
		payload = masked
	}
	if _, err := w.raw.Write(hdr); err != nil {
		return err
	}
	if n == 0 {
		return nil
	}
	_, err := w.raw.Write(payload)
	return err
}
