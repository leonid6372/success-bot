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

func (ur *usersRepository) GetTopUsersData(ctx context.Context) (int64, []*domain.TopUserData, error) {
	tx, err := ur.psql.Begin(ctx)
	if err != nil {
		return 0, nil, errs.NewStack(err)
	}

	query := `SELECT count(*) FROM success_bot.users;`
	var usersCount int64
	if err := ur.psql.QueryRow(ctx, query).Scan(&usersCount); err != nil {
		return 0, nil, err
	}

	if usersCount == 0 {
		return 0, []*domain.TopUserData{}, nil
	}

	query = `SELECT
			u.username,
			u.balance,
			i.ticker,
			p.count 
		FROM success_bot.users u
		LEFT JOIN success_bot.portfolios p
			ON u.id = p.user_id
		LEFT JOIN success_bot.instruments i
			ON p.instrument_id = i.id;`
	rows, err := tx.Query(ctx, query)
	if err != nil {
		return 0, nil, err
	}
	defer rows.Close()

	topUsersData := []*domain.TopUserData{}
	for rows.Next() {
		data := &TopUserData{}
		if err := rows.Scan(
			&data.Username,
			&data.Balance,
			&data.Ticker,
			&data.Count,
		); err != nil {
			return 0, nil, err
		}

		topUsersData = append(topUsersData, data.CreateDomain())
	}

	return usersCount, topUsersData, nil
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
