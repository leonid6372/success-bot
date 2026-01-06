package domain

import (
	"context"
	"time"
)

type PortfolioRepository interface {
	GetUsersInstrumentsCount(ctx context.Context) (int64, error)
	GetUserPortfolioPagesCount(ctx context.Context, userID int64) (int64, error)
	GetUserPortfolioByPage(ctx context.Context, userID int64, currentPage int64) ([]*UserInstrument, error)
	GetUserMostExpensiveShort(ctx context.Context, userID int64) (*UserInstrument, error)
	GetMaxInstrumentCountToBuy(ctx context.Context, userID int64, ticker string, price float64) (int64, error)
	BuyInstrument(ctx context.Context, userID, instrumentID, countToBuy int64, price float64) error
	GetMaxInstrumentCountToSell(ctx context.Context, userID int64, ticker string, price float64) (int64, error)
	SellInstrument(ctx context.Context, userID, instrumentID, countToSell int64, price float64) error
}

type UserInstrument struct {
	UserID int64 `json:"user_id"`

	InstrumentIdentifiers
	InstrumentPrices

	Count      int64   `json:"count"`
	AvgPrice   float64 `json:"avg_price"`
	BlockPrice float64 `json:"block_price"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
