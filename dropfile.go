package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	commLocal  = 0
	commSerial = 1
	commTelnet = 2
)

type dropfile struct {
	path       string
	commType   int
	handle     uintptr
	baud       int
	bbsID      string
	userNum    int
	realName   string
	alias      string
	secLevel   int
	timeLeft   int
	emulation  int
	node       int
	socketMode bool
}

func loadDropfile(p string) (*dropfile, error) {
	resolved, err := resolveDropfile(p)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(resolved)
	if err != nil {
		return nil, fmt.Errorf("open dropfile: %w", err)
	}
	defer f.Close()

	var lines []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		lines = append(lines, strings.TrimRight(sc.Text(), "\r"))
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	if len(lines) < 11 {
		return nil, fmt.Errorf("dropfile %s is not DOOR32.SYS (need 11 lines)", resolved)
	}

	d := &dropfile{path: resolved}
	d.commType, _ = strconv.Atoi(strings.TrimSpace(lines[0]))
	h, _ := strconv.ParseUint(strings.TrimSpace(lines[1]), 10, 64)
	d.handle = uintptr(h)
	d.baud, _ = strconv.Atoi(strings.TrimSpace(lines[2]))
	d.bbsID = strings.TrimSpace(lines[3])
	d.userNum, _ = strconv.Atoi(strings.TrimSpace(lines[4]))
	d.realName = strings.TrimSpace(lines[5])
	d.alias = strings.TrimSpace(lines[6])
	d.secLevel, _ = strconv.Atoi(strings.TrimSpace(lines[7]))
	d.timeLeft, _ = strconv.Atoi(strings.TrimSpace(lines[8]))
	d.emulation, _ = strconv.Atoi(strings.TrimSpace(lines[9]))
	d.node, _ = strconv.Atoi(strings.TrimSpace(lines[10]))
	d.socketMode = d.commType == commTelnet && d.handle != 0
	return d, nil
}

func resolveDropfile(p string) (string, error) {
	if p == "" {
		for _, cand := range []string{"door32.sys", "DOOR32.SYS"} {
			if st, err := os.Stat(cand); err == nil && !st.IsDir() {
				abs, _ := filepath.Abs(cand)
				return abs, nil
			}
		}
		return "", fmt.Errorf("no dropfile given and door32.sys not in the current directory")
	}
	st, err := os.Stat(p)
	if err != nil {
		return "", err
	}
	if !st.IsDir() {
		abs, _ := filepath.Abs(p)
		return abs, nil
	}
	for _, name := range []string{"door32.sys", "DOOR32.SYS"} {
		cand := filepath.Join(p, name)
		if _, err := os.Stat(cand); err == nil {
			abs, _ := filepath.Abs(cand)
			return abs, nil
		}
	}
	return "", fmt.Errorf("no door32.sys in %s", p)
}
