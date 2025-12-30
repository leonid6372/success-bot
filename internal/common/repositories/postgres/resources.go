package postgres

import (
	"time"

	"github.com/leonid6372/success-bot/internal/common/domain"
)

type User struct {
	ID int64 `db:"id"`

	Username     string `db:"username"`
	FirstName    string `db:"first_name"`
	LastName     string `db:"last_name"`
	LanguageCode string `db:"language_code"`
	IsPremium    bool   `db:"is_premium"`

	Balance float64 `db:"balance"`

	UpdatedAt time.Time `db:"updated_at"`
	CreatedAt time.Time `db:"created_at"`
}

func (u *User) CreateDomain() *domain.User {
	user := &domain.User{
		ID:           u.ID,
		Username:     u.Username,
		FirstName:    u.FirstName,
		LastName:     u.LastName,
		LanguageCode: u.LanguageCode,
		IsPremium:    u.IsPremium,
		Balance:      u.Balance,
		CreatedAt:    u.CreatedAt,
		UpdatedAt:    u.UpdatedAt,
	}

	return user
}

type Instrument struct {
	ID     int64  `db:"id"`
	Ticker string `db:"ticker"`
	Name   string `db:"name"`
}

func (i *Instrument) CreateDomain() *domain.Instrument {
	instrument := &domain.Instrument{
		ID:     i.ID,
		Ticker: i.Ticker,
		Name:   i.Name,
	}

	return instrument
}
