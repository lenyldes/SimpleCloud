package auth

import (
	"context"

	"github.com/google/uuid"
)

type contextKey string

const userIDKey contextKey = "user_id"

// WithUserID returns a new context with the given user_id attached.
func WithUserID(ctx context.Context, userID uuid.UUID) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

// GetUserIDFromContext retrieves the user_id from the context if present.
func GetUserIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	val := ctx.Value(userIDKey)
	if val == nil {
		return uuid.Nil, false
	}
	id, ok := val.(uuid.UUID)
	return id, ok
}
