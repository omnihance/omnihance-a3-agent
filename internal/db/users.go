package db

import (
	"fmt"

	"github.com/doug-martin/goqu/v9"
	"github.com/omnihance/omnihance-a3-agent/internal/constants"
	"github.com/omnihance/omnihance-a3-agent/internal/logger"
)

type User struct {
	ID        int64  `db:"id" json:"id"`
	IsDeleted bool   `db:"is_deleted" json:"is_deleted"`
	Email     string `db:"email" json:"email"`
	Password  string `db:"password" json:"password"`
	Roles     string `db:"roles" json:"roles"`
	Status    string `db:"status" json:"status"`
	CreatedBy *int64 `db:"created_by" json:"created_by"`
	CreatedAt int64  `db:"created_at" json:"created_at"`
	UpdatedBy *int64 `db:"updated_by" json:"updated_by"`
	UpdatedAt *int64 `db:"updated_at" json:"updated_at"`
	DeletedBy *int64 `db:"deleted_by" json:"deleted_by"`
	DeletedAt *int64 `db:"deleted_at" json:"deleted_at"`
}

func (s *sqliteInternalDB) GetUserByID(userID int64) (*User, error) {
	var user User
	found, err := s.goqu.From("users").
		Prepared(true).
		Where(goqu.Ex{"id": userID, "is_deleted": false}).
		ScanStruct(&user)
	if err != nil {
		s.logger.Error(
			"failed to get user by id",
			logger.Field{Key: "user_id", Value: userID},
			logger.Field{Key: "error", Value: err},
		)
		return nil, fmt.Errorf("failed to get user by id: %w", err)
	}

	if !found {
		return nil, fmt.Errorf("user not found")
	}

	return &user, nil
}

func (s *sqliteInternalDB) GetActiveUserByID(userID int64) (*User, error) {
	var user User
	found, err := s.goqu.From("users").
		Prepared(true).
		Where(goqu.Ex{"id": userID, "is_deleted": false, "status": constants.UserStatusActive}).
		ScanStruct(&user)
	if err != nil {
		s.logger.Error(
			"failed to get active user by id",
			logger.Field{Key: "user_id", Value: userID},
			logger.Field{Key: "error", Value: err},
		)
		return nil, fmt.Errorf("failed to get active user by id: %w", err)
	}

	if !found {
		return nil, fmt.Errorf("user not found or not active")
	}

	return &user, nil
}

func (s *sqliteInternalDB) GetUserByEmail(email string) (*User, error) {
	var user User
	found, err := s.goqu.From("users").
		Prepared(true).
		Where(goqu.Ex{"email": email, "is_deleted": false}).
		ScanStruct(&user)
	if err != nil {
		s.logger.Error(
			"failed to get user by email",
			logger.Field{Key: "email", Value: email},
			logger.Field{Key: "error", Value: err},
		)
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}

	if !found {
		return nil, fmt.Errorf("user not found")
	}

	return &user, nil
}

func (s *sqliteInternalDB) GetUserByIDIncludeDeleted(userID int64) (*User, error) {
	var user User
	found, err := s.goqu.From("users").
		Prepared(true).
		Where(goqu.Ex{"id": userID}).
		ScanStruct(&user)
	if err != nil {
		s.logger.Error(
			"failed to get user by id including deleted",
			logger.Field{Key: "user_id", Value: userID},
			logger.Field{Key: "error", Value: err},
		)
		return nil, fmt.Errorf("failed to get user by id: %w", err)
	}

	if !found {
		return nil, fmt.Errorf("user not found")
	}

	return &user, nil
}

func (s *sqliteInternalDB) GetUserByEmailIncludeDeleted(email string) (*User, error) {
	var user User
	found, err := s.goqu.From("users").
		Prepared(true).
		Where(goqu.Ex{"email": email}).
		ScanStruct(&user)
	if err != nil {
		s.logger.Error(
			"failed to get user by email including deleted",
			logger.Field{Key: "email", Value: email},
			logger.Field{Key: "error", Value: err},
		)
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}

	if !found {
		return nil, fmt.Errorf("user not found")
	}

	return &user, nil
}

