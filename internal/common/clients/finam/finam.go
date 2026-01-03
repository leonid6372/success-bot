package finam

import (
	"context"

	"github.com/Ruvad39/go-finam-rest"
	"github.com/leonid6372/success-bot/internal/common/domain"
	"github.com/leonid6372/success-bot/pkg/errs"
)

type Client struct {
	*finam.Client
	accountID string
}

func NewClient(ctx context.Context, token, accountID string) (*Client, error) {
	finam, err := finam.NewClient(ctx, token)
	if err != nil {
		return nil, errs.NewStack(err)
	}

	return &Client{
		Client:    finam,
		accountID: accountID,
	}, nil
}

// GetInstrumentPrices return domain.Instrument struct with actual Price's values by Finam Quote API.
func (c *Client) GetInstrumentPrices(ctx context.Context, ticker string) (*domain.Instrument, error) {
	var err error
	res := &getInstrumentResponse{}

	res.QuoteResponse, err = c.Client.NewQuoteRequest(ticker).Do(ctx)
	if err != nil {
		return nil, errs.NewStack(err)
	}

	return res.CreateDomain(), nil
}
