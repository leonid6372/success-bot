package domain

import (
	"context"
	"time"
)

type OperationsRepository interface {
	GetOperationsPagesCount(ctx context.Context) (int64, error)
	GetOperationsByPage(ctx context.Context, userID, page int64) ([]*Operation, error)
}

type Operation struct {
	ID       int64 `json:"id"`
	ParentID int64 `json:"parent_id"`

	Type           string  `json:"type"`
	InstrumentName string  `json:"instrument_name"`
	Count          int64   `json:"count"`
	TotalAmount    float64 `json:"total_amount"`

	CreatedAt time.Time `json:"created_at"`
}
