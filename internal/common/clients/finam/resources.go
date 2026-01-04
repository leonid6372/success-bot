package finam

import (
	"github.com/Ruvad39/go-finam-rest"
	"github.com/leonid6372/success-bot/internal/common/domain"
)

type getInstrumentResponse struct {
	finam.QuoteResponse
}

func (res *getInstrumentResponse) CreateDomain() *domain.Instrument {
	return &domain.Instrument{
		InstrumentIdentifiers: domain.InstrumentIdentifiers{
			Ticker: res.Quote.Symbol,
		},
		InstrumentPrices: domain.InstrumentPrices{
			Last:   res.Quote.Last.Float64(),
			Bid:    res.Quote.Bid.Float64(),
			Ask:    res.Quote.Ask.Float64(),
			Change: res.Quote.Change.Float64(),
		},
	}
}
