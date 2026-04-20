package repository

import (
	"context"
	"errors"
	"sync"
	"time"
)

type sessionItem struct {
	userID    string
	expiresAt time.Time
}

type memorySessionRepository struct {
	sessions sync.Map // map[string]sessionItem
}

func NewMemorySessionRepository() SessionRepository {
	return &memorySessionRepository{}
}

func (r *memorySessionRepository) Store(ctx context.Context, token, userID string, ttl time.Duration) error {
	r.sessions.Store(token, sessionItem{
		userID:    userID,
		expiresAt: time.Now().Add(ttl),
	})
	return nil
}

func (r *memorySessionRepository) Get(ctx context.Context, token string) (string, error) {
	val, ok := r.sessions.Load(token)
	if !ok {
		return "", errors.New("session not found")
	}

	item := val.(sessionItem)
	if time.Now().After(item.expiresAt) {
		r.sessions.Delete(token)
		return "", errors.New("session expired")
	}

	return item.userID, nil
}

func (r *memorySessionRepository) Delete(ctx context.Context, token string) error {
	r.sessions.Delete(token)
	return nil
}
