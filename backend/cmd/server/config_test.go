package main

import (
	"strings"
	"testing"
)

func TestListenAddr(t *testing.T) {
	for value, want := range map[string]string{
		"":               "127.0.0.1:8080",
		"127.0.0.1:9090": "127.0.0.1:9090",
		"localhost:9090": "localhost:9090",
		"[::1]:9090":     "[::1]:9090",
		"10.0.1.5:8080":  "10.0.1.5:8080",
		"192.168.1.2:80": "192.168.1.2:80",
		"[fd00::1]:8080": "[fd00::1]:8080",
	} {
		if got, err := listenAddr(value); err != nil || got != want {
			t.Errorf("listenAddr(%q) = %q, %v; want %q", value, got, err, want)
		}
	}
	for _, value := range []string{":8080", "0.0.0.0:8080", "[::]:8080", "8.8.8.8:80", "[2001:4860::1]:80", "example.com:80", "127.0.0.1"} {
		if got, err := listenAddr(value); err == nil {
			t.Errorf("listenAddr(%q) = %q, want an error", value, got)
		}
	}
}

func TestTokens(t *testing.T) {
	a, b := strings.Repeat("a", 32), strings.Repeat("b", 32)
	if _, _, err := tokens(a, b); err != nil {
		t.Fatal(err)
	}
	if _, _, err := tokens("", ""); err != nil {
		t.Fatal(err)
	}
	for _, pair := range [][2]string{{a, a}, {"short", b}, {a, "short"}} {
		if _, _, err := tokens(pair[0], pair[1]); err == nil {
			t.Errorf("tokens(%q, %q) accepted", pair[0], pair[1])
		}
	}
}
