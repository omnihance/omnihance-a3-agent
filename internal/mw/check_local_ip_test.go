package mw

import (
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetClientIPIgnoresSpoofableForwardedHeaders(t *testing.T) {
	req := httptest.NewRequest("GET", "/docs", nil)
	req.RemoteAddr = "203.0.113.10:12345"
	req.Header.Set("X-Real-IP", "127.0.0.1")
	req.Header.Set("X-Forwarded-For", "127.0.0.1")

	require.Equal(t, "203.0.113.10", getClientIP(req))
}

func TestIsLocalIPParsesIPv6LoopbackWithPort(t *testing.T) {
	require.True(t, isLocalIP("[::1]:12345"))
}
