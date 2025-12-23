package services

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/omnihance/omnihance-a3-agent/internal/logger"
	"github.com/shirou/gopsutil/v3/process"
)

type ProcessInfo struct {
	PID         int       `json:"pid"`
	Name        string    `json:"name"`
	Path        string    `json:"path"`
	CommandLine string    `json:"command_line"`
	StartTime   time.Time `json:"start_time"`
}

type ProcessService interface {
	GetProcessList() ([]ProcessInfo, error)
	GetProcessCount() (int, error)
	IsProcessRunning(pathOfBinary string) (bool, error)
	StartProcess(pathOfBinary string, startParams ...string) error
	StopProcess(pathOfBinary string) error
}

type processService struct {
	logger logger.Logger
}

func NewProcessService(logger logger.Logger) ProcessService {
	return &processService{logger: logger}
}

func (ps *processService) GetProcessList() ([]ProcessInfo, error) {
	processes, err := process.Processes()
	if err != nil {
		ps.logger.Error("failed to get process list", logger.Field{Key: "error", Value: err})
		return nil, fmt.Errorf("failed to get process list: %w", err)
	}

	result := make([]ProcessInfo, 0, len(processes))

	for _, proc := range processes {
		info, err := ps.getProcessInfo(proc)
		if err != nil {
			continue
		}

		result = append(result, *info)
	}

	return result, nil
}

func (ps *processService) getProcessInfo(proc *process.Process) (*ProcessInfo, error) {
	pid := int(proc.Pid)

	name, err := proc.Name()
	if err != nil {
		return nil, fmt.Errorf("failed to get process name: %w", err)
	}

	exe, err := proc.Exe()
	if err != nil {
		exe = ""
	}

	cmdline, err := proc.Cmdline()
	if err != nil {
		cmdline = ""
	}

	createTime, err := proc.CreateTime()
	if err != nil {
		createTime = 0
	}

	startTime := time.Unix(0, createTime*int64(time.Millisecond))

	return &ProcessInfo{
		PID:         pid,
		Name:        name,
		Path:        exe,
		CommandLine: cmdline,
		StartTime:   startTime,
	}, nil
}

func (ps *processService) GetProcessCount() (int, error) {
	list, err := ps.GetProcessList()
	if err != nil {
		return 0, err
	}

	return len(list), nil
}

func (ps *processService) IsProcessRunning(pathOfBinary string) (bool, error) {
	normalizedPath, err := ps.normalizePath(pathOfBinary)
	if err != nil {
		return false, fmt.Errorf("failed to normalize path: %w", err)
	}

	processes, err := ps.GetProcessList()
	if err != nil {
		return false, err
	}

	for _, proc := range processes {
		if proc.Path == "" {
			continue
		}

		procPath, err := ps.normalizePath(proc.Path)
		if err != nil {
			continue
		}

		if procPath == normalizedPath {
			return true, nil
		}
	}

	return false, nil
}

func (ps *processService) normalizePath(path string) (string, error) {
	if path == "" {
		return "", errors.New("path is empty")
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %w", err)
	}

	normalized := filepath.Clean(absPath)

	if runtime.GOOS == "windows" {
		normalized = strings.ToLower(normalized)
		normalized = strings.ReplaceAll(normalized, "/", "\\")
	} else {
		normalized = strings.ReplaceAll(normalized, "\\", "/")
	}

	return normalized, nil
}

func (ps *processService) StartProcess(pathOfBinary string, startParams ...string) error {
	normalizedPath, err := ps.normalizePath(pathOfBinary)
	if err != nil {
		return fmt.Errorf("failed to normalize path: %w", err)
	}

	if runtime.GOOS == "windows" {
		normalizedPath = strings.TrimSuffix(normalizedPath, ".exe")
		normalizedPath += ".exe"
	}

	absPath, err := filepath.Abs(normalizedPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return fmt.Errorf("binary not found: %s", absPath)
	}

	isRunning, err := ps.IsProcessRunning(absPath)
	if err != nil {
		ps.logger.Warn("failed to check if process is running", logger.Field{Key: "error", Value: err})
	}

	if isRunning {
		return fmt.Errorf("process is already running: %s", absPath)
	}

	cmd := exec.Command(absPath, startParams...)

	dir := filepath.Dir(absPath)
	cmd.Dir = dir

	if err := cmd.Start(); err != nil {
		ps.logger.Error("failed to start process", logger.Field{Key: "path", Value: absPath}, logger.Field{Key: "error", Value: err})
		return fmt.Errorf("failed to start process: %w", err)
	}

	ps.logger.Info("process started", logger.Field{Key: "path", Value: absPath}, logger.Field{Key: "pid", Value: cmd.Process.Pid})

	return nil
}

func (ps *processService) StopProcess(pathOfBinary string) error {
	normalizedPath, err := ps.normalizePath(pathOfBinary)
	if err != nil {
		return fmt.Errorf("failed to normalize path: %w", err)
	}

	processes, err := ps.GetProcessList()
	if err != nil {
		return err
	}

	var targetProcesses []ProcessInfo
	for _, proc := range processes {
		if proc.Path == "" {
			continue
		}

		procPath, err := ps.normalizePath(proc.Path)
		if err != nil {
			continue
		}

		if procPath == normalizedPath {
			targetProcesses = append(targetProcesses, proc)
		}
	}

	if len(targetProcesses) == 0 {
		return fmt.Errorf("process not found: %s", pathOfBinary)
	}

	var lastErr error
	for _, procInfo := range targetProcesses {
		if err := ps.terminateProcess(procInfo.PID); err != nil {
			ps.logger.Error("failed to terminate process", logger.Field{Key: "pid", Value: procInfo.PID}, logger.Field{Key: "error", Value: err})
			lastErr = err
		} else {
			ps.logger.Info("process terminated", logger.Field{Key: "pid", Value: procInfo.PID}, logger.Field{Key: "path", Value: pathOfBinary})
		}
	}

	if lastErr != nil {
		return fmt.Errorf("failed to stop one or more processes: %w", lastErr)
	}

	return nil
}

func (ps *processService) terminateProcess(pid int) error {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("failed to find process: %w", err)
	}

	if runtime.GOOS == "windows" {
		return ps.terminateProcessWindows(proc, pid)
	}

	return ps.terminateProcessUnix(proc, pid)
}

func (ps *processService) terminateProcessUnix(proc *os.Process, pid int) error {
	if err := proc.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("failed to send SIGTERM: %w", err)
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
		ps.logger.Warn("process did not terminate gracefully, sending SIGKILL", logger.Field{Key: "pid", Value: pid})
		if err := proc.Signal(syscall.SIGKILL); err != nil {
			return fmt.Errorf("failed to send SIGKILL: %w", err)
		}

		waitCtx, waitCancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer waitCancel()

		select {
		case <-done:
			return nil
		case <-waitCtx.Done():
			return errors.New("process did not terminate after SIGKILL")
		}
	case err := <-done:
		if err != nil {
			return fmt.Errorf("process wait error: %w", err)
		}
		return nil
	}
}

func (ps *processService) terminateProcessWindows(proc *os.Process, pid int) error {
	return terminateProcessWindowsImpl(ps, proc, pid)
}
