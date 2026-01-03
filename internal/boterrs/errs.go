package boterrs

import "errors"

var (
	ErrInvalidPromocode = errors.New("invalid promocode")
	ErrUsedPromocode    = errors.New("used promocode")
)
