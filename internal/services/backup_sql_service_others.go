//go:build !windows

package services

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	mssqlLinuxServiceName  = "mssql-server"
	mssqlLinuxProcessName  = "sqlservr"
	mssqlServiceCheckLimit = 3 * time.Second
)

func localSQLServerServiceRunning() (bool, error) {
	running, err := serviceCommandReportsRunning("systemctl", "is-active", "--quiet", mssqlLinuxServiceName)
	if err != nil {
		return false, err
	}

	if running {
		return true, nil
	}

	running, err = serviceCommandReportsRunning("service", mssqlLinuxServiceName, "status")
	if err != nil {
		return false, err
	}

	if running {
		return true, nil
	}

	return sqlServerProcessRunning()
}

func serviceCommandReportsRunning(name string, args ...string) (bool, error) {
	if _, err := exec.LookPath(name); err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return false, nil
		}

		return false, fmt.Errorf("failed to find %s: %w", name, err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), mssqlServiceCheckLimit)
	defer cancel()

	err := exec.CommandContext(ctx, name, args...).Run()
	if ctx.Err() != nil {
		return false, fmt.Errorf("%s service check timed out: %w", name, ctx.Err())
	}

	if err == nil {
		return true, nil
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return false, nil
	}

	return false, fmt.Errorf("failed to run %s service check: %w", name, err)
}

func sqlServerProcessRunning() (bool, error) {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}

		return false, fmt.Errorf("failed to read /proc: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		if _, err := strconv.Atoi(entry.Name()); err != nil {
			continue
		}

		running, err := procEntryIsSQLServer(filepath.Join("/proc", entry.Name()))
		if err != nil {
			return false, err
		}

		if running {
			return true, nil
		}
	}

	return false, nil
}

func procEntryIsSQLServer(procPath string) (bool, error) {
	comm, err := os.ReadFile(filepath.Join(procPath, "comm"))
	if err == nil && sqlServerProcessName(string(comm)) {
		return true, nil
	}

	if err != nil && !ignorableProcReadError(err) {
		return false, fmt.Errorf("failed to read process name %s: %w", procPath, err)
	}

	cmdline, err := os.ReadFile(filepath.Join(procPath, "cmdline"))
	if err == nil {
		return sqlServerCommandLine(string(cmdline)), nil
	}

	if ignorableProcReadError(err) {
		return false, nil
	}

	return false, fmt.Errorf("failed to read process command line %s: %w", procPath, err)
}

func sqlServerProcessName(name string) bool {
	name = strings.ToLower(strings.TrimSpace(name))
	return name == mssqlLinuxProcessName || name == mssqlLinuxServiceName
}

func sqlServerCommandLine(cmdline string) bool {
	parts := strings.Fields(strings.ReplaceAll(cmdline, "\x00", " "))
	for _, part := range parts {
		if sqlServerProcessName(filepath.Base(part)) {
			return true
		}
	}

	return false
}

func ignorableProcReadError(err error) bool {
	return errors.Is(err, os.ErrNotExist) || errors.Is(err, os.ErrPermission)
}
