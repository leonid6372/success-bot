package domain

import (
	"context"
	"time"
)

type PortfolioRepository interface {
	GetUsersInstrumentsCount(ctx context.Context) (int64, error)
	GetUserPortfolioPagesCount(ctx context.Context, userID int64) (int64, error)
	GetUserPortfolioByPage(ctx context.Context, userID int64, currentPage int64) ([]*UserInstrument, error)
}

type UserInstrument struct {
	UserID int64 `json:"user_id"`

	InstrumentIdentifiers
	InstrumentPrices

	Count    int64   `json:"count"`
	AvgPrice float64 `json:"avg_price"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
