// Package userdsn provides CRUD operations for Windows ODBC User DSNs.
// On non-Windows platforms, all operations return ErrNotSupported.
package userdsn

import "errors"

// ErrNotSupported is returned on non-Windows platforms.
var ErrNotSupported = errors.New("ODBC User DSN management is only supported on Windows")

// ErrDSNNotFound is returned when a DSN does not exist.
var ErrDSNNotFound = errors.New("DSN not found")

// ErrDSNAlreadyExists is returned when attempting to add a DSN that already exists.
var ErrDSNAlreadyExists = errors.New("DSN already exists")

// ErrDSNNameRequired is returned when DSN name is empty.
var ErrDSNNameRequired = errors.New("DSN name is required")

// ErrDriverRequired is returned when Driver is empty.
var ErrDriverRequired = errors.New("driver name is required")

// Config holds the configuration for an ODBC User DSN.
// The Attrs map holds driver-specific key-value pairs
// (e.g. "Server", "Database", "Port", "Trusted_Connection").
type Config struct {
	Name   string            // DSN name (required)
	Driver string            // ODBC driver name, e.g. "SQL Server", "MySQL ODBC 8.0 Driver"
	Attrs  map[string]string // Driver-specific attributes
}

// Manager defines the operations available for User DSN management.
type Manager interface {
	// List returns all User DSN names registered in ODBC Data Sources.
	List() ([]string, error)

	// Get returns the full Config for the named DSN.
	Get(name string) (*Config, error)

	// Add creates a new User DSN. Returns ErrDSNAlreadyExists if it already exists.
	Add(cfg Config) error

	// Update overwrites settings on an existing DSN. Returns ErrDSNNotFound if missing.
	Update(cfg Config) error

	// Delete removes a User DSN. Returns ErrDSNNotFound if missing.
	Delete(name string) error
}

// validate checks the required fields of a Config.
func validate(cfg Config) error {
	if cfg.Name == "" {
		return ErrDSNNameRequired
	}
	if cfg.Driver == "" {
		return ErrDriverRequired
	}
	return nil
}
