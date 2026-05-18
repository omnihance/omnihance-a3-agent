//go:build !windows

package services

func localSQLServerServiceRunning() (bool, error) {
	return true, nil
}
