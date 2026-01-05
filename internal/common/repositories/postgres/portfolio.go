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

func (pr *portfolioRepository) GetUserMostExpensiveShort(ctx context.Context, userID int64) (*domain.UserInstrument, error) {
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
		WHERE ui.user_id = $1 AND ui.count < 0
		ORDER BY ui.average_price * ABS(ui.count) desc
		LIMIT 1;`
	userInstrument := &UserInstrument{}
	if err := pr.psql.QueryRow(ctx, query, userID).Scan(
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

	return userInstrument.CreateDomain(), nil
}

func (pr *portfolioRepository) SellInstrument(ctx context.Context, userID, instrumentID, countToSell int64, price float64) error {
	tx, err := pr.psql.Begin(ctx)
	if err != nil {
		return errs.NewStack(err)
	}
	defer func() {
		if err := tx.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			log.Error("failed to rollback transaction", zap.Error(err))
		}
	}()

	var currentCount int64
	var avgPrice float64
	query := `SELECT count, average_price
		FROM success_bot.users_instruments
		WHERE user_id = $1 AND instrument_id = $2 FOR UPDATE`
	err = tx.QueryRow(ctx, query, userID, instrumentID).Scan(&currentCount, &avgPrice)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return errs.NewStack(err)
	}

	// close long
	if currentCount > 0 {
		count := min(countToSell, currentCount)

		// balanceDiff := buyAmount + 3% fee for selling
		balanceDiff := float64(count)*(price-avgPrice) - float64(count)*price*0.003

		query = `UPDATE success_bot.users
			SET available_balance = available_balance + $1`
		if _, err = tx.Exec(ctx, query, balanceDiff); err != nil {
			return errs.NewStack(err)
		}

		if count == currentCount { // close whole long
			query = `DELETE FROM success_bot.users_instruments
			WHERE user_id = $1 AND instrument_id = $2`
			if _, err = tx.Exec(ctx, query, userID, instrumentID); err != nil {
				return errs.NewStack(err)
			}
		} else { // close part of long
			query = `UPDATE success_bot.users_instruments
			SET count = count - $1
			WHERE user_id = $2 AND instrument_id = $3`
			if _, err = tx.Exec(ctx, query, count, userID, instrumentID); err != nil {
				return errs.NewStack(err)
			}

			return nil
		}

		countToSell = countToSell - currentCount
		currentCount = 0
		avgPrice = 0
	}

	// make sell
	if countToSell > 0 {
		fee := float64(countToSell) * price * 0.003         // 3% fee for selling
		amountToBlock := float64(countToSell) * price * 0.5 // 50% guarantee coverage

		needBalance := amountToBlock + fee

		var actualBalance float64
		query = `SELECT available_balance FROM success_bot.users WHERE id = $1 FOR UPDATE`
		if err = tx.QueryRow(ctx, query, userID).Scan(&actualBalance); err != nil {
			return errs.NewStack(err)
		}

		if actualBalance < needBalance {
			return boterrs.ErrInsufficientFunds
		}

		query = `UPDATE success_bot.users
			SET available_balance = available_balance - $1, blocked_balance = blocked_balance + $2`
		if _, err = tx.Exec(ctx, query, needBalance, amountToBlock); err != nil {
			return errs.NewStack(err)
		}

		newCount := currentCount - countToSell
		newAvgPrice := (avgPrice*float64(-currentCount) + price*float64(countToSell)) / float64(-newCount)

		query = `INSERT INTO success_bot.users_instruments(user_id, instrument_id, count, average_price)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (user_id, instrument_id) DO UPDATE
			SET count = $3, average_price = $4`
		if _, err = tx.Exec(ctx, query, userID, instrumentID, newCount, newAvgPrice); err != nil {
			return errs.NewStack(err)
		}
	}

	var opID int64
	query = `INSERT INTO success_bot.operations(user_id, instrument_id, type, count, price, total_amount)
		VALUES ($1, $2, 'sell', $3, $4, $5) RETURNING id`
	if err = tx.QueryRow(ctx, query, userID, instrumentID, countToSell, price, float64(countToSell)*price).
		Scan(&opID); err != nil {
		return errs.NewStack(err)
	}

	query = `INSERT INTO success_bot.operations(parent_id, user_id, instrument_id, type, count, price, total_amount)
		VALUES ($1, $2, $3, 'fee', 1, $4, $4)`
	if _, err = tx.Exec(ctx, query, opID, userID, instrumentID, float64(countToSell)*price*0.003); err != nil {
		return errs.NewStack(err)
	}

	if err := tx.Commit(ctx); err != nil {
		return errs.NewStack(err)
	}

	return nil
}

func (pr *portfolioRepository) BuyInstrument(ctx context.Context, userID, instrumentID, countToBuy int64, price float64) error {
	tx, err := pr.psql.Begin(ctx)
	if err != nil {
		return errs.NewStack(err)
	}
	defer func() {
		if err := tx.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			log.Error("failed to rollback transaction", zap.Error(err))
		}
	}()

	var currentCount int64
	var avgPrice float64
	query := `SELECT count, average_price
		FROM success_bot.users_instruments
		WHERE user_id = $1 AND instrument_id = $2 FOR UPDATE`
	err = tx.QueryRow(ctx, query, userID, instrumentID).Scan(&currentCount, &avgPrice)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return errs.NewStack(err)
	}

	// close short
	if currentCount < 0 {
		count := min(countToBuy, -currentCount)

		// balanceDiff := shortResult - 3% fee for buying
		balanceDiff := float64(count)*(price-avgPrice) - float64(count)*price*0.003

		query = `UPDATE success_bot.users
			SET available_balance = available_balance + $1`
		if _, err = tx.Exec(ctx, query, balanceDiff); err != nil {
			return errs.NewStack(err)
		}

		if count == -currentCount { // close whole short
			query = `DELETE FROM success_bot.users_instruments
			WHERE user_id = $1 AND instrument_id = $2`
			if _, err = tx.Exec(ctx, query, userID, instrumentID); err != nil {
				return errs.NewStack(err)
			}
		} else { // close part of short
			query = `UPDATE success_bot.users_instruments
			SET count = count + $1
			WHERE user_id = $2 AND instrument_id = $3`
			if _, err = tx.Exec(ctx, query, count, userID, instrumentID); err != nil {
				return errs.NewStack(err)
			}

			return nil
		}

		countToBuy = countToBuy + currentCount
		currentCount = 0
		avgPrice = 0
	}

	// make buy
	if countToBuy > 0 {
		// balanceDiff := buyAmount + 3% fee for buying
		balanceDiff := float64(countToBuy) * price * 1.003

		var actualBalance float64
		query = `SELECT available_balance FROM success_bot.users WHERE id = $1 FOR UPDATE`
		if err = tx.QueryRow(ctx, query, userID).Scan(&actualBalance); err != nil {
			return errs.NewStack(err)
		}

		if actualBalance < balanceDiff {
			return boterrs.ErrInsufficientFunds
		}

		query = `UPDATE success_bot.users
			SET available_balance = available_balance - $1`
		if _, err = tx.Exec(ctx, query, balanceDiff); err != nil {
			return errs.NewStack(err)
		}

		newCount := currentCount + countToBuy
		newAvgPrice := (avgPrice*float64(currentCount) + price*float64(countToBuy)) / float64(newCount)

		query = `INSERT INTO success_bot.users_instruments(user_id, instrument_id, count, average_price)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (user_id, instrument_id) DO UPDATE
			SET count = $3, average_price = $4`
		if _, err = tx.Exec(ctx, query, userID, instrumentID, newCount, newAvgPrice); err != nil {
			return errs.NewStack(err)
		}
	}

	var opID int64
	query = `INSERT INTO success_bot.operations(user_id, instrument_id, type, count, price, total_amount)
		VALUES ($1, $2, 'buy', $3, $4, $5) RETURNING id`
	if err = tx.QueryRow(ctx, query, userID, instrumentID, countToBuy, price, float64(countToBuy)*price).
		Scan(&opID); err != nil {
		return errs.NewStack(err)
	}

	query = `INSERT INTO success_bot.operations(parent_id, user_id, instrument_id, type, count, price, total_amount)
		VALUES ($1, $2, $3, 'fee', 1, $4, $4)`
	if _, err = tx.Exec(ctx, query, opID, userID, instrumentID, float64(countToBuy)*price*0.003); err != nil {
		return errs.NewStack(err)
	}

	if err := tx.Commit(ctx); err != nil {
		return errs.NewStack(err)
	}

	return nil
}
