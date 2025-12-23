//go:build !windows

package services

import (
	"errors"
	"os"

	"github.com/omnihance/omnihance-a3-agent/internal/logger"
)

func terminateProcessWindowsImpl(ps *processService, _ *os.Process, pid int) error {
	ps.logger.Error("terminateProcessWindows called on non-Windows system", logger.Field{Key: "pid", Value: pid})
	return errors.New("windows-specific function called on non-windows system")
}
