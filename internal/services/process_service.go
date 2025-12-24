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
	"github.com/omnihance/omnihance-a3-agent/internal/utils"
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
	IsBatchFile(path string) bool
	GetProcessByCommandLine(pattern string) ([]ProcessInfo, error)
	WaitForPort(host string, port int, timeout, checkInterval time.Duration) (bool, error)
	WaitForProcess(path string, timeout, checkInterval time.Duration) (bool, error)
	StartProcessWithHealthCheck(path string, port *int, timeout, checkInterval time.Duration, startParams ...string) error
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

	if ps.IsBatchFile(pathOfBinary) {
		return ps.isBatchFileRunning(normalizedPath)
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

func (ps *processService) isBatchFileRunning(normalizedPath string) (bool, error) {
	processes, err := ps.GetProcessList()
	if err != nil {
		return false, err
	}

	for _, proc := range processes {
		if proc.Name != "cmd.exe" {
			continue
		}

		if proc.CommandLine == "" {
			continue
		}

		normalizedCmdLine := ps.normalizeCommandLine(proc.CommandLine)
		if strings.Contains(normalizedCmdLine, normalizedPath) {
			return true, nil
		}
	}

	return false, nil
}

func (ps *processService) normalizeCommandLine(cmdLine string) string {
	normalized := strings.ToLower(cmdLine)
	normalized = strings.ReplaceAll(normalized, "/", "\\")
	normalized = strings.Trim(normalized, `"`)
	normalized = strings.TrimSpace(normalized)
	return normalized
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

	absPath, err := filepath.Abs(normalizedPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return fmt.Errorf("file not found: %s", absPath)
	}

	isRunning, err := ps.IsProcessRunning(absPath)
	if err != nil {
		ps.logger.Warn("failed to check if process is running", logger.Field{Key: "error", Value: err})
	}

	if isRunning {
		return fmt.Errorf("process is already running: %s", absPath)
	}

	var cmd *exec.Cmd
	if ps.IsBatchFile(absPath) {
		if runtime.GOOS == "windows" {
			cmd = exec.Command("cmd.exe", "/c", absPath)
		} else {
			cmd = exec.Command("sh", absPath)
		}
		cmd.Args = append(cmd.Args, startParams...)
	} else {
		if runtime.GOOS == "windows" {
			normalizedPath = strings.TrimSuffix(normalizedPath, ".exe")
			normalizedPath += ".exe"
			absPath, err = filepath.Abs(normalizedPath)
			if err != nil {
				return fmt.Errorf("failed to get absolute path: %w", err)
			}
		}
		cmd = exec.Command(absPath, startParams...)
	}

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

	var targetProcesses []ProcessInfo

	if ps.IsBatchFile(pathOfBinary) {
		processes, err := ps.GetProcessList()
		if err != nil {
			return err
		}

		normalizedCmdLine := ps.normalizeCommandLine(normalizedPath)
		for _, proc := range processes {
			if proc.Name == "cmd.exe" && proc.CommandLine != "" {
				procCmdLine := ps.normalizeCommandLine(proc.CommandLine)
				if strings.Contains(procCmdLine, normalizedCmdLine) {
					targetProcesses = append(targetProcesses, proc)
				}
			}
		}
	} else {
		processes, err := ps.GetProcessList()
		if err != nil {
			return err
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
				targetProcesses = append(targetProcesses, proc)
			}
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

func (ps *processService) IsBatchFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".bat" || ext == ".cmd"
}

func (ps *processService) GetProcessByCommandLine(pattern string) ([]ProcessInfo, error) {
	processes, err := ps.GetProcessList()
	if err != nil {
		return nil, err
	}

	normalizedPattern := ps.normalizeCommandLine(pattern)
	var matches []ProcessInfo

	for _, proc := range processes {
		if proc.CommandLine == "" {
			continue
		}

		normalizedCmdLine := ps.normalizeCommandLine(proc.CommandLine)
		if strings.Contains(normalizedCmdLine, normalizedPattern) {
			matches = append(matches, proc)
		}
	}

	return matches, nil
}

func (ps *processService) WaitForPort(host string, port int, timeout, checkInterval time.Duration) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return false, fmt.Errorf("timeout waiting for port %d", port)
		case <-ticker.C:
			isOpen, err := utils.IsPortOpen(host, port, 2*time.Second)
			if err != nil {
				ps.logger.Warn("error checking port", logger.Field{Key: "port", Value: port}, logger.Field{Key: "error", Value: err})
				continue
			}

			if isOpen {
				return true, nil
			}
		}
	}
}

func (ps *processService) WaitForProcess(path string, timeout, checkInterval time.Duration) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return false, fmt.Errorf("timeout waiting for process: %s", path)
		case <-ticker.C:
			isRunning, err := ps.IsProcessRunning(path)
			if err != nil {
				ps.logger.Warn("error checking process", logger.Field{Key: "path", Value: path}, logger.Field{Key: "error", Value: err})
				continue
			}

			if isRunning {
				return true, nil
			}
		}
	}
}

func (ps *processService) StartProcessWithHealthCheck(path string, port *int, timeout, checkInterval time.Duration, startParams ...string) error {
	if err := ps.StartProcess(path, startParams...); err != nil {
		return err
	}

	if port != nil {
		isReady, err := ps.WaitForPort("127.0.0.1", *port, timeout, checkInterval)
		if err != nil {
			return fmt.Errorf("process started but port check failed: %w", err)
		}

		if !isReady {
			return fmt.Errorf("process started but port %d did not become available within timeout", *port)
		}

		ps.logger.Info("process started and port is ready", logger.Field{Key: "path", Value: path}, logger.Field{Key: "port", Value: *port})
	} else {
		isReady, err := ps.WaitForProcess(path, timeout, checkInterval)
		if err != nil {
			return fmt.Errorf("process started but health check failed: %w", err)
		}

		if !isReady {
			return fmt.Errorf("process started but did not become available within timeout")
		}

		ps.logger.Info("process started and is ready", logger.Field{Key: "path", Value: path})
	}

	return nil
}
