package sqlserverdsn

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"time"

	_ "github.com/microsoft/go-mssqldb"
	"github.com/omnihance/omnihance-a3-agent/internal/utils/userdsn"
)

const (
	DriverName = "SQL Server"
)

var (
	ErrNameRequired     = errors.New("dsn name is required")
	ErrServerRequired   = errors.New("server is required")
	ErrDatabaseRequired = errors.New("database is required")
	ErrLoginIDRequired  = errors.New("login id is required")
	ErrPasswordRequired = errors.New("password is required")
)

type DSN struct {
	Name        string `json:"name"`
	Server      string `json:"server"`
	Database    string `json:"database"`
	LoginID     string `json:"login_id"`
	Password    string `json:"-"`
	Description string `json:"description,omitempty"`
	LastUser    string `json:"last_user,omitempty"`
}

type Service struct {
	manager userdsn.Manager
}

func NewService(manager userdsn.Manager) *Service {
	return &Service{manager: manager}
}

func (s *Service) List() ([]DSN, error) {
	names, err := s.manager.List()
	if err != nil {
		return nil, err
	}

	dsns := make([]DSN, 0, len(names))
	for _, name := range names {
		cfg, getErr := s.manager.Get(name)
		if getErr != nil {
			continue
		}
		if cfg.Driver != DriverName {
			continue
		}
		dsns = append(dsns, fromConfig(*cfg))
	}

	return dsns, nil
}

func (s *Service) Get(name string) (*DSN, error) {
	cfg, err := s.manager.Get(name)
	if err != nil {
		return nil, err
	}
	if cfg.Driver != DriverName {
		return nil, userdsn.ErrDSNNotFound
	}

	dsn := fromConfig(*cfg)
	return &dsn, nil
}

func (s *Service) Add(dsn DSN) error {
	if err := validateDSN(dsn); err != nil {
		return err
	}
	cfg, err := toConfig(dsn)
	if err != nil {
		return err
	}
	return s.manager.Add(cfg)
}

func (s *Service) Update(dsn DSN) error {
	if err := validateDSN(dsn); err != nil {
		return err
	}
	if _, err := s.Get(dsn.Name); err != nil {
		return err
	}
	cfg, err := toConfig(dsn)
	if err != nil {
		return err
	}
	return s.manager.Update(cfg)
}

func (s *Service) Delete(name string) error {
	if name == "" {
		return ErrNameRequired
	}
	if _, err := s.Get(name); err != nil {
		return err
	}
	return s.manager.Delete(name)
}

func (s *Service) TestConnection(ctx context.Context, dsn DSN) error {
	if err := validateDSN(dsn); err != nil {
		return err
	}

	connectionString := fmt.Sprintf(
		"sqlserver://%s:%s@%s?database=%s&encrypt=disable",
		url.QueryEscape(dsn.LoginID),
		url.QueryEscape(dsn.Password),
		dsn.Server,
		url.QueryEscape(dsn.Database),
	)

	db, err := sql.Open("sqlserver", connectionString)
	if err != nil {
		return err
	}

	testCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := db.PingContext(testCtx); err != nil {
		if closeErr := db.Close(); closeErr != nil {
			return errors.Join(err, closeErr)
		}
		return err
	}

	return db.Close()
}

func validateDSN(dsn DSN) error {
	if dsn.Name == "" {
		return ErrNameRequired
	}
	if dsn.Server == "" {
		return ErrServerRequired
	}
	if dsn.Database == "" {
		return ErrDatabaseRequired
	}
	if dsn.LoginID == "" {
		return ErrLoginIDRequired
	}
	if dsn.Password == "" {
		return ErrPasswordRequired
	}

	return nil
}

func toConfig(dsn DSN) (userdsn.Config, error) {
	driverPath, err := resolveDriverDLLPath()
	if err != nil {
		return userdsn.Config{}, err
	}

	attrs := map[string]string{
		"Driver":             driverPath,
		"Server":             dsn.Server,
		"Database":           dsn.Database,
		"Trusted_Connection": "No",
		"UID":                dsn.LoginID,
		"PWD":                dsn.Password,
	}
	if dsn.Description != "" {
		attrs["Description"] = dsn.Description
	}
	if dsn.LastUser != "" {
		attrs["LastUser"] = dsn.LastUser
	}

	return userdsn.Config{
		Name:   dsn.Name,
		Driver: DriverName,
		Attrs:  attrs,
	}, nil
}

func fromConfig(cfg userdsn.Config) DSN {
	return DSN{
		Name:        cfg.Name,
		Server:      cfg.Attrs["Server"],
		Database:    cfg.Attrs["Database"],
		LoginID:     cfg.Attrs["UID"],
		Password:    cfg.Attrs["PWD"],
		Description: cfg.Attrs["Description"],
		LastUser:    cfg.Attrs["LastUser"],
	}
}
