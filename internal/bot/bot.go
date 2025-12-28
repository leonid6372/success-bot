package bot

import (
	"context"
	"fmt"

	"github.com/leonid6372/success-bot/internal/common/config"
	"github.com/leonid6372/success-bot/internal/common/domain"
	"github.com/leonid6372/success-bot/pkg/dictionary"
	"github.com/tucnak/telebot"
)

type Bot struct {
	Telebot *telebot.Bot

	deps *Dependencies
}

type Dependencies struct {
	dictionary *dictionary.Dictionary

	userRepository domain.UsersRepository
}

func New(ctx context.Context,
	cfg *config.Bot,
	dictionary *dictionary.Dictionary,
	userRepository domain.UsersRepository,
) (*Bot, error) {
	bot, err := telebot.NewBot(telebot.Settings{
		Token: cfg.APIKey,
	})
	if err != nil {
		return nil, fmt.Errorf("telebot.NewBot: %w", err)
	}

	return &Bot{
		Telebot: bot,
		deps: &Dependencies{
			dictionary:     dictionary,
			userRepository: userRepository,
		},
	}, nil
}

func (b *Bot) Start() {
	b.Telebot.Start()
}

func (b *Bot) Stop() {
	b.Telebot.Stop()
}
