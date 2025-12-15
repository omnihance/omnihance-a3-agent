package db

import (
	"fmt"
	"time"

	"github.com/doug-martin/goqu/v9"
	"github.com/google/uuid"
	"github.com/omnihance/omnihance-a3-agent/internal/logger"
)

type Session struct {
	SessionID      string  `db:"session_id" json:"session_id"`
	UserID         int64   `db:"user_id" json:"user_id"`
	CreatedAt      int64   `db:"created_at" json:"created_at"`
	ExpiresAt      int64   `db:"expires_at" json:"expires_at"`
	LastAccessedAt int64   `db:"last_accessed_at" json:"last_accessed_at"`
	UserAgent      *string `db:"user_agent" json:"user_agent"`
	IPAddress      *string `db:"ip_address" json:"ip_address"`
}

func (s *sqliteInternalDB) CreateSession(userID int64, expiresAt time.Time, userAgent, ipAddress *string) (*Session, error) {
	sessionID := uuid.New().String()

	session := Session{
		SessionID:      sessionID,
		UserID:         userID,
		CreatedAt:      time.Now().Unix(),
		ExpiresAt:      expiresAt.Unix(),
		LastAccessedAt: time.Now().Unix(),
		UserAgent:      userAgent,
		IPAddress:      ipAddress,
	}

	_, err := s.goqu.Insert("sessions").
		Prepared(true).
		Rows(goqu.Record{
			"session_id":       session.SessionID,
			"user_id":          session.UserID,
			"created_at":       session.CreatedAt,
			"expires_at":       session.ExpiresAt,
			"last_accessed_at": session.LastAccessedAt,
			"user_agent":       session.UserAgent,
			"ip_address":       session.IPAddress,
		}).
		Executor().
		Exec()
	if err != nil {
		s.logger.Error(
			"failed to create session",
			logger.Field{Key: "user_id", Value: userID},
			logger.Field{Key: "error", Value: err},
		)
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	return &session, nil
}

func (s *sqliteInternalDB) GetSession(sessionID string) (*Session, error) {
	var session Session
	found, err := s.goqu.From("sessions").
		Prepared(true).
		Where(goqu.Ex{"session_id": sessionID}).
		ScanStruct(&session)
	if err != nil {
		s.logger.Error(
			"failed to get session",
			logger.Field{Key: "session_id", Value: sessionID},
			logger.Field{Key: "error", Value: err},
		)
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	if !found {
		return nil, fmt.Errorf("session not found")
	}

	if time.Now().Unix() > session.ExpiresAt {
		return nil, fmt.Errorf("session expired")
	}

	return &session, nil
}

func (s *sqliteInternalDB) UpdateSessionLastAccessed(sessionID string) error {
	_, err := s.goqu.Update("sessions").
		Prepared(true).
		Set(goqu.Record{
			"last_accessed_at": time.Now(),
		}).
		Where(goqu.Ex{"session_id": sessionID}).
		Executor().
		Exec()
	if err != nil {
		s.logger.Error(
			"failed to update session last accessed",
			logger.Field{Key: "session_id", Value: sessionID},
			logger.Field{Key: "error", Value: err},
		)
		return fmt.Errorf("failed to update session last accessed: %w", err)
	}

	return nil
}

func (s *sqliteInternalDB) DeleteSession(sessionID string) error {
	_, err := s.goqu.Delete("sessions").
		Prepared(true).
		Where(goqu.Ex{"session_id": sessionID}).
		Executor().
		Exec()
	if err != nil {
		s.logger.Error(
			"failed to delete session",
			logger.Field{Key: "session_id", Value: sessionID},
			logger.Field{Key: "error", Value: err},
		)
		return fmt.Errorf("failed to delete session: %w", err)
	}

	return nil
}

func (s *sqliteInternalDB) DeleteUserSessions(userID int64) error {
	_, err := s.goqu.Delete("sessions").
		Prepared(true).
		Where(goqu.Ex{"user_id": userID}).
		Executor().
		Exec()
	if err != nil {
		s.logger.Error(
			"failed to delete user sessions",
			logger.Field{Key: "user_id", Value: userID},
			logger.Field{Key: "error", Value: err},
		)
		return fmt.Errorf("failed to delete user sessions: %w", err)
	}

	return nil
}

func (s *sqliteInternalDB) DeleteExpiredSessions() error {
	_, err := s.goqu.Delete("sessions").
		Prepared(true).
		Where(goqu.C("expires_at").Lt(time.Now())).
		Executor().
		Exec()
	if err != nil {
		s.logger.Error(
			"failed to delete expired sessions",
			logger.Field{Key: "error", Value: err},
		)
		return fmt.Errorf("failed to delete expired sessions: %w", err)
	}

	return nil
}
