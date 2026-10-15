package main

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// UserTimeOut Word in the packed RA 2.50 CONFIG.RA (same offset as elereader).
const raUserTimeOutOff = 1479

// idleFromRA reads UserTimeOut (seconds) from CONFIG.RA in %RA%.
// Zero or a missing file means no idle limit.
func idleFromRA() time.Duration {
	dir := strings.TrimRight(strings.Trim(strings.TrimSpace(os.Getenv("RA")), `"'`), `\/`)
	if dir == "" {
		return 0
	}
	for _, name := range []string{"CONFIG.RA", "config.ra"} {
		b, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			continue
		}
		if len(b) < raUserTimeOutOff+2 {
			continue
		}
		sec := int(binary.LittleEndian.Uint16(b[raUserTimeOutOff:]))
		if sec <= 0 {
			return 0
		}
		return time.Duration(sec) * time.Second
	}
	return 0
}
