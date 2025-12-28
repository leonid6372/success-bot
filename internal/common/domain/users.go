package domain

import (
	"context"
	"time"
)

type UsersRepository interface {
	CreateUser(ctx context.Context, user *User) error
}

type User struct {
	ID int64 `json:"id"`

	Username     string `db:"username"`
	FirstName    string `db:"first_name"`
	LastName     string `db:"last_name"`
	LanguageCode string `db:"language_code"`
	IsPremium    bool   `db:"is_premium"`

	Balance float64 `db:"balance"`

	UpdatedAt time.Time `db:"updated_at"`
	CreatedAt time.Time `db:"created_at"`
}
