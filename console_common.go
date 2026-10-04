package main

import (
	"net"
	"os"
	"time"
)

type consoleConn struct {
	rdl time.Time
	wdl time.Time
}

func (c *consoleConn) Write(b []byte) (int, error) {
	return os.Stdout.Write(b)
}

func (c *consoleConn) Close() error { return nil }

func (c *consoleConn) LocalAddr() net.Addr  { return dummyAddr("console") }
func (c *consoleConn) RemoteAddr() net.Addr { return dummyAddr("local") }

func (c *consoleConn) SetDeadline(t time.Time) error {
	c.rdl = t
	c.wdl = t
	return nil
}
func (c *consoleConn) SetReadDeadline(t time.Time) error {
	c.rdl = t
	return nil
}
func (c *consoleConn) SetWriteDeadline(t time.Time) error {
	c.wdl = t
	return nil
}
