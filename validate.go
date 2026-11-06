package main

import (
	"strings"
)

func checkName(s string) (string, string) {
	s = clip(s, fieldName)
	if s == "" {
		return "", "BBS name is required."
	}
	return s, ""
}

func checkAddr(s string) (string, string) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", "Telnet address is required."
	}
	if strings.ContainsAny(s, " \t") {
		return "", "Telnet address cannot contain spaces."
	}
	low := strings.ToLower(s)
	switch {
	case strings.HasPrefix(low, "telnet://"):
		s = s[9:]
	case strings.HasPrefix(low, "http://"), strings.HasPrefix(low, "https://"):
		return "", "Use a telnet host or host:port, not an http URL."
	}
	if s == "" {
		return "", "Telnet address is required."
	}
	host := s
	if i := strings.LastIndex(s, ":"); i >= 0 && strings.Count(s, ":") == 1 {
		host = s[:i]
		if !validPort(s[i+1:]) {
			return "", "Port must be a number from 1 to 65535."
		}
	}
	if !validHost(host) {
		return "", "Enter a hostname or IP, optional :port."
	}
	return s, ""
}

func checkSysop(s string) (string, string) {
	s = clip(s, fieldSysop)
	if s == "" {
		return "", "Sysop name is required."
	}
	return s, ""
}

func checkType(s string) (string, string) {
	s = clip(s, fieldType)
	if s == "" {
		return "", "BBS type is required."
	}
	return s, ""
}

func validPort(p string) bool {
	if p == "" {
		return false
	}
	n := 0
	for i := 0; i < len(p); i++ {
		if p[i] < '0' || p[i] > '9' {
			return false
		}
		n = n*10 + int(p[i]-'0')
		if n > 65535 {
			return false
		}
	}
	return n >= 1
}

func validHost(h string) bool {
	if h == "" || len(h) > 253 {
		return false
	}
	if isIPv4(h) {
		return true
	}
	if h[0] == '.' || h[len(h)-1] == '.' {
		return false
	}
	hasLetter := false
	start := 0
	for i := 0; i <= len(h); i++ {
		if i < len(h) && h[i] != '.' {
			continue
		}
		lab := h[start:i]
		if len(lab) < 1 || len(lab) > 63 || lab[0] == '-' || lab[len(lab)-1] == '-' {
			return false
		}
		for j := 0; j < len(lab); j++ {
			c := lab[j]
			switch {
			case c >= 'A' && c <= 'Z', c >= 'a' && c <= 'z':
				hasLetter = true
			case c >= '0' && c <= '9', c == '-':
			default:
				return false
			}
		}
		start = i + 1
	}
	return hasLetter
}

func isIPv4(h string) bool {
	n := 0
	part := 0
	digits := 0
	for i := 0; i <= len(h); i++ {
		if i < len(h) {
			c := h[i]
			if c >= '0' && c <= '9' {
				part = part*10 + int(c-'0')
				digits++
				if digits > 3 || part > 255 {
					return false
				}
				continue
			}
			if c != '.' {
				return false
			}
		}
		if digits < 1 {
			return false
		}
		n++
		part = 0
		digits = 0
	}
	return n == 4
}

func addrListed(path, addr string) bool {
	rows, err := loadAffiliates(path)
	if err != nil {
		return false
	}
	for _, r := range rows {
		if strings.EqualFold(r.Address, addr) {
			return true
		}
	}
	return false
}
