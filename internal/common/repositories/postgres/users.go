package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/leonid6372/success-bot/internal/boterrs"
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
    		available_balance,
			blocked_balance,
			margin_call,
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
		&user.AvailableBalance,
		&user.BlockedBalance,
		&user.MarginCall,
		&user.CreatedAt,
		&user.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, boterrs.ErrUserNotFound
		}

		return nil, errs.NewStack(err)
	}

	return user.CreateDomain(), nil
}

func (ur *usersRepository) GetUsersCount(ctx context.Context) (int64, error) {
	query := `SELECT count(*) FROM success_bot.users`
	var usersCount int64
	if err := ur.psql.QueryRow(ctx, query).Scan(&usersCount); err != nil {
		return 0, errs.NewStack(err)
	}

	return usersCount, nil
}

func (ur *usersRepository) GetTopUsersData(ctx context.Context) ([]*domain.TopUserData, error) {
	query := `SELECT
			u.id,
			u.username,
			u.language_code,
			u.available_balance,
			u.blocked_balance,
			u.margin_call,
			i.ticker,
			ui.count 
		FROM success_bot.users u
		LEFT JOIN success_bot.users_instruments ui
			ON u.id = ui.user_id
		LEFT JOIN success_bot.instruments i
			ON ui.instrument_id = i.id`
	rows, err := ur.psql.Query(ctx, query)
	if err != nil {
		return nil, errs.NewStack(err)
	}
	defer rows.Close()

	topUsersData := []*domain.TopUserData{}
	for rows.Next() {
		data := &TopUserData{}
		if err := rows.Scan(
			&data.ID,
			&data.Username,
			&data.LanguageCode,
			&data.AvailableBalance,
			&data.BlockedBalance,
			&data.MarginCall,
			&data.Ticker,
			&data.Count,
		); err != nil {
			return nil, errs.NewStack(err)
		}

		topUsersData = append(topUsersData, data.CreateDomain())
	}

	return topUsersData, nil
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

// UpdateUserBalancesAndMarginCall updates available_balance and margin_call by gotten values.
// Set blocked_balance = blocked_balance - blockedBalanceDelta. Nil values will be ignored to update.
func (ur *usersRepository) UpdateUserBalancesAndMarginCall(
	ctx context.Context, userID int64, availableBalance float64, blockedBalanceDelta *float64, marginCall *bool) error {
	args := make([]any, 0, 4)

	args = append(args, availableBalance)
	query := `UPDATE success_bot.users
		SET available_balance = $1`

	if blockedBalanceDelta != nil {
		args = append(args, *blockedBalanceDelta)
		query += fmt.Sprintf(`, blocked_balance = blocked_balance - $%d`, len(args))
	}

	if marginCall != nil {
		args = append(args, *marginCall)
		query += fmt.Sprintf(`, margin_call = $%d`, len(args))
	}

	args = append(args, userID)
	query += fmt.Sprintf(` WHERE id = $%d`, len(args))

	_, err := ur.psql.Exec(ctx, query, args...)
	if err != nil {
		return errs.NewStack(err)
	}

	return nil
}
