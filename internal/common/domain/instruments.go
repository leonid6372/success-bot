package domain

import "context"

type InstrumentsRepository interface {
	GetInstrumentsPagesCount(ctx context.Context) (int64, error)
	GetInstruments(ctx context.Context) ([]*Instrument, error)
	GetInstrumentsByPage(ctx context.Context, page int64) ([]*Instrument, error)
}

type Instrument struct {
	ID     int64  `json:"id"`
	Ticker string `json:"ticker"`
	Name   string `json:"name"`
}
