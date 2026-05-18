//go:build windows

package services

import (
	"strings"

	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

func localSQLServerServiceRunning() (bool, error) {
	manager, err := mgr.Connect()
	if err != nil {
		return false, err
	}
	defer func() {
		_ = manager.Disconnect()
	}()

	serviceNames, err := manager.ListServices()
	if err != nil {
		return false, err
	}

	for _, serviceName := range serviceNames {
		upperName := strings.ToUpper(serviceName)
		if upperName != "MSSQLSERVER" && !strings.HasPrefix(upperName, "MSSQL$") {
			continue
		}

		service, err := manager.OpenService(serviceName)
		if err != nil {
			continue
		}

		status, queryErr := service.Query()
		_ = service.Close()
		if queryErr != nil {
			continue
		}

		if status.State == svc.Running {
			return true, nil
		}
	}

	return false, nil
}
