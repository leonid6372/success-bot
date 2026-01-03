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
		Ticker: res.Symbol,
		Price: domain.Price{
			Last:   res.Quote.Last.Float64(),
			Bid:    res.Quote.Bid.Float64(),
			Ask:    res.Quote.Ask.Float64(),
			Change: res.Quote.Change.Float64(),
		},
	}
}
