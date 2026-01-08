package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/leonid6372/success-bot/internal/common/domain"
	"github.com/leonid6372/success-bot/pkg/errs"
)

type operationsRepository struct {
	psql *pgxpool.Pool
}

func NewOperationsRepository(pool *pgxpool.Pool) domain.OperationsRepository {
	return &operationsRepository{
		psql: pool,
	}
}

func (or *operationsRepository) GetOperationsPagesCount(ctx context.Context, userID int64) (int64, error) {
	query := `SELECT COUNT(*) FROM success_bot.operations WHERE user_id = $1`
	var operationsCount int64
	if err := or.psql.QueryRow(ctx, query, userID).Scan(&operationsCount); err != nil {
		return 0, errs.NewStack(err)
	}

	pagesCount := (operationsCount + domain.OperationsPerPage - 1) / domain.OperationsPerPage
	return pagesCount, nil
}

func (or *operationsRepository) GetOperationsByPage(ctx context.Context, userID, page int64) ([]*domain.Operation, error) {
	query := `SELECT o.id,
			o.parent_id,
			o.type,
			CASE
				WHEN o.type = 'promocode' OR o.type = 'daily_reward' THEN p.value
			ELSE i.name END as name,
			o.count,
			o.total_amount,
			o.created_at
		FROM success_bot.operations o
		LEFT JOIN success_bot.instruments i
			ON o.instrument_id = i.id
		LEFT JOIN success_bot.promocodes p
			ON o.instrument_id = p.id
		WHERE o.user_id = $1
		ORDER BY o.created_at DESC
		LIMIT $2 OFFSET $3`
	rows, err := or.psql.Query(ctx, query, userID, domain.OperationsPerPage, (page-1)*domain.OperationsPerPage)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return []*domain.Operation{}, nil
		}

		return nil, errs.NewStack(err)
	}
	defer rows.Close()

	operations := []*domain.Operation{}
	for rows.Next() {
		operation := &HistoryOperation{}
		if err := rows.Scan(
			&operation.ID,
			&operation.ParentID,
			&operation.Type,
			&operation.InstrumentName,
			&operation.Count,
			&operation.TotalAmount,
			&operation.CreatedAt,
		); err != nil {
			return nil, errs.NewStack(err)
		}
		operations = append(operations, operation.CreateDomain())
	}

	return operations, nil
}
