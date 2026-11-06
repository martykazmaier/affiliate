package main

import "testing"

func TestCheckAddr(t *testing.T) {
	ok := []string{
		"bbs.example.com",
		"bbs.example.com:2323",
		"telnet://darkrealms.ca:23",
		"192.168.1.10",
		"192.168.1.10:2323",
		"localhost",
	}
	for _, in := range ok {
		if _, msg := checkAddr(in); msg != "" {
			t.Errorf("%q rejected: %s", in, msg)
		}
	}
	bad := []string{
		"",
		"   ",
		"not a host",
		"http://example.com",
		"bbs.example.com:99999",
		"bbs.example.com:abc",
		":23",
		"-bad.com",
		"1.2.3.999",
	}
	for _, in := range bad {
		if _, msg := checkAddr(in); msg == "" {
			t.Errorf("%q accepted", in)
		}
	}
}

func TestCheckName(t *testing.T) {
	if _, msg := checkName("   "); msg == "" {
		t.Fatal("empty name accepted")
	}
	got, msg := checkName(" Dark Realms ")
	if msg != "" || got != "Dark Realms" {
		t.Fatalf("got %q %q", got, msg)
	}
}
