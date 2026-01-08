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

	AvailableBalance float64 `db:"available_balance"`
	BlockedBalance   float64 `db:"blocked_balance"`
	MarginCall       bool    `db:"margin_call"`

	DailyReward bool `db:"daily_reward"`

	UpdatedAt time.Time `db:"updated_at"`
	CreatedAt time.Time `db:"created_at"`
}

func (u *User) CreateDomain() *domain.User {
	user := &domain.User{
		ID:               u.ID,
		Username:         u.Username,
		FirstName:        u.FirstName,
		LastName:         u.LastName,
		LanguageCode:     u.LanguageCode,
		IsPremium:        u.IsPremium,
		AvailableBalance: u.AvailableBalance,
		BlockedBalance:   u.BlockedBalance,
		MarginCall:       u.MarginCall,
		DailyReward:      u.DailyReward,
		CreatedAt:        u.CreatedAt,
		UpdatedAt:        u.UpdatedAt,
	}

	return user
}

type TopUserData struct {
	ID               int64   `db:"id"`
	Username         string  `db:"username"`
	LanguageCode     string  `db:"language_code"`
	AvailableBalance float64 `db:"available_balance"`
	BlockedBalance   float64 `db:"blocked_balance"`
	MarginCall       bool    `db:"margin_call"`
	Ticker           *string `db:"ticker"`
	Count            *int64  `db:"count"`
}

func (d *TopUserData) CreateDomain() *domain.TopUserData {
	data := &domain.TopUserData{
		TopUser: domain.TopUser{
			ID:               d.ID,
			Username:         d.Username,
			LanguageCode:     d.LanguageCode,
			AvailableBalance: d.AvailableBalance,
			BlockedBalance:   d.BlockedBalance,
			MarginCall:       d.MarginCall,
		},
	}

	if d.Ticker != nil {
		data.Ticker = *d.Ticker
	}
	if d.Count != nil {
		data.Count = *d.Count
	}

	return data
}

type Instrument struct {
	ID     int64  `db:"id"`
	Ticker string `db:"ticker"`
	Name   string `db:"name"`
}

func (i *Instrument) CreateDomain() *domain.Instrument {
	instrument := &domain.Instrument{
		InstrumentIdentifiers: domain.InstrumentIdentifiers{
			ID:     i.ID,
			Ticker: i.Ticker,
			Name:   i.Name,
		},
	}

	return instrument
}

type Promocode struct {
	ID             int64     `db:"id"`
	AvailableCount int64     `db:"available_count"`
	Value          string    `db:"value"`
	BonusAmount    float64   `db:"bonus_amount"`
	CreatedAt      time.Time `db:"created_at"`
}

func (p *Promocode) CreateDomain() *domain.Promocode {
	promocode := &domain.Promocode{
		ID:             p.ID,
		AvailableCount: p.AvailableCount,
		Value:          p.Value,
		BonusAmount:    p.BonusAmount,
		CreatedAt:      p.CreatedAt,
	}

	return promocode
}

type HistoryOperation struct {
	ID             int64     `db:"id"`
	ParentID       *int64    `db:"parent_id"`
	Type           string    `db:"type"`
	InstrumentName string    `db:"instrument_name"`
	Count          int64     `db:"count"`
	TotalAmount    float64   `db:"total_amount"`
	CreatedAt      time.Time `db:"created_at"`
}

func (ho *HistoryOperation) CreateDomain() *domain.Operation {
	operation := &domain.Operation{
		ID:             ho.ID,
		Type:           ho.Type,
		InstrumentName: ho.InstrumentName,
		Count:          ho.Count,
		TotalAmount:    ho.TotalAmount,
		CreatedAt:      ho.CreatedAt,
	}

	if ho.ParentID != nil {
		operation.ParentID = *ho.ParentID
	}

	return operation
}

type UserInstrument struct {
	UserID           int64     `db:"user_id"`
	InstrumentID     int64     `db:"instrument_id"`
	InstrumentTicker string    `db:"instrument_ticker"`
	InstrumentName   string    `db:"instrument_name"`
	Count            int64     `db:"count"`
	AvgPrice         float64   `db:"average_price"`
	CreatedAt        time.Time `db:"created_at"`
	UpdatedAt        time.Time `db:"updated_at"`
}

func (ui *UserInstrument) CreateDomain() *domain.UserInstrument {
	userInstrument := &domain.UserInstrument{
		UserID: ui.UserID,
		InstrumentIdentifiers: domain.InstrumentIdentifiers{
			ID:     ui.InstrumentID,
			Ticker: ui.InstrumentTicker,
			Name:   ui.InstrumentName,
		},
		Count:     ui.Count,
		AvgPrice:  ui.AvgPrice,
		CreatedAt: ui.CreatedAt,
		UpdatedAt: ui.UpdatedAt,
	}

	return userInstrument
}
