//go:build windows

package sqlserverdsn

import (
	"fmt"

	"golang.org/x/sys/windows/registry"
)

var odbcInstDriverKeys = []string{
	`Software\WOW6432Node\ODBC\ODBCINST.INI\SQL Server`,
	`Software\ODBC\ODBCINST.INI\SQL Server`,
}

func resolveDriverDLLPath() (string, error) {
	for _, driverKey := range odbcInstDriverKeys {
		key, err := registry.OpenKey(registry.LOCAL_MACHINE, driverKey, registry.QUERY_VALUE)
		if err != nil {
			continue
		}
		defer func() {
			_ = key.Close()
		}()

		value, _, err := key.GetStringValue("Driver")
		if err != nil || value == "" {
			continue
		}

		return value, nil
	}

	return "", fmt.Errorf("sql server odbc driver path not found")
}
