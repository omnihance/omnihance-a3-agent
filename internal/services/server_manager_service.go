package services

import (
	"fmt"
	"time"

	"github.com/omnihance/omnihance-a3-agent/internal/db"
	"github.com/omnihance/omnihance-a3-agent/internal/logger"
	"github.com/omnihance/omnihance-a3-agent/internal/utils"
)

type ServerManagerService interface {
	StartServerSequence() error
	StopServerSequence() error
	StartProcess(id int64) error
	StopProcess(id int64) error
	GetProcessStatus(id int64) (*ProcessStatus, error)
}

type serverManagerService struct {
	db             db.InternalDB
	processService ProcessService
	logger         logger.Logger
}

func NewServerManagerService(internalDB db.InternalDB, processService ProcessService, log logger.Logger) ServerManagerService {
	return &serverManagerService{
		db:             internalDB,
		processService: processService,
		logger:         log,
	}
}

func (s *serverManagerService) StartServerSequence() error {
	processes, err := s.db.GetServerProcesses()
	if err != nil {
		return fmt.Errorf("failed to get server processes: %w", err)
	}

	if len(processes) == 0 {
		return fmt.Errorf("no processes configured")
	}

	for i, proc := range processes {
		s.logger.Info("starting process in sequence", logger.Field{Key: "name", Value: proc.Name}, logger.Field{Key: "order", Value: i + 1})

		if err := s.startProcessInternal(&proc); err != nil {
			s.logger.Error("failed to start process in sequence", logger.Field{Key: "name", Value: proc.Name}, logger.Field{Key: "error", Value: err})
			return fmt.Errorf("failed to start process %s: %w", proc.Name, err)
		}

		s.logger.Info("process started successfully", logger.Field{Key: "name", Value: proc.Name})
	}

	return nil
}

func (s *serverManagerService) StopServerSequence() error {
	processes, err := s.db.GetServerProcesses()
	if err != nil {
		return fmt.Errorf("failed to get server processes: %w", err)
	}

	if len(processes) == 0 {
		return fmt.Errorf("no processes configured")
	}

	for i := len(processes) - 1; i >= 0; i-- {
		proc := processes[i]
		s.logger.Info("stopping process in sequence", logger.Field{Key: "name", Value: proc.Name}, logger.Field{Key: "order", Value: len(processes) - i})

		if err := s.stopProcessInternal(&proc); err != nil {
			s.logger.Error("failed to stop process in sequence", logger.Field{Key: "name", Value: proc.Name}, logger.Field{Key: "error", Value: err})
			return fmt.Errorf("failed to stop process %s: %w", proc.Name, err)
		}

		s.logger.Info("process stopped successfully", logger.Field{Key: "name", Value: proc.Name})
	}

	return nil
}

func (s *serverManagerService) StartProcess(id int64) error {
	proc, err := s.db.GetServerProcess(id)
	if err != nil {
		return fmt.Errorf("failed to get server process: %w", err)
	}

	return s.startProcessInternal(proc)
}

func (s *serverManagerService) StopProcess(id int64) error {
	proc, err := s.db.GetServerProcess(id)
	if err != nil {
		return fmt.Errorf("failed to get server process: %w", err)
	}

	return s.stopProcessInternal(proc)
}

func (s *serverManagerService) startProcessInternal(proc *db.ServerProcess) error {
	timeout := 60 * time.Second
	checkInterval := 2 * time.Second

	var port *int
	if proc.Port != nil {
		port = proc.Port
	}

	if err := s.processService.StartProcessWithHealthCheck(proc.Path, port, timeout, checkInterval); err != nil {
		return fmt.Errorf("failed to start process: %w", err)
	}

	now := time.Now()
	if err := s.db.UpdateProcessStartTime(proc.ID, now); err != nil {
		s.logger.Warn("failed to update process start time", logger.Field{Key: "id", Value: proc.ID}, logger.Field{Key: "error", Value: err})
	}

	s.logger.Info("process started and health check passed", logger.Field{Key: "name", Value: proc.Name}, logger.Field{Key: "id", Value: proc.ID})

	return nil
}

func (s *serverManagerService) stopProcessInternal(proc *db.ServerProcess) error {
	if err := s.processService.StopProcess(proc.Path); err != nil {
		return fmt.Errorf("failed to stop process: %w", err)
	}

	now := time.Now()
	if err := s.db.UpdateProcessEndTime(proc.ID, now); err != nil {
		s.logger.Warn("failed to update process end time", logger.Field{Key: "id", Value: proc.ID}, logger.Field{Key: "error", Value: err})
	}

	s.logger.Info("process stopped", logger.Field{Key: "name", Value: proc.Name}, logger.Field{Key: "id", Value: proc.ID})

	return nil
}

func (s *serverManagerService) GetProcessStatus(id int64) (*ProcessStatus, error) {
	proc, err := s.db.GetServerProcess(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get server process: %w", err)
	}

	isRunning, err := s.processService.IsProcessRunning(proc.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to check if process is running: %w", err)
	}

	status := &ProcessStatus{
		Running: isRunning,
	}

	if proc.Port != nil {
		portOpen, err := utils.IsPortOpen("127.0.0.1", *proc.Port, 2*time.Second)
		if err == nil {
			status.PortOpen = &portOpen
		}
	}

	if proc.StartTime != nil {
		startTimeStr := proc.StartTime.Format(time.RFC3339)
		status.StartTime = &startTimeStr

		if isRunning {
			currentUptime := time.Now().Unix() - proc.StartTime.Unix()
			status.CurrentUptimeSeconds = &currentUptime
		} else if proc.EndTime != nil {
			lastUptime := proc.EndTime.Unix() - proc.StartTime.Unix()
			status.LastUptimeSeconds = &lastUptime
			endTimeStr := proc.EndTime.Format(time.RFC3339)
			status.EndTime = &endTimeStr
		}
	}

	return status, nil
}

type ProcessStatus struct {
	Running              bool    `json:"running"`
	PortOpen             *bool   `json:"port_open,omitempty"`
	StartTime            *string `json:"start_time,omitempty"`
	EndTime              *string `json:"end_time,omitempty"`
	CurrentUptimeSeconds *int64  `json:"current_uptime_seconds,omitempty"`
	LastUptimeSeconds    *int64  `json:"last_uptime_seconds,omitempty"`
}
