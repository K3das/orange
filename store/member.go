package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/K3das/orange/store/db"
	"github.com/jackc/pgx/v5"
)

func (s *Store) GetOrCreateUser(ctx context.Context, memberID string) (*db.User, error) {
	err := s.CreateUser(ctx, memberID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("creating member: %w", err)
	}

	member, err := s.GetUsers(ctx, memberID)
	if err != nil {
		return nil, fmt.Errorf("getting member: %w", err)
	}

	return &member, nil
}
