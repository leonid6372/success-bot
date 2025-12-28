package domain

import (
	"context"
	"time"
)

type UsersRepository interface {
	CreateUser(ctx context.Context, user *User) error
	GetUserByID(ctx context.Context, id int64) (*User, error)
	// UpdateUserTGData updates username, first name, last name and is_premium fields of the user.
	UpdateUserTGData(ctx context.Context, user *User) error
	UpdateUserLanguage(ctx context.Context, userID int64, languageCode string) error
}

type User struct {
	ID int64 `json:"id"`

	Username     string `json:"username"`
	FirstName    string `json:"first_name"`
	LastName     string `json:"last_name"`
	LanguageCode string `json:"language_code"`
	IsPremium    bool   `json:"is_premium"`

	Balance float64 `json:"balance"`

	UpdatedAt time.Time `json:"updated_at"`
	CreatedAt time.Time `json:"created_at"`
}
