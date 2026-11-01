package handler

import (
	"net"
	"net/http"
	"strings"
)

func securityHeaders(next http.Handler, https bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
		h.Set("Cache-Control", "no-store")
		if https {
			h.Set("Strict-Transport-Security", "max-age=31536000")
		}
		next.ServeHTTP(w, r)
	})
}

func proxyMiddleware(next http.Handler, trust bool) http.Handler {
	if !trust {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if peer := net.ParseIP(host); err == nil && peer != nil && peer.IsLoopback() {
			if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
				parts := strings.Split(fwd, ",")
				if client := net.ParseIP(strings.TrimSpace(parts[len(parts)-1])); client != nil {
					r.RemoteAddr = net.JoinHostPort(client.String(), "0")
				}
			}
		}
		next.ServeHTTP(w, r)
	})
}
