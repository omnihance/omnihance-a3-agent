//go:build windows

package userdsn

import (
	"fmt"

	"golang.org/x/sys/windows/registry"
)

const (
	odbcDataSources = `Software\ODBC\ODBC.INI\ODBC Data Sources`
	odbcIniBase     = `Software\ODBC\ODBC.INI\`
)

// windowsManager implements Manager on Windows via the registry.
type windowsManager struct{}

// New returns a Manager backed by the Windows registry.
func New() Manager {
	return &windowsManager{}
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

// List returns all User DSN names from the ODBC Data Sources registry key.
func (m *windowsManager) List() ([]string, error) {
	key, err := registry.OpenKey(registry.CURRENT_USER, odbcDataSources, registry.QUERY_VALUE)
	if err != nil {
		// Key may not exist yet if no DSNs have been created.
		if err == registry.ErrNotExist {
			return []string{}, nil
		}
		return nil, fmt.Errorf("userdsn: open ODBC Data Sources: %w", err)
	}
	defer key.Close()

	names, err := key.ReadValueNames(-1)
	if err != nil {
		return nil, fmt.Errorf("userdsn: read DSN names: %w", err)
	}
	return names, nil
}

// Get returns the full Config for the named DSN.
func (m *windowsManager) Get(name string) (*Config, error) {
	if name == "" {
		return nil, ErrDSNNameRequired
	}

	sourcesKey, err := registry.OpenKey(registry.CURRENT_USER, odbcDataSources, registry.QUERY_VALUE)
	if err != nil {
		if err == registry.ErrNotExist {
			return nil, ErrDSNNotFound
		}
		return nil, fmt.Errorf("userdsn: open ODBC Data Sources: %w", err)
	}
	defer sourcesKey.Close()

	driver, _, err := sourcesKey.GetStringValue(name)
	if err != nil {
		if err == registry.ErrNotExist {
			return nil, ErrDSNNotFound
		}
		return nil, fmt.Errorf("userdsn: read driver mapping for %q: %w", name, err)
	}

	dsnKey, err := registry.OpenKey(registry.CURRENT_USER, odbcIniBase+name, registry.QUERY_VALUE)
	if err != nil {
		if err == registry.ErrNotExist {
			return nil, ErrDSNNotFound
		}
		return nil, fmt.Errorf("userdsn: open DSN key %q: %w", name, err)
	}
	defer dsnKey.Close()

	valueNames, err := dsnKey.ReadValueNames(-1)
	if err != nil {
		return nil, fmt.Errorf("userdsn: read values for DSN %q: %w", name, err)
	}

	attrs := make(map[string]string, len(valueNames))

	for _, vn := range valueNames {
		val, _, err := dsnKey.GetStringValue(vn)
		if err != nil {
			continue // skip non-string values
		}
		attrs[vn] = val
	}

	return &Config{
		Name:   name,
		Driver: driver,
		Attrs:  attrs,
	}, nil
}

// Add creates a new User DSN. Returns ErrDSNAlreadyExists if the DSN is already registered.
func (m *windowsManager) Add(cfg Config) error {
	if err := validate(cfg); err != nil {
		return err
	}

	// Check for duplicate.
	existing, err := m.List()
	if err != nil {
		return err
	}
	for _, n := range existing {
		if n == cfg.Name {
			return ErrDSNAlreadyExists
		}
	}

	// Register in ODBC Data Sources (DSN name -> driver).
	sourcesKey, _, err := registry.CreateKey(registry.CURRENT_USER, odbcDataSources, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("userdsn: open ODBC Data Sources: %w", err)
	}
	defer sourcesKey.Close()

	if err := sourcesKey.SetStringValue(cfg.Name, cfg.Driver); err != nil {
		return fmt.Errorf("userdsn: register DSN name %q: %w", cfg.Name, err)
	}

	if err := writeAttrs(cfg); err != nil {
		_ = sourcesKey.DeleteValue(cfg.Name)
		_ = registry.DeleteKey(registry.CURRENT_USER, odbcIniBase+cfg.Name)
		return err
	}

	return nil
}

// Update overwrites the settings of an existing DSN. Returns ErrDSNNotFound if absent.
func (m *windowsManager) Update(cfg Config) error {
	if err := validate(cfg); err != nil {
		return err
	}

	// Confirm DSN exists.
	if _, err := m.Get(cfg.Name); err != nil {
		return err // ErrDSNNotFound or other
	}

	// Update the driver mapping in ODBC Data Sources.
	sourcesKey, err := registry.OpenKey(registry.CURRENT_USER, odbcDataSources, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("userdsn: open ODBC Data Sources: %w", err)
	}
	defer sourcesKey.Close()

	if err := sourcesKey.SetStringValue(cfg.Name, cfg.Driver); err != nil {
		return fmt.Errorf("userdsn: update driver mapping for %q: %w", cfg.Name, err)
	}

	return writeAttrs(cfg)
}

// Delete removes a User DSN entirely. Returns ErrDSNNotFound if absent.
func (m *windowsManager) Delete(name string) error {
	if name == "" {
		return ErrDSNNameRequired
	}

	// Confirm DSN exists.
	if _, err := m.Get(name); err != nil {
		return err
	}

	// Remove name from ODBC Data Sources.
	sourcesKey, err := registry.OpenKey(registry.CURRENT_USER, odbcDataSources, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("userdsn: open ODBC Data Sources: %w", err)
	}
	defer sourcesKey.Close()

	if err := sourcesKey.DeleteValue(name); err != nil {
		return fmt.Errorf("userdsn: remove DSN entry %q: %w", name, err)
	}

	// Delete the DSN registry key.
	if err := registry.DeleteKey(registry.CURRENT_USER, odbcIniBase+name); err != nil {
		return fmt.Errorf("userdsn: delete DSN key %q: %w", name, err)
	}

	return nil
}

// writeAttrs creates (or replaces) the DSN-specific registry key and writes all attributes.
func writeAttrs(cfg Config) error {
	dsnKey, _, err := registry.CreateKey(registry.CURRENT_USER, odbcIniBase+cfg.Name, registry.SET_VALUE|registry.QUERY_VALUE)
	if err != nil {
		return fmt.Errorf("userdsn: create DSN key %q: %w", cfg.Name, err)
	}
	defer dsnKey.Close()

	valueNames, err := dsnKey.ReadValueNames(-1)
	if err != nil {
		return fmt.Errorf("userdsn: read existing values for DSN %q: %w", cfg.Name, err)
	}

	for _, valueName := range valueNames {
		if err := dsnKey.DeleteValue(valueName); err != nil {
			return fmt.Errorf("userdsn: delete stale value %q for DSN %q: %w", valueName, cfg.Name, err)
		}
	}

	for k, v := range cfg.Attrs {
		if err := dsnKey.SetStringValue(k, v); err != nil {
			return fmt.Errorf("userdsn: set %q for DSN %q: %w", k, cfg.Name, err)
		}
	}
	return nil
}
