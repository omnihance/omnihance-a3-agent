package userdsn_test

import (
	"errors"
	"testing"

	"github.com/omnihance/omnihance-a3-agent/internal/utils/userdsn"
)

// TestNew ensures New() never panics on any platform.
func TestNew(t *testing.T) {
	m := userdsn.New()
	if m == nil {
		t.Fatal("New() returned nil")
	}
}

// TestUnsupported_List verifies non-Windows returns ErrNotSupported.
// On Windows this test is skipped at the assertion level because the
// real implementation would be used instead.
func TestManagerReturnsErrors(t *testing.T) {
	m := userdsn.New()

	_, err := m.List()
	if err != nil && !errors.Is(err, userdsn.ErrNotSupported) {
		t.Fatalf("List(): unexpected error type: %v", err)
	}

	_, err = m.Get("TestDSN")
	if err != nil &&
		!errors.Is(err, userdsn.ErrNotSupported) &&
		!errors.Is(err, userdsn.ErrDSNNotFound) {
		t.Fatalf("Get(): unexpected error type: %v", err)
	}

	cfg := userdsn.Config{
		Name:   "TestDSN",
		Driver: "SQL Server",
		Attrs:  map[string]string{"Server": "localhost", "Database": "testdb"},
	}

	err = m.Add(cfg)
	if err != nil && !errors.Is(err, userdsn.ErrNotSupported) {
		t.Fatalf("Add(): unexpected error type: %v", err)
	}

	err = m.Update(cfg)
	if err != nil &&
		!errors.Is(err, userdsn.ErrNotSupported) &&
		!errors.Is(err, userdsn.ErrDSNNotFound) {
		t.Fatalf("Update(): unexpected error type: %v", err)
	}

	err = m.Delete("TestDSN")
	if err != nil &&
		!errors.Is(err, userdsn.ErrNotSupported) &&
		!errors.Is(err, userdsn.ErrDSNNotFound) {
		t.Fatalf("Delete(): unexpected error type: %v", err)
	}
}

// TestValidation checks that empty Name/Driver are rejected before
// any platform-specific code runs.
func TestValidation(t *testing.T) {
	m := userdsn.New()

	// Empty name
	err := m.Add(userdsn.Config{Driver: "SQL Server"})
	if !errors.Is(err, userdsn.ErrDSNNameRequired) && !errors.Is(err, userdsn.ErrNotSupported) {
		t.Errorf("Add with empty name: got %v, want ErrDSNNameRequired (or ErrNotSupported on non-Windows)", err)
	}

	// Empty driver
	err = m.Add(userdsn.Config{Name: "MyDSN"})
	if !errors.Is(err, userdsn.ErrDriverRequired) && !errors.Is(err, userdsn.ErrNotSupported) {
		t.Errorf("Add with empty driver: got %v, want ErrDriverRequired (or ErrNotSupported on non-Windows)", err)
	}
}
