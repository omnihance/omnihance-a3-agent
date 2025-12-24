package utils

import (
	"fmt"
	"net"
	"time"
)

func IsPortOpen(host string, port int, timeout time.Duration) (bool, error) {
	if port < 1 || port > 65535 {
		return false, nil
	}

	address := net.JoinHostPort(host, fmt.Sprintf("%d", port))
	conn, err := net.DialTimeout("tcp", address, timeout)
	if err != nil {
		return false, nil
	}

	_ = conn.Close()
	return true, nil
}
