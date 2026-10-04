package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type config struct {
	telnetDoor string
	dropfile   string
	csvPath    string
	webSocket  bool
	exeDir     string
}

func parseArgs(args []string) config {
	cfg := config{}
	if exe, err := os.Executable(); err == nil {
		cfg.exeDir = filepath.Dir(exe)
	} else {
		cfg.exeDir, _ = os.Getwd()
	}
	cfg.csvPath = filepath.Join(cfg.exeDir, "affiliates.csv")

	for i := 0; i < len(args); i++ {
		a := args[i]
		if len(a) < 2 || (a[0] != '-' && a[0] != '/') {
			continue
		}
		body := a[1:]
		if strings.HasPrefix(a, "--") {
			body = a[2:]
		}
		key, val := splitOpt(body)
		switch strings.ToLower(key) {
		case "t", "telnet", "telnetdoor":
			val, i = takeVal(val, args, i)
			cfg.telnetDoor = val
		case "d", "drop", "dropfile":
			val, i = takeVal(val, args, i)
			cfg.dropfile = val
		case "c", "csv":
			val, i = takeVal(val, args, i)
			cfg.csvPath = val
		case "ws", "websocket":
			cfg.webSocket = true
			if val == "0" || strings.EqualFold(val, "false") {
				cfg.webSocket = false
			}
		case "h", "help", "?":
			printUsage()
			os.Exit(0)
		}
	}
	return cfg
}

func splitOpt(body string) (key, val string) {
	for i, c := range body {
		if c == '=' {
			return body[:i], body[i+1:]
		}
	}
	if len(body) == 1 {
		return body, ""
	}
	switch strings.ToLower(body[:1]) {
	case "t", "d", "c":
		rest := body[1:]
		if rest != "" && (rest[0] == '\\' || rest[0] == '/' || rest[0] == '.' ||
			(len(rest) >= 2 && rest[1] == ':') ||
			strings.EqualFold(body, "t") || strings.EqualFold(body, "d") || strings.EqualFold(body, "c")) {
			return body[:1], rest
		}
		if looksLikePath(rest) {
			return body[:1], rest
		}
	}
	return body, ""
}

func looksLikePath(s string) bool {
	if s == "" {
		return false
	}
	if s[0] == '\\' || s[0] == '/' || s[0] == '.' {
		return true
	}
	if len(s) >= 2 && s[1] == ':' {
		return true
	}
	return strings.ContainsAny(s, `/\`)
}

func takeVal(val string, args []string, i int) (string, int) {
	if val != "" {
		return val, i
	}
	if i+1 < len(args) && !isSwitch(args[i+1]) {
		return args[i+1], i + 1
	}
	return "", i
}

func isSwitch(s string) bool {
	return len(s) >= 2 && (s[0] == '-' || s[0] == '/')
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `AFFILIATE — EleBBS affiliate telnet menu

Usage:
  affiliate -T<telnetdoor> -D<node/door32.sys> [-C<affiliates.csv>] [-ws]

Windows EleBBS (32-bit build):
  C:\AFFILIATE\AFFILIATE.EXE -TC:\DOORS\TELNETDOOR.EXE -D*N\door32.sys

Linux EleBBS (amd64 build):
  /bbs/affiliate/affiliate -T/bbs/doors/telnetdoor -D*N/door32.sys

Switches may be glued (EleBBS style) or spaced:
  -T  -telnet     path to telnetdoor
  -D  -dropfile   path to DOOR32.SYS (or the node directory)
  -C  -csv        affiliate list CSV (default: affiliates.csv next to this binary)
  -ws             wrap the EleBBS socket as gorilla/websocket frames

Keys in the door:
  Up/Down  move lightbar
  Enter    telnetdoor -S<address> -D<dropfile> -W0
  A        add a BBS (after affiliate agreement)
  Esc      exit
`)
}
