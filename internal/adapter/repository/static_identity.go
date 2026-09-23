package repository

import (
	"context"
	"errors"
	
	"github.com/leomichalski/aigateway/internal/domain"
)

// TODO: move to domain/errors.go
var ErrIdentityNotFound = errors.New("identity not found")

// TODO: delete StaticIdentityRepository, an in-memory placeholder waiting for mongodb implementation
type StaticIdentityRepository struct {
	store map[string]domain.Identity
}

func NewStaticIdentityRepository(store map[string]domain.Identity) *StaticIdentityRepository {
	if store == nil {
		store = make(map[string]domain.Identity)
	}
	return &StaticIdentityRepository{
		store: store,
	}
}

func (r *StaticIdentityRepository) GetIdentityByAPIKey(ctx context.Context, keyHash string) (domain.Identity, error) {
	if identity, exists := r.store[keyHash]; exists {
		return identity, nil
	}
	return domain.Identity{}, ErrIdentityNotFound
}
