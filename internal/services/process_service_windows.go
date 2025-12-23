//go:build windows

package services

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/omnihance/omnihance-a3-agent/internal/logger"
	"golang.org/x/sys/windows"
)

func terminateProcessWindowsImpl(ps *processService, proc *os.Process, pid int) error {
	handle, err := windows.OpenProcess(windows.PROCESS_TERMINATE, false, uint32(pid))
	if err != nil {
		ps.logger.Warn("failed to open process handle, using Kill()", logger.Field{Key: "pid", Value: pid}, logger.Field{Key: "error", Value: err})
		return killProcessWindows(ps, proc, pid)
	}
	defer windows.CloseHandle(handle)

	if err := windows.TerminateProcess(handle, 0); err != nil {
		windows.CloseHandle(handle)
		ps.logger.Warn("TerminateProcess failed, using Kill()", logger.Field{Key: "pid", Value: pid}, logger.Field{Key: "error", Value: err})
		return killProcessWindows(ps, proc, pid)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		_, err := proc.Wait()
		done <- err
	}()

	select {
	case <-ctx.Done():
		return errors.New("process did not terminate within timeout")
	case err := <-done:
		if err != nil {
			return fmt.Errorf("process wait error: %w", err)
		}
		return nil
	}
}

func killProcessWindows(ps *processService, proc *os.Process, pid int) error {
	if err := proc.Kill(); err != nil {
		return fmt.Errorf("failed to kill process: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		_, err := proc.Wait()
		done <- err
	}()

	select {
	case <-ctx.Done():
		return errors.New("process did not terminate after Kill()")
	case err := <-done:
		if err != nil {
			return fmt.Errorf("process wait error: %w", err)
		}
		return nil
	}
}

