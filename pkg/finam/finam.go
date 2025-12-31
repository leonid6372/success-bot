package finam

import (
	"context"

	"github.com/Ruvad39/go-finam-rest"
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

func (c *Client) NewAssetInfoRequest(ticker string) *finam.AssetInfoRequest {
	return c.Client.NewAssetInfoRequest(ticker, c.accountID)
}
