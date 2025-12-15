package utils

import (
	"context"
)

type contextKey string

const (
	ContextKeyUserId    = "user_id"
	ContextKeyUserRoles = "user_roles"
	ContextKeyUserEmail = "user_email"
	ContextKeySessionID = "session_id"
)

func SetStringInContext(ctx context.Context, key string, value string) context.Context {
	return context.WithValue(ctx, contextKey(key), value)
}

func GetStringFromContext(ctx context.Context, key string) (string, bool) {
	value, ok := ctx.Value(contextKey(key)).(string)
	return value, ok
}

func SetUserIdInContext(ctx context.Context, userId int64) context.Context {
	return context.WithValue(ctx, contextKey(ContextKeyUserId), userId)
}

func GetUserIdFromContext(ctx context.Context) (int64, bool) {
	value, ok := ctx.Value(contextKey(ContextKeyUserId)).(int64)
	return value, ok
}

func SetUserRolesInContext(ctx context.Context, roles []string) context.Context {
	return context.WithValue(ctx, contextKey(ContextKeyUserRoles), roles)
}

func GetUserRolesFromContext(ctx context.Context) ([]string, bool) {
	value, ok := ctx.Value(contextKey(ContextKeyUserRoles)).([]string)
	return value, ok
}

func SetUserEmailInContext(ctx context.Context, email string) context.Context {
	return context.WithValue(ctx, contextKey(ContextKeyUserEmail), email)
}

func GetUserEmailFromContext(ctx context.Context) (string, bool) {
	value, ok := ctx.Value(contextKey(ContextKeyUserEmail)).(string)
	return value, ok
}

func SetSessionIDInContext(ctx context.Context, sessionID string) context.Context {
	return context.WithValue(ctx, contextKey(ContextKeySessionID), sessionID)
}

func GetSessionIDFromContext(ctx context.Context) (string, bool) {
	value, ok := ctx.Value(contextKey(ContextKeySessionID)).(string)
	return value, ok
}
