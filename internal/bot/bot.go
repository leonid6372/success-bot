package bot

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/Ruvad39/go-finam-rest"
	"github.com/leonid6372/success-bot/internal/common/config"
	"github.com/leonid6372/success-bot/internal/common/domain"
	"github.com/leonid6372/success-bot/pkg/cache"
	"github.com/leonid6372/success-bot/pkg/dictionary"
	"github.com/leonid6372/success-bot/pkg/errs"
	"gopkg.in/telebot.v4"
)

type Bot struct {
	Telebot *telebot.Bot
	cfg     *config.Bot
	cache   *cache.Cache

	deps *Dependencies
}

type Dependencies struct {
	finam      *finam.Client
	dictionary *dictionary.Dictionary

	userRepository domain.UsersRepository
}

func New(ctx context.Context,
	cfg *config.Bot,
	finam *finam.Client,
	dictionary *dictionary.Dictionary,
	userRepository domain.UsersRepository,
) (*Bot, error) {
	b, err := telebot.NewBot(telebot.Settings{
		Token:  cfg.APIKey,
		Poller: &telebot.LongPoller{Timeout: cfg.Timeout},
	})
	if err != nil {
		return nil, fmt.Errorf("telebot.NewBot: %w", err)
	}

	bot := &Bot{
		Telebot: b,
		cfg:     cfg,
		cache:   cache.New(16*time.Minute, 8*time.Minute),
		deps: &Dependencies{
			finam:          finam,
			dictionary:     dictionary,
			userRepository: userRepository,
		},
	}

	if err := bot.setCommands(); err != nil {
		return nil, fmt.Errorf("bot.setCommands: %w", err)
	}

	bot.setupMiddlewares()
	bot.setupMessageRoutes()
	bot.setupCallbackRoutes()

	return bot, nil
}

func (b *Bot) setCommands() error {
	commands := []telebot.Command{
		{Text: "start", Description: "📈 Get started"},
		{Text: "language", Description: "🌎 Choose language"},
	}

	if err := b.Telebot.SetCommands(commands); err != nil {
		return errs.NewStack(err)
	}

	return nil
}

func (b *Bot) setupMiddlewares() {
	b.Telebot.Use(
		b.recoveryMiddleware,
		b.defaultErrorMiddleware,
		b.timeoutMiddleware,
		b.updateUserInfoMiddleware,
		b.selectUserMiddleware,
		b.subscribeMiddleware,
	)
}

func (b *Bot) setupMessageRoutes() {
	message := b.Telebot.Group()

	message.Handle("/start", b.startHandler)
	message.Handle("/language", b.selectLanguageHandler)
}

func (b *Bot) setupCallbackRoutes() {
	callback := b.Telebot.Group()

	callback.Handle(&telebot.Btn{Unique: cbkLanguage}, b.setLanguageHandler)
	callback.Handle(&telebot.Btn{Unique: cbkCheckSubscription}, b.checkSubscriptionHandler)
}

func (b *Bot) Start() {
	b.Telebot.Start()
}

func (b *Bot) Stop() {
	b.Telebot.Stop()
}

// func (b *Bot) GetUser(c telebot.Context) *domain.User {
// 	if user, ok := c.Get(ctxUser).(*domain.User); ok {
// 		return user
// 	}
// 	return nil
// }

func (b *Bot) mustUser(c telebot.Context) *domain.User {
	tgID := c.Sender().ID

	user, ok := b.cache.Get(tgID)
	if !ok {
		log.Fatal("user not found in cache")
	}

	// usr := b.GetUser(c)
	// if usr == nil {
	// 	log.Fatal("user not found in context")
	// }

	return user.(*domain.User)
}
