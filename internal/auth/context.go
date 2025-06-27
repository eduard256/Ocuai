package auth

import (
	"context"
)

// ContextKey тип для ключей контекста
type ContextKey string

const (
	// SessionContextKey ключ для сессии в контексте
	SessionContextKey ContextKey = "session"
)

// SetSessionContext устанавливает сессию в контекст
func SetSessionContext(ctx context.Context, session *Session) context.Context {
	return context.WithValue(ctx, SessionContextKey, session)
}

// GetSessionFromContext получает сессию из контекста
func GetSessionFromContext(ctx context.Context) *Session {
	session, ok := ctx.Value(SessionContextKey).(*Session)
	if !ok {
		return nil
	}
	return session
}
