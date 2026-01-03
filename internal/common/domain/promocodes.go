package domain

import (
	"context"
	"time"
)

type PromocodesRepository interface {
	ApplyPromocode(ctx context.Context, value string, userID int64) (*Promocode, error)
}

type Promocode struct {
	ID             int64     `json:"id"`
	AvailableCount int64     `json:"available_count"`
	Value          string    `json:"value"`
	BonusAmount    float64   `json:"bonus_amount"`
	CreatedAt      time.Time `json:"created_at"`
}
