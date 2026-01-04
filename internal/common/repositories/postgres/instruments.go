package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/leonid6372/success-bot/internal/common/domain"
	"github.com/leonid6372/success-bot/pkg/errs"
)

type instrumentsRepository struct {
	psql *pgxpool.Pool
}

func NewInstrumentsRepository(pool *pgxpool.Pool) domain.InstrumentsRepository {
	return &instrumentsRepository{
		psql: pool,
	}
}

func (ir *instrumentsRepository) GetInstrumentByTicker(ctx context.Context, ticker string) (*domain.Instrument, error) {
	query := `SELECT
			id,
			ticker,
			name
		FROM success_bot.instruments
		WHERE ticker = $1`
	instrument := &Instrument{}
	if err := ir.psql.QueryRow(ctx, query, ticker).Scan(
		&instrument.ID,
		&instrument.Ticker,
		&instrument.Name,
	); err != nil {
		return nil, errs.NewStack(err)
	}

	return instrument.CreateDomain(), nil
}

func (ir *instrumentsRepository) GetInstrumentsPagesCount(ctx context.Context) (int64, error) {
	query := `SELECT COUNT(*) FROM success_bot.instruments`
	var instrumentsCount int64
	if err := ir.psql.QueryRow(ctx, query).Scan(&instrumentsCount); err != nil {
		return 0, errs.NewStack(err)
	}

	pagesCount := (instrumentsCount + domain.ReviewInstrumentsPerPage - 1) / domain.ReviewInstrumentsPerPage

	return pagesCount, nil
}

func (ir *instrumentsRepository) GetInstrumentsByPage(ctx context.Context, page int64) ([]*domain.Instrument, error) {
	query := `SELECT
			id,
			ticker,
			name
		FROM success_bot.instruments
		ORDER BY name ASC
		LIMIT $1 OFFSET $2`
	rows, err := ir.psql.Query(ctx, query, domain.ReviewInstrumentsPerPage, (page-1)*domain.ReviewInstrumentsPerPage)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return []*domain.Instrument{}, nil
		}

		return nil, errs.NewStack(err)
	}
	defer rows.Close()

	instruments := []*domain.Instrument{}
	for rows.Next() {
		instrument := &Instrument{}
		if err := rows.Scan(
			&instrument.ID,
			&instrument.Ticker,
			&instrument.Name,
		); err != nil {
			return nil, errs.NewStack(err)
		}
		instruments = append(instruments, instrument.CreateDomain())
	}

	return instruments, nil
}
