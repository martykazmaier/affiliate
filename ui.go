package main

import (
	"fmt"
	"strings"
)

const (
	fieldName  = 24
	fieldAddr  = 24 // lightbar only; CSV and TelnetDoor keep the full host:port
	fieldSysop = 14
	fieldType  = 10
	addrInput  = 80

	// CP437 box drawing (raw bytes, not UTF-8).
	boxH  = "\xCD" // ═
	boxV  = "\xBA" // ║
	boxSV = "\xB3" // │
	boxTL = "\xC9" // ╔
	boxTR = "\xBB" // ╗
	boxBL = "\xC8" // ╚
	boxBR = "\xBC" // ╝
	boxML = "\xCC" // ╠
	boxMR = "\xB9" // ╣
	boxTD = "\xCB" // ╦
	boxBU = "\xCA" // ╩
	boxX  = "\xCE" // ╬
	boxSH = "\xC4" // ─

	screenCols = 79
	screenRows = 24
	// 24│24│14│10 = 75 inside, plus ║ ║ = 77, plus 1-col gutters = 79.
	panelCols = fieldName + 1 + fieldAddr + 1 + fieldSysop + 1 + fieldType
	frameCols = panelCols + 2
	gutter    = (screenCols - frameCols) / 2
	listRows  = screenRows - 7

	colBg    = "\x1b[0;37;44m"
	colPanel = "\x1b[0;30;46m"
	colTitle = "\x1b[1;37;46m"
	colHead  = "\x1b[1;33;46m"
	colBar   = "\x1b[0;30;47m"
	colYel   = "\x1b[1;33;44m"
	colBWhi  = "\x1b[1;37;44m"
	colCyan  = "\x1b[0;36;44m"
	colRed   = "\x1b[1;31;44m"
	colGrn   = "\x1b[1;32;44m"
	colReset = "\x1b[0m"
	crlf     = "\r\n"
)

func pad(s string, n int) string {
	b := make([]byte, 0, n)
	for i := 0; i < len(s) && len(b) < n; i++ {
		c := s[i]
		if c < 32 || c == 127 {
			c = '?'
		}
		b = append(b, c)
	}
	for len(b) < n {
		b = append(b, ' ')
	}
	return string(b)
}

func clip(s string, n int) string {
	s = strings.TrimSpace(s)
	b := make([]byte, 0, n)
	for i := 0; i < len(s) && len(b) < n; i++ {
		c := s[i]
		if c < 32 || c == 127 {
			continue
		}
		b = append(b, c)
	}
	return string(b)
}

func center(s string, n int) string {
	s = pad(s, len(s))
	if len(s) > n {
		s = s[:n]
	}
	left := (n - len(s)) / 2
	return strings.Repeat(" ", left) + s + strings.Repeat(" ", n-left-len(s))
}

func affiliateLine(name, addr, sysop, typ string) string {
	return boxV + pad(name, fieldName) + boxSV + pad(addr, fieldAddr) + boxSV + pad(sysop, fieldSysop) + boxSV + pad(typ, fieldType) + boxV
}

func frameLine(left, fill, junct, right string) string {
	return left +
		strings.Repeat(fill, fieldName) + junct +
		strings.Repeat(fill, fieldAddr) + junct +
		strings.Repeat(fill, fieldSysop) + junct +
		strings.Repeat(fill, fieldType) + right
}

func boxTop() string    { return frameLine(boxTL, boxH, boxTD, boxTR) }
func boxRule() string   { return frameLine(boxML, boxSH, boxX, boxMR) }
func boxBottom() string { return frameLine(boxBL, boxH, boxBU, boxBR) }

func boxed(inner string) string {
	b := []byte(pad(inner, panelCols))
	if len(b) == panelCols {
		b[fieldName] = boxSV[0]
		b[fieldName+1+fieldAddr] = boxSV[0]
		b[fieldName+1+fieldAddr+1+fieldSysop] = boxSV[0]
	}
	return boxV + string(b) + boxV
}

func (s *session) at(row, col int) {
	s.print(fmt.Sprintf("\x1b[%d;%dH", row, col))
}

// panelAt paints one 79-column row at `row` (1-based). No CR/LF — wrap cannot
// push a high-bit glyph into column 80. Gutters stay ASCII spaces.
func (s *session) panelAt(row int, attr, inner string) {
	inner = pad(inner, frameCols)
	s.at(row, 1)
	s.print(colBg)
	s.print(strings.Repeat(" ", gutter))
	s.print(attr)
	s.print(inner)
	s.print(colBg)
	s.print(strings.Repeat(" ", screenCols-gutter-frameCols))
}

func (s *session) keysAt(row int, text string) {
	s.at(row, 1)
	s.print(colBg + colTitle + center(text, screenCols))
}

