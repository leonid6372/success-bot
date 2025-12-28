package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/leonid6372/success-bot/internal/common/domain"
)

type usersRepository struct {
	pool *pgxpool.Pool
}

func NewUsersRepository(pool *pgxpool.Pool) domain.UsersRepository {
	return &usersRepository{
		pool: pool,
	}
}

func (ur *usersRepository) CreateUser(ctx context.Context, user *domain.User) error {
	return nil
}
