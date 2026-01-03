package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/leonid6372/success-bot/internal/boterrs"
	"github.com/leonid6372/success-bot/internal/common/domain"
	"github.com/leonid6372/success-bot/pkg/errs"
	"github.com/leonid6372/success-bot/pkg/log"
	"go.uber.org/zap"
)

type promocodesRepository struct {
	psql *pgxpool.Pool
}

func NewPromocodesRepository(pool *pgxpool.Pool) domain.PromocodesRepository {
	return &promocodesRepository{
		psql: pool,
	}
}

func (pr *promocodesRepository) ApplyPromocode(ctx context.Context, value string, userID int64) (*domain.Promocode, error) {
	tx, err := pr.psql.Begin(ctx)
	if err != nil {
		return nil, errs.NewStack(err)
	}
	defer func() {
		if err := tx.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			log.Error("failed to rollback transaction", zap.Error(err))
		}
	}()

	query := `SELECT * FROM success_bot.promocodes WHERE value = $1 FOR UPDATE`
	promocode := &Promocode{}
	if err := tx.QueryRow(ctx, query, value).Scan(
		&promocode.ID,
		&promocode.AvailableCount,
		&promocode.Value,
		&promocode.BonusAmount,
		&promocode.CreatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, boterrs.ErrInvalidPromocode
		}

		return nil, errs.NewStack(err)
	}

	if promocode.AvailableCount <= 0 {
		return nil, boterrs.ErrInvalidPromocode
	}

	query = `SELECT COUNT(*) FROM success_bot.operations
		WHERE user_id = $1 AND instrument_id = $2 AND type = 'promocode'`
	var usedCount int64
	if err := tx.QueryRow(ctx, query, userID, promocode.ID).Scan(&usedCount); err != nil {
		return nil, errs.NewStack(err)
	}

	if usedCount > 0 {
		return nil, boterrs.ErrUsedPromocode
	}

	query = `UPDATE success_bot.promocodes SET available_count = available_count - 1 WHERE id = $1`
	if _, err = tx.Exec(ctx, query, promocode.ID); err != nil {
		return nil, errs.NewStack(err)
	}

	query = `UPDATE success_bot.users SET balance = balance + $1, updated_at = NOW() WHERE id = $2`
	if _, err = tx.Exec(ctx, query, promocode.BonusAmount, userID); err != nil {
		return nil, errs.NewStack(err)
	}

	query = `INSERT INTO success_bot.operations(user_id, instrument_id, type, count, price, total_amount)
		VALUES ($1, $2, 'promocode', 1, $3, $3)`
	if _, err = tx.Exec(ctx, query, userID, promocode.ID, promocode.BonusAmount); err != nil {
		return nil, errs.NewStack(err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, errs.NewStack(err)
	}

	return promocode.CreateDomain(), nil
}
