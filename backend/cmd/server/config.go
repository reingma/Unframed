package main

import (
	"errors"
	"fmt"
	"net"
	"net/netip"

	"github.com/reingma/Unframed/backend/internal/api"
)

const defaultAddr = "127.0.0.1:8080"

// listenAddr parses BACKEND_ADDR. Only the Next.js server calls Go, never a
// browser, so Go binds a loopback or private address and refuses an
// unspecified or public one.
func listenAddr(value string) (string, error) {
	if value == "" {
		return defaultAddr, nil
	}
	host, _, err := net.SplitHostPort(value)
	if err != nil {
		return "", fmt.Errorf("BACKEND_ADDR %q: %w", value, err)
	}
	if host == "localhost" {
		return value, nil
	}
	ip, err := netip.ParseAddr(host)
	if err != nil || !(ip.IsLoopback() || ip.IsPrivate()) {
		return "", fmt.Errorf("BACKEND_ADDR %q must name a loopback or private IP address: the backend is never publicly reachable", value)
	}
	return value, nil
}

// tokens parses PREVIEW_TOKEN and ADMIN_TOKEN, which must differ so that a
// shared preview link never authorizes a reload.
func tokens(preview, admin string) (api.Token, api.Token, error) {
	previewToken, err := api.ParseToken(preview)
	if err != nil {
		return api.Token{}, api.Token{}, fmt.Errorf("PREVIEW_TOKEN: %w", err)
	}
	adminToken, err := api.ParseToken(admin)
	if err != nil {
		return api.Token{}, api.Token{}, fmt.Errorf("ADMIN_TOKEN: %w", err)
	}
	if preview != "" && preview == admin {
		return api.Token{}, api.Token{}, errors.New("PREVIEW_TOKEN and ADMIN_TOKEN must differ")
	}
	return previewToken, adminToken, nil
}
