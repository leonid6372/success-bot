package boterrs

import "errors"

var (
	ErrUserNotFound           = errors.New("user not found")
	ErrInvalidPromocode       = errors.New("invalid promocode")
	ErrUsedPromocode          = errors.New("used promocode")
	ErrEmptyTickerToBuy       = errors.New("empty ticker to buy")
	ErrEmptyTickerToSell      = errors.New("empty ticker to sell")
	ErrInsufficientFunds      = errors.New("insufficient funds")
	ErrUnavailableDailyReward = errors.New("unavailable daily reward")
)
