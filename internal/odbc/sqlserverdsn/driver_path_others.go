//go:build !windows

package sqlserverdsn

import "errors"

func resolveDriverDLLPath() (string, error) {
	return "", errors.New("sql server odbc management is only supported on windows")
}
