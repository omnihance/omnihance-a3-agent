//go:build !windows

package userdsn

// unsupportedManager is returned on non-Windows platforms.
type unsupportedManager struct{}

// New returns a Manager that always returns ErrNotSupported.
// ODBC User DSN management is a Windows-only concept.
func New() Manager {
	return &unsupportedManager{}
}

func (m *unsupportedManager) List() ([]string, error) {
	return nil, ErrNotSupported
}

func (m *unsupportedManager) Get(name string) (*Config, error) {
	return nil, ErrNotSupported
}

func (m *unsupportedManager) Add(cfg Config) error {
	return ErrNotSupported
}

func (m *unsupportedManager) Update(cfg Config) error {
	return ErrNotSupported
}

func (m *unsupportedManager) Delete(name string) error {
	return ErrNotSupported
}
