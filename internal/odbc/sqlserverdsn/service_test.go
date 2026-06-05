package sqlserverdsn

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/omnihance/omnihance-a3-agent/internal/utils/userdsn"
)

type fakeManager struct {
	items map[string]userdsn.Config
}

func newFakeManager() *fakeManager {
	return &fakeManager{items: map[string]userdsn.Config{}}
}

func (m *fakeManager) List() ([]string, error) {
	names := make([]string, 0, len(m.items))
	for name := range m.items {
		names = append(names, name)
	}
	return names, nil
}

func (m *fakeManager) Get(name string) (*userdsn.Config, error) {
	cfg, ok := m.items[name]
	if !ok {
		return nil, userdsn.ErrDSNNotFound
	}
	return &cfg, nil
}

func (m *fakeManager) Add(cfg userdsn.Config) error {
	if _, exists := m.items[cfg.Name]; exists {
		return userdsn.ErrDSNAlreadyExists
	}
	m.items[cfg.Name] = cfg
	return nil
}

func (m *fakeManager) Update(cfg userdsn.Config) error {
	if _, exists := m.items[cfg.Name]; !exists {
		return userdsn.ErrDSNNotFound
	}
	m.items[cfg.Name] = cfg
	return nil
}

func (m *fakeManager) Delete(name string) error {
	if _, exists := m.items[name]; !exists {
		return userdsn.ErrDSNNotFound
	}
	delete(m.items, name)
	return nil
}

func TestValidateDSN(t *testing.T) {
	err := validatePersistDSN(DSN{})
	if !errors.Is(err, ErrNameRequired) {
		t.Fatalf("expected ErrNameRequired, got %v", err)
	}

	err = validatePersistDSN(DSN{
		Name:     "A3",
		Server:   "localhost",
		Database: "A3",
		LoginID:  "sa",
	})
	if err != nil {
		t.Fatalf("persist validation should not require password: %v", err)
	}

	err = validateTestDSN(DSN{
		Name:     "A3",
		Server:   "localhost",
		Database: "A3",
		LoginID:  "sa",
	})
	if !errors.Is(err, ErrPasswordRequired) {
		t.Fatalf("expected ErrPasswordRequired, got %v", err)
	}
}

func TestFromConfig(t *testing.T) {
	result := fromConfig(userdsn.Config{
		Name:   "A3",
		Driver: DriverName,
		Attrs: map[string]string{
			"Server":   "localhost",
			"Database": "A3DB",
			"UID":      "sa",
			"PWD":      "pw",
		},
	})
	if result.Name != "A3" || result.Server != "localhost" || result.Database != "A3DB" {
		t.Fatalf("unexpected mapping result: %+v", result)
	}
	if result.Password != "" {
		t.Fatalf("password should not be read from persisted attrs: %+v", result)
	}
}

func TestToConfigMatchesA3RegistryExport(t *testing.T) {
	originalResolveDriverPath := resolveDriverPath
	resolveDriverPath = func() (string, error) {
		return `C:\WINDOWS\system32\SQLSRV32.dll`, nil
	}
	defer func() {
		resolveDriverPath = originalResolveDriverPath
	}()

	cfg, err := toConfig(DSN{
		Name:     "ASD",
		Server:   `DESKTOP-PE9M1HH\SQLEXPRESS`,
		Database: "ASD",
		LoginID:  "sa",
		Password: "secret",
	})
	if err != nil {
		t.Fatalf("toConfig failed: %v", err)
	}

	expected := map[string]string{
		"Driver":   `C:\WINDOWS\system32\SQLSRV32.dll`,
		"Server":   `DESKTOP-PE9M1HH\SQLEXPRESS`,
		"Database": "ASD",
		"LastUser": "sa",
	}
	if cfg.Name != "ASD" || cfg.Driver != DriverName {
		t.Fatalf("unexpected config metadata: %+v", cfg)
	}
	if len(cfg.Attrs) != len(expected) {
		t.Fatalf("unexpected attrs: %+v", cfg.Attrs)
	}
	for key, value := range expected {
		if cfg.Attrs[key] != value {
			t.Fatalf("attr %s = %q, want %q", key, cfg.Attrs[key], value)
		}
	}
	for _, key := range []string{"UID", "Trusted_Connection", "Description"} {
		if _, ok := cfg.Attrs[key]; ok {
			t.Fatalf("unexpected persisted attr %s", key)
		}
	}
}

