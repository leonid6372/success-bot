package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/leonid6372/success-bot/internal/common/domain"
)

const (
	instrumentsListPageCount = 10
)

type instrumentsRepository struct {
	psql *pgxpool.Pool
}

func NewInstrumentsRepository(pool *pgxpool.Pool) domain.InstrumentsRepository {
	return &instrumentsRepository{
		psql: pool,
	}
}

func (ir *instrumentsRepository) GetInstrumentsCount(ctx context.Context) (int64, error) {
	query := `SELECT COUNT(*) FROM success_bot.instruments`
	var instrumentsCount int64
	if err := ir.psql.QueryRow(ctx, query).Scan(&instrumentsCount); err != nil {
		return 0, err
	}

	pagesCount := (instrumentsCount + instrumentsListPageCount - 1) / instrumentsListPageCount

	return pagesCount, nil
}

func (ir *instrumentsRepository) GetInstruments(ctx context.Context) ([]*domain.Instrument, error) {
	query := `SELECT
			id,
			ticker,
			name
		FROM success_bot.instruments`
	rows, err := ir.psql.Query(ctx, query)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return []*domain.Instrument{}, nil
		}

		return nil, err
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
			return nil, err
		}
		instruments = append(instruments, instrument.CreateDomain())
	}

	return instruments, nil
}

func (ir *instrumentsRepository) GetInstrumentsByPage(ctx context.Context, page int64) ([]*domain.Instrument, error) {
	query := `SELECT
			id,
			ticker,
			name
		FROM success_bot.instruments
		LIMIT $1 OFFSET $2`
	rows, err := ir.psql.Query(ctx, query, instrumentsListPageCount, (page-1)*instrumentsListPageCount)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return []*domain.Instrument{}, nil
		}

		return nil, err
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
			return nil, err
		}
		instruments = append(instruments, instrument.CreateDomain())
	}

	return instruments, nil
}