func (s *sqliteInternalDB) GetUsers() ([]User, error) {
	var users []User
	err := s.goqu.From("users").
		Prepared(true).
		Where(goqu.Ex{"is_deleted": false}).
		Order(goqu.I("created_at").Desc()).
		ScanStructs(&users)
	if err != nil {
		s.logger.Error(
			"failed to get users",
			logger.Field{Key: "error", Value: err},
		)
		return nil, fmt.Errorf("failed to get users: %w", err)
	}

	return users, nil
}

func (s *sqliteInternalDB) CreateUser(email string, password string, roles string, createdBy *int64) (*User, error) {
	return s.CreateUserWithStatus(email, password, roles, constants.UserStatusPending, createdBy)
}

func (s *sqliteInternalDB) CreateUserWithStatus(email string, password string, roles string, status string, createdBy *int64) (*User, error) {
	insertRecord := goqu.Record{
		"email":      email,
		"password":   password,
		"roles":      roles,
		"created_at": goqu.L("strftime('%s', 'now')"),
	}

	if status != "" {
		insertRecord["status"] = status
	}

	if createdBy != nil {
		insertRecord["created_by"] = *createdBy
	}

	result, err := s.goqu.Insert("users").
		Prepared(true).
		Rows(insertRecord).
		Executor().
		Exec()
	if err != nil {
		s.logger.Error(
			"failed to create user",
			logger.Field{Key: "email", Value: email},
			logger.Field{Key: "created_by", Value: createdBy},
			logger.Field{Key: "error", Value: err},
		)
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	userID, err := result.LastInsertId()
	if err != nil {
		s.logger.Error(
			"failed to get last insert id",
			logger.Field{Key: "error", Value: err},
		)
		return nil, fmt.Errorf("failed to get last insert id: %w", err)
	}

	return s.GetUserByID(userID)
}

func (s *sqliteInternalDB) UpdateUserPassword(userID int64, newPassword string, updatedBy int64) error {
	_, err := s.goqu.Update("users").
		Prepared(true).
		Set(goqu.Record{
			"password":   newPassword,
			"updated_by": updatedBy,
			"updated_at": goqu.L("strftime('%s', 'now')"),
		}).
		Where(goqu.Ex{"id": userID, "is_deleted": false}).
		Executor().
		Exec()
	if err != nil {
		s.logger.Error(
			"failed to update user password",
			logger.Field{Key: "user_id", Value: userID},
			logger.Field{Key: "updated_by", Value: updatedBy},
			logger.Field{Key: "error", Value: err},
		)
		return fmt.Errorf("failed to update user password: %w", err)
	}

	return nil
}

func (s *sqliteInternalDB) UpdateUserRoles(userID int64, roles string, updatedBy int64) error {
	_, err := s.goqu.Update("users").
		Prepared(true).
		Set(goqu.Record{
			"roles":      roles,
			"updated_by": updatedBy,
			"updated_at": goqu.L("strftime('%s', 'now')"),
		}).
		Where(goqu.Ex{"id": userID, "is_deleted": false}).
		Executor().
		Exec()
	if err != nil {
		s.logger.Error(
			"failed to update user roles",
			logger.Field{Key: "user_id", Value: userID},
			logger.Field{Key: "roles", Value: roles},
			logger.Field{Key: "updated_by", Value: updatedBy},
			logger.Field{Key: "error", Value: err},
		)
		return fmt.Errorf("failed to update user roles: %w", err)
	}

	return nil
}

func (s *sqliteInternalDB) DeleteUser(userID int64, deletedBy int64) error {
	_, err := s.goqu.Update("users").
		Prepared(true).
		Set(goqu.Record{
			"is_deleted": true,
			"deleted_by": deletedBy,
			"deleted_at": goqu.L("strftime('%s', 'now')"),
		}).
		Where(goqu.Ex{"id": userID, "is_deleted": false}).
		Executor().
		Exec()
	if err != nil {
		s.logger.Error(
			"failed to delete user",
			logger.Field{Key: "user_id", Value: userID},
			logger.Field{Key: "deleted_by", Value: deletedBy},
			logger.Field{Key: "error", Value: err},
		)
		return fmt.Errorf("failed to delete user: %w", err)
	}

	return nil
}

func (s *sqliteInternalDB) GetAdminUserCount() (int64, error) {
	count, err := s.goqu.From("users").
		Prepared(true).
		Where(goqu.Ex{"roles": constants.RoleSuperAdmin, "is_deleted": false}).
		Count()
	if err != nil {
		return 0, fmt.Errorf("failed to get admin user count: %w", err)
	}

	return count, nil
}
