package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func main() {
	cfg := parseArgs(os.Args[1:])
	_ = os.Chdir(cfg.exeDir)

	if cfg.csvPath != "" && !filepath.IsAbs(cfg.csvPath) {
		cfg.csvPath = filepath.Join(cfg.exeDir, cfg.csvPath)
	}

	drop, dropErr := loadDropfile(cfg.dropfile)
	local := dropErr != nil || drop == nil || drop.commType == commLocal || drop.handle == 0

	var sock *eleSocket
	var rw io.ReadWriter
	if !local {
		var err error
		sock, err = openEleSocket(drop.handle)
		if err != nil {
			fmt.Fprintf(os.Stderr, "EleBBS socket %d: %v\nfalling back to local console\n", drop.handle, err)
			local = true
		} else {
			var conn io.ReadWriter = sock
			if cfg.webSocket {
				conn = newWSBridge(sock, true)
			}
			rw = conn
		}
	}
	if local {
		rw = &consoleConn{}
		if drop == nil {
			drop = &dropfile{alias: "Local", path: cfg.dropfile}
		}
	}

	sess := newSession(rw, sock, local)
	defer sess.Close()

	if err := runDoor(sess, cfg, drop, sock); err != nil && err != io.EOF {
		sess.print(crlf + colRed + "  " + err.Error() + colReset + crlf)
		_ = sess.pause("Hit any key")
	}
	sess.goodbye()
}

func runDoor(sess *session, cfg config, drop *dropfile, sock *eleSocket) error {
	for {
		rows, err := loadAffiliates(cfg.csvPath)
		if err != nil {
			return fmt.Errorf("read csv: %w", err)
		}
		sel := 0
		off := 0
		const vis = listRows

		for {
			if sel >= len(rows) {
				sel = len(rows) - 1
			}
			if sel < 0 {
				sel = 0
			}
			if sel < off {
				off = sel
			}
			if sel >= off+vis {
				off = sel - vis + 1
			}
			if off < 0 {
				off = 0
			}

			sess.drawList(rows, sel, off, vis)
			k, err := sess.readKey()
			if err != nil {
				return err
			}
			switch k {
			case keyEsc:
				return nil
			case keyUp:
				if sel > 0 {
					sel--
				}
			case keyDown:
				if sel+1 < len(rows) {
					sel++
				}
			case keyHome:
				sel = 0
			case keyEnd:
				if len(rows) > 0 {
					sel = len(rows) - 1
				}
			case keyPgUp:
				sel -= vis
				if sel < 0 {
					sel = 0
				}
			case keyPgDn:
				sel += vis
				if sel >= len(rows) {
					sel = len(rows) - 1
					if sel < 0 {
						sel = 0
					}
				}
			case keyEnter:
				if len(rows) == 0 {
					continue
				}
				dropPath := cfg.dropfile
				if drop != nil && drop.path != "" {
					dropPath = drop.path
				}
				if err := launchTelnetDoor(sess, cfg.telnetDoor, rows[sel].Address, dropPath, sock); err != nil {
					return err
				}
				// Back to the top of the list after the telnet session.
				goto beginning
			case 'a', 'A':
				if err := sess.addAffiliate(cfg.csvPath); err != nil {
					return err
				}
				goto beginning
			}
		}
	beginning:
	}
}
