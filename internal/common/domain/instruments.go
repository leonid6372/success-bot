package domain

import "context"

type InstrumentsRepository interface {
	GetInstrumentByTicker(ctx context.Context, ticker string) (*Instrument, error)
	GetInstrumentsPagesCount(ctx context.Context) (int64, error)
	GetInstrumentsByPage(ctx context.Context, page int64) ([]*Instrument, error)
}

type InstrumentIdentifiers struct {
	ID     int64  `json:"id"`
	Ticker string `json:"ticker"`
	Name   string `json:"name"`
}

type InstrumentPrices struct {
	Last   float64 `json:"last"`
	Bid    float64 `json:"bid"`
	Ask    float64 `json:"ask"`
	Change float64 `json:"change"`
}

type Instrument struct {
	InstrumentIdentifiers
	InstrumentPrices

	Decimals int32 `json:"decimals"`
}
