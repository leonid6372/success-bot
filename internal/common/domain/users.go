package domain

import (
	"context"
	"time"
)

const (
	InputTypePromocode = "promocode"
	InputTypeTicker    = "ticker"
	InputTypeCount     = "count"
)

type UsersRepository interface {
	CreateUser(ctx context.Context, user *User) error
	GetUserByID(ctx context.Context, id int64) (*User, error)
	GetUsersCount(ctx context.Context) (int64, error)
	GetTopUsersData(ctx context.Context) ([]*TopUserData, error)
	// UpdateUserTGData updates username, first name, last name and is_premium fields of the user.
	UpdateUserTGData(ctx context.Context, user *User) error
	UpdateUserLanguage(ctx context.Context, userID int64, languageCode string) error
	// UpdateUserBalancesAndMarginCall updates available_balance and margin_call by gotten values.
	// Set blocked_balance = blocked_balance - blockedBalanceDelta. Nil values will be ignored to update.
	UpdateUserBalancesAndMarginCall(
		ctx context.Context, userID int64, availableBalance float64, blockedBalanceDelta *float64, marginCall *bool,
	) error
}

type Metadata struct {
	InstrumentDone      *chan struct{}
	InstrumentTicker    string
	InstrumentBuyPrice  float64
	InstrumentSellPrice float64
	InstrumentOperation string

	InputType string
}

type User struct {
	ID int64 `json:"id"`

	Username     string `json:"username"`
	FirstName    string `json:"first_name"`
	LastName     string `json:"last_name"`
	LanguageCode string `json:"language_code"`
	IsPremium    bool   `json:"is_premium"`

	AvailableBalance float64 `json:"available_balance"`
	BlockedBalance   float64 `json:"blocked_balance"`
	MarginCall       bool    `json:"margin_call"`

	Metadata Metadata `json:"metadata"`

	UpdatedAt time.Time `json:"updated_at"`
	CreatedAt time.Time `json:"created_at"`
}

type TopUser struct {
	ID           int64  `json:"id"`
	Username     string `json:"username"`
	LanguageCode string `json:"language_code"`

	AvailableBalance   float64 `json:"available_balance"`
	BlockedBalance     float64 `json:"blocked_balance"`
	BlockedBalanceDiff float64 `json:"blocked_balance_diff"`
	TotalBalance       float64 `json:"total_balance"`
	MarginCall         bool    `json:"margin_call"`
}

type TopUserData struct {
	TopUser

	Ticker string `json:"ticker"`
	Count  int64  `json:"count"`
}
