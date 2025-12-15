package mw

import (
	"net"
	"net/http"
	"strings"
)

func RequireLocalIP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := getClientIP(r)
		if !isLocalIP(ip) {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func getClientIP(r *http.Request) string {
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}

	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		if idx := strings.Index(ip, ","); idx != -1 {
			ip = strings.TrimSpace(ip[:idx])
		}

		return ip
	}

	return r.RemoteAddr
}

func isLocalIP(ipStr string) bool {
	ipStr = strings.TrimSpace(ipStr)
	if ipStr == "" {
		return false
	}

	if strings.Contains(ipStr, ":") {
		ipStr, _, _ = strings.Cut(ipStr, ":")
	}

	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}

	if ip.IsLoopback() {
		return true
	}

	if ip.IsPrivate() {
		return true
	}

	return false
}
