package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/leonid6372/success-bot/internal/common/domain"
	"github.com/leonid6372/success-bot/pkg/errs"
)

type portfolioRepository struct {
	psql *pgxpool.Pool
}

func NewPortfolioRepository(pool *pgxpool.Pool) domain.PortfolioRepository {
	return &portfolioRepository{
		psql: pool,
	}
}

func (pr *portfolioRepository) GetUsersInstrumentsCount(ctx context.Context) (int64, error) {
	query := `SELECT COUNT(DISTINCT instrument_id) FROM success_bot.users_instruments`
	var instrumentsCount int64
	if err := pr.psql.QueryRow(ctx, query).Scan(&instrumentsCount); err != nil {
		return 0, errs.NewStack(err)
	}

	return instrumentsCount, nil
}

func (pr *portfolioRepository) GetUserPortfolioPagesCount(ctx context.Context, userID int64) (int64, error) {
	query := `SELECT COUNT(*) FROM success_bot.users_instruments WHERE user_id = $1`
	var instrumentsCount int64
	if err := pr.psql.QueryRow(ctx, query, userID).Scan(&instrumentsCount); err != nil {
		return 0, errs.NewStack(err)
	}

	pagesCount := (instrumentsCount + domain.PortfolioInstrumentsPerPage - 1) / domain.PortfolioInstrumentsPerPage

	return pagesCount, nil
}

func (pr *portfolioRepository) GetUserPortfolioByPage(ctx context.Context, userID, page int64) ([]*domain.UserInstrument, error) {
	query := `SELECT
			ui.user_id,
			i.ticker,
			i.name,
			ui.count,
			ui.average_price,
			ui.created_at,
			ui.updated_at
		FROM success_bot.users_instruments ui
		JOIN success_bot.instruments i
			ON ui.instrument_id = i.id
		WHERE ui.user_id = $1
		ORDER BY i.name ASC
		LIMIT $2 OFFSET $3`
	rows, err := pr.psql.Query(ctx, query, userID, domain.PortfolioInstrumentsPerPage, (page-1)*domain.PortfolioInstrumentsPerPage)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return []*domain.UserInstrument{}, nil
		}

		return nil, errs.NewStack(err)
	}
	defer rows.Close()

	userInstruments := []*domain.UserInstrument{}
	for rows.Next() {
		userInstrument := &UserInstrument{}
		if err := rows.Scan(
			&userInstrument.UserID,
			&userInstrument.InstrumentTicker,
			&userInstrument.InstrumentName,
			&userInstrument.Count,
			&userInstrument.AvgPrice,
			&userInstrument.CreatedAt,
			&userInstrument.UpdatedAt,
		); err != nil {
			return nil, errs.NewStack(err)
		}

		userInstruments = append(userInstruments, userInstrument.CreateDomain())
	}

	return userInstruments, nil
}
