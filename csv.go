package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"strings"
)

type affiliate struct {
	Name    string
	Address string
	Sysop   string
	Type    string
}

func loadAffiliates(path string) ([]affiliate, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.TrimLeadingSpace = true
	r.FieldsPerRecord = -1
	rows, err := r.ReadAll()
	if err != nil {
		return nil, err
	}

	out := make([]affiliate, 0, len(rows))
	for i, row := range rows {
		if len(row) < 4 {
			continue
		}
		a := affiliate{
			Name:    clip(row[0], fieldName),
			Address: strings.TrimSpace(row[1]),
			Sysop:   clip(row[2], fieldSysop),
			Type:    clip(row[3], fieldType),
		}
		if a.Name == "" && a.Address == "" {
			continue
		}
		if i == 0 && isHeader(a) {
			continue
		}
		out = append(out, a)
	}
	return out, nil
}

func isHeader(a affiliate) bool {
	return strings.EqualFold(a.Name, "BBS Name") &&
		strings.Contains(strings.ToLower(a.Address), "telnet")
}

func appendAffiliate(path string, a affiliate) error {
	_, err := os.Stat(path)
	createHeader := os.IsNotExist(err)

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open csv: %w", err)
	}
	defer f.Close()

	w := csv.NewWriter(f)
	if createHeader {
		if err := w.Write([]string{"BBS Name", "Telnet Address", "Sysop Name", "BBS Type"}); err != nil {
			return err
		}
	}
	if err := w.Write([]string{
		clip(a.Name, fieldName),
		strings.TrimSpace(a.Address),
		clip(a.Sysop, fieldSysop),
		clip(a.Type, fieldType),
	}); err != nil {
		return err
	}
	w.Flush()
	return w.Error()
}