func (s *session) drawList(rows []affiliate, sel, off, vis int) {
	s.cls()
	if vis < 1 {
		vis = listRows
	}

	s.panelAt(1, colTitle, boxTop())
	s.panelAt(2, colTitle, boxed(center("AFFILIATE NETWORK", panelCols)))
	s.panelAt(3, colHead, boxRule())
	s.panelAt(4, colHead, affiliateLine("BBS Name", "Telnet Address", "Sysop Name", "BBS Type"))
	s.panelAt(5, colHead, boxRule())

	for i := 0; i < vis; i++ {
		row := 6 + i
		idx := off + i
		if idx >= len(rows) {
			if len(rows) == 0 && i == 0 {
				s.panelAt(row, colPanel, boxed(" No affiliates yet. Hit A to add your BBS."))
			} else {
				s.panelAt(row, colPanel, affiliateLine("", "", "", ""))
			}
			continue
		}
		line := affiliateLine(rows[idx].Name, rows[idx].Address, rows[idx].Sysop, rows[idx].Type)
		attr := colPanel
		if idx == sel {
			attr = colBar
		}
		s.panelAt(row, attr, line)
	}

	s.panelAt(23, colTitle, boxBottom())
	s.keysAt(24, "ENTER Connect    A Add affiliate    ESC Exit")
}

func (s *session) pause(msg string) error {
	s.at(22, 1)
	s.print(colYel + pad(" "+msg, screenCols)[:screenCols])
	_, err := s.readKey()
	return err
}

func (s *session) addAffiliate(csvPath string) error {
	s.cls()
	s.panelAt(1, colTitle, boxTop())
	s.panelAt(2, colTitle, boxed(center("AFFILIATE AGREEMENT", panelCols)))
	s.panelAt(3, colTitle, boxBottom())
	s.panelAt(4, colPanel, boxed(""))
	s.panelAt(5, colPanel, boxed(" To list your BBS in this directory you must:"))
	s.panelAt(6, colPanel, boxed(""))
	s.panelAt(7, colTitle, boxed("  1) Download and run this door on your system"))
	s.panelAt(8, colTitle, boxed("  2) Add THIS BBS to your affiliate list"))
	s.panelAt(9, colPanel, boxed(""))
	s.panelAt(10, colHead, boxed(" Do you agree? (Y/N) "))
	s.at(10, gutter+23)
	s.print(colHead + "\x1b[?25h")

	for {
		k, err := s.readKey()
		if err != nil {
			return err
		}
		if k == keyEsc || k == 'n' || k == 'N' {
			s.print("N")
			s.print("\x1b[?25l")
			return nil
		}
		if k == 'y' || k == 'Y' {
			s.print("Y")
			break
		}
	}
	s.print("\x1b[?25l")

	s.cls()
	s.panelAt(1, colTitle, boxTop())
	s.panelAt(2, colTitle, boxed(center("ADD AFFILIATE", panelCols)))
	s.panelAt(3, colTitle, boxBottom())
	s.panelAt(5, colPanel, boxed(""))
	s.at(6, 1)
	s.print(colBg)

	ask := func(row int, label string, max int) (string, bool, error) {
		s.at(row, 1)
		s.print(colBg + strings.Repeat(" ", gutter) + colCyan + label + " " + colBWhi)
		return s.readLine(max)
	}

	name, ok, err := ask(6, "BBS Name........:", fieldName)
	if err != nil || !ok {
		return err
	}
	addr, ok, err := ask(8, "Telnet address..:", addrInput)
	if err != nil || !ok {
		return err
	}
	sysop, ok, err := ask(10, "Sysop name......:", fieldSysop)
	if err != nil || !ok {
		return err
	}
	typ, ok, err := ask(12, "BBS type........:", fieldType)
	if err != nil || !ok {
		return err
	}

	name = clip(name, fieldName)
	addr = strings.TrimSpace(addr)
	sysop = clip(sysop, fieldSysop)
	typ = clip(typ, fieldType)
	if name == "" || addr == "" {
		s.panelAt(20, colRed, boxed(" BBS name and telnet address are required."))
		return s.pause("Hit any key")
	}

	if err := appendAffiliate(csvPath, affiliate{Name: name, Address: addr, Sysop: sysop, Type: typ}); err != nil {
		s.panelAt(20, colRed, boxed(" Could not save: "+clip(err.Error(), 50)))
		return s.pause("Hit any key")
	}
	s.panelAt(20, colGrn, boxed(" Added "+name+" to the affiliate list."))
	return s.pause("Hit any key")
}

func (s *session) goodbye() {
	s.print("\x1b[?7h\x1b[0m\x1b[2J\x1b[H\x1b[?25h")
	s.print("  Returning to the BBS...\r\n")
}