func TestServiceUpdateRemovesPersistedPassword(t *testing.T) {
	originalResolveDriverPath := resolveDriverPath
	resolveDriverPath = func() (string, error) {
		return `C:\WINDOWS\system32\SQLSRV32.dll`, nil
	}
	defer func() {
		resolveDriverPath = originalResolveDriverPath
	}()

	manager := newFakeManager()
	manager.items["A3"] = userdsn.Config{
		Name:   "A3",
		Driver: DriverName,
		Attrs: map[string]string{
			"Server":   "old",
			"Database": "old",
			"LastUser": "sa",
			"PWD":      "existing-password",
		},
	}

	service := NewService(manager)
	err := service.Update(DSN{
		Name:     "A3",
		Server:   "new",
		Database: "newdb",
		LoginID:  "sa",
		Password: "new-password",
	})
	if err != nil {
		t.Fatalf("update failed: %v", err)
	}

	updated := manager.items["A3"]
	if _, ok := updated.Attrs["PWD"]; ok {
		t.Fatalf("password should not be persisted: %+v", updated.Attrs)
	}
}

func TestServiceCRUD(t *testing.T) {
	manager := newFakeManager()
	service := NewService(manager)

	manager.items["A3"] = userdsn.Config{
		Name:   "A3",
		Driver: DriverName,
		Attrs: map[string]string{
			"Server":             "localhost",
			"Database":           "a3",
			"UID":                "sa",
			"PWD":                "pw",
			"Trusted_Connection": "No",
		},
	}

	items, err := service.List()
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}

	dsn, err := service.Get("A3")
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if dsn.Name != "A3" {
		t.Fatalf("unexpected dsn: %+v", dsn)
	}

	if err := service.Delete("A3"); err != nil {
		t.Fatalf("delete failed: %v", err)
	}
}

func TestCreateDefaultsSkipsExistingDSNs(t *testing.T) {
	originalResolveDriverPath := resolveDriverPath
	resolveDriverPath = func() (string, error) {
		return `C:\WINDOWS\system32\SQLSRV32.dll`, nil
	}
	defer func() {
		resolveDriverPath = originalResolveDriverPath
	}()

	manager := newFakeManager()
	manager.items["ASD"] = userdsn.Config{
		Name:   "ASD",
		Driver: DriverName,
		Attrs: map[string]string{
			"Server":   "old",
			"Database": "old",
		},
	}

	service := NewService(manager)
	result, err := service.CreateDefaults(`DESKTOP-PE9M1HH\SQLEXPRESS`, "sa")
	if err != nil {
		t.Fatalf("CreateDefaults failed: %v", err)
	}
	if len(result.Created) != len(defaultDSNs)-1 {
		t.Fatalf("created %d defaults, want %d", len(result.Created), len(defaultDSNs)-1)
	}
	if len(result.Skipped) != 1 || result.Skipped[0] != "ASD" {
		t.Fatalf("unexpected skipped defaults: %+v", result.Skipped)
	}
	if manager.items["ASD"].Attrs["Server"] != "old" {
		t.Fatal("existing DSN was overwritten")
	}

	itemEvent := manager.items["A3ItemEvent"]
	if itemEvent.Attrs["Database"] != "A3ItemEvent" ||
		itemEvent.Attrs["LastUser"] != "sa" ||
		itemEvent.Attrs["Server"] != `DESKTOP-PE9M1HH\SQLEXPRESS` {
		t.Fatalf("unexpected default DSN attrs: %+v", itemEvent.Attrs)
	}
}

func TestServiceRejectsNonSQLServerUpdateAndDelete(t *testing.T) {
	manager := newFakeManager()
	service := NewService(manager)

	manager.items["Other"] = userdsn.Config{
		Name:   "Other",
		Driver: "Other Driver",
		Attrs: map[string]string{
			"Server":   "localhost",
			"Database": "a3",
			"UID":      "sa",
			"PWD":      "pw",
		},
	}

	err := service.Update(DSN{
		Name:     "Other",
		Server:   "localhost",
		Database: "a3",
		LoginID:  "sa",
		Password: "pw",
	})
	if !errors.Is(err, userdsn.ErrDSNNotFound) {
		t.Fatalf("expected ErrDSNNotFound on update, got %v", err)
	}

	err = service.Delete("Other")
	if !errors.Is(err, userdsn.ErrDSNNotFound) {
		t.Fatalf("expected ErrDSNNotFound on delete, got %v", err)
	}

	if _, ok := manager.items["Other"]; !ok {
		t.Fatal("non SQL Server DSN was deleted")
	}
}

func TestDSNJSONDoesNotExposePassword(t *testing.T) {
	data, err := json.Marshal(DSN{
		Name:     "A3",
		Server:   "localhost",
		Database: "a3",
		LoginID:  "sa",
		Password: "pw",
	})
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	if string(data) != `{"name":"A3","server":"localhost","database":"a3","login_id":"sa"}` {
		t.Fatalf("unexpected json: %s", data)
	}
}

func TestServiceTestConnectionValidation(t *testing.T) {
	service := NewService(newFakeManager())
	err := service.TestConnection(context.Background(), DSN{})
	if !errors.Is(err, ErrNameRequired) {
		t.Fatalf("expected validation error, got %v", err)
	}
}
