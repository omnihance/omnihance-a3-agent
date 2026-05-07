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
	err := validateDSN(DSN{})
	if !errors.Is(err, ErrNameRequired) {
		t.Fatalf("expected ErrNameRequired, got %v", err)
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
