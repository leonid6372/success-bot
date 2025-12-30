package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/leonid6372/success-bot/internal/common/domain"
	"github.com/leonid6372/success-bot/pkg/errs"
)

type usersRepository struct {
	psql *pgxpool.Pool
}

func NewUsersRepository(pool *pgxpool.Pool) domain.UsersRepository {
	return &usersRepository{
		psql: pool,
	}
}

func (ur *usersRepository) CreateUser(ctx context.Context, user *domain.User) error {
	query := `INSERT INTO success_bot.users(
			id,
			username,
			first_name,
			last_name,
			is_premium
		)
		VALUES ($1, $2, $3, $4, $5)`
	_, err := ur.psql.Exec(ctx,
		query,
		user.ID,
		user.Username,
		user.FirstName,
		user.LastName,
		user.IsPremium,
	)
	if err != nil {
		return errs.NewStack(err)
	}

	return nil
}

func (ur *usersRepository) GetUserByID(ctx context.Context, id int64) (*domain.User, error) {
	query := `SELECT
			id,
    		username,
    		first_name,
    		last_name,
    		language_code,
    		is_premium,
    		balance,
    		created_at,
    		updated_at
		FROM success_bot.users WHERE id = $1`
	user := &User{}
	if err := ur.psql.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.Username,
		&user.FirstName,
		&user.LastName,
		&user.LanguageCode,
		&user.IsPremium,
		&user.Balance,
		&user.CreatedAt,
		&user.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}

		return nil, err
	}

	return user.CreateDomain(), nil
}

// UpdateUserTGData updates username, first name, last name and is_premium fields of the user.
func (ur *usersRepository) UpdateUserTGData(ctx context.Context, user *domain.User) error {
	query := `UPDATE success_bot.users
		SET username = $1,
			first_name = $2,
			last_name = $3,
			is_premium = $4
		WHERE id = $5`
	_, err := ur.psql.Exec(ctx, query, user.Username, user.FirstName, user.LastName, user.IsPremium, user.ID)
	if err != nil {
		return errs.NewStack(err)
	}

	return nil
}

func (ur *usersRepository) UpdateUserLanguage(ctx context.Context, userID int64, languageCode string) error {
	query := `UPDATE success_bot.users
		SET language_code = $1
		WHERE id = $2`
	_, err := ur.psql.Exec(ctx, query, languageCode, userID)
	if err != nil {
		return errs.NewStack(err)
	}

	return nil
}
