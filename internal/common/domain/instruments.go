package domain

import "context"

type InstrumentsRepository interface {
	GetInstrumentsPagesCount(ctx context.Context) (int64, error)
	GetInstrumentsByPage(ctx context.Context, page int64) ([]*Instrument, error)
}

type Price struct {
	Last   float64 `json:"last"`
	Bid    float64 `json:"bid"`
	Ask    float64 `json:"ask"`
	Change float64 `json:"change"`
}

type Instrument struct {
	ID     int64  `json:"id"`
	Ticker string `json:"ticker"`
	Name   string `json:"name"`

	Price
}
