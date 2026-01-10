package finam

import (
	"context"
	"errors"

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
	res := &getInstrumentQuoteResponse{}

	res.QuoteResponse, err = c.Client.NewQuoteRequest(ticker).Do(ctx)
	if err != nil {
		return nil, errs.NewStack(err)
	}

	return res.CreateDomain(), nil
}

// GetInstrumentInfo return domain.Instrument struct with actual Decimals count value.
func (c *Client) GetInstrumentInfo(ctx context.Context, ticker string) (*domain.Instrument, error) {
	var err error
	res := &getInstrumentInfoResponse{}

	res.AssetInfo, err = c.Client.NewAssetInfoRequest(ticker, c.accountID).Do(ctx)
	if err != nil {
		if errors.Is(err, finam.ErrNotFound) {
			return nil, finam.ErrNotFound
		}

		return nil, errs.NewStack(err)
	}

	return res.CreateDomain(), nil
}
