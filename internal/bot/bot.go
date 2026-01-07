package bot

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/leonid6372/success-bot/internal/common/clients/finam"
	"github.com/leonid6372/success-bot/internal/common/config"
	"github.com/leonid6372/success-bot/internal/common/domain"
	"github.com/leonid6372/success-bot/pkg/cache"
	"github.com/leonid6372/success-bot/pkg/dictionary"
	"github.com/leonid6372/success-bot/pkg/errs"
	"github.com/leonid6372/success-bot/pkg/log"
	"go.uber.org/zap"
	"gopkg.in/telebot.v4"
)

type Bot struct {
	Telebot *telebot.Bot

	cfg *config.Bot
	ctx context.Context

	cache *cache.Cache // tgID -> *domain.User

	topUsers []*domain.TopUser // sorted by live-balance descending
	mu       sync.RWMutex

	deps *Dependencies
}

type Dependencies struct {
	finam      *finam.Client
	dictionary *dictionary.Dictionary

	usersRepository       domain.UsersRepository
	instrumentsRepository domain.InstrumentsRepository
	promocodesRepository  domain.PromocodesRepository
	operationsRepository  domain.OperationsRepository
	portfoliosRepository  domain.PortfolioRepository
}

func New(ctx context.Context,
	cfg *config.Bot,
	finam *finam.Client,
	dictionary *dictionary.Dictionary,
	usersRepository domain.UsersRepository,
	instrumentsRepository domain.InstrumentsRepository,
	promocodesRepository domain.PromocodesRepository,
	operationsRepository domain.OperationsRepository,
	portfoliosRepository domain.PortfolioRepository,
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
		ctx:     ctx,
		cache:   cache.New(16*time.Minute, 8*time.Minute),
		deps: &Dependencies{
			finam:                 finam,
			dictionary:            dictionary,
			usersRepository:       usersRepository,
			instrumentsRepository: instrumentsRepository,
			promocodesRepository:  promocodesRepository,
			operationsRepository:  operationsRepository,
			portfoliosRepository:  portfoliosRepository,
		},
	}

	if err := bot.setCommands(); err != nil {
		return nil, fmt.Errorf("bot.setCommands: %w", err)
	}

	bot.setupMiddlewares()
	bot.setupMessageRoutes()
	bot.setupCallbackRoutes()

	go bot.setupBalancesAndTopUpdater()
	go bot.setupStopOutProcessor()

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
	message.Handle(telebot.OnText, b.textHandler)

	for _, lang := range b.cfg.Languages {
		message.Handle(&telebot.Btn{Text: b.deps.dictionary.Text(lang, btnMainMenu)}, b.mainMenuHandler)
		message.Handle(&telebot.Btn{Text: b.deps.dictionary.Text(lang, btnPortfolio)}, b.portfolioHandler)
		message.Handle(&telebot.Btn{Text: b.deps.dictionary.Text(lang, btnOperations)}, b.operationsHandler)
		message.Handle(&telebot.Btn{Text: b.deps.dictionary.Text(lang, btnInstrumentsList)}, b.instrumentsListHandler)
		message.Handle(&telebot.Btn{Text: b.deps.dictionary.Text(lang, btnEnterPromocode)}, b.enterPromocodeHandler)
		// message.Handle(&telebot.Btn{Text: b.deps.dictionary.Text(lang, btnFAQ)}, b.faqHandler)
		message.Handle(&telebot.Btn{Text: b.deps.dictionary.Text(lang, btnTopUsers)}, b.topUsersHandler)
		message.Handle(&telebot.Btn{Text: b.deps.dictionary.Text(lang, btnBuy)}, b.buyHandler)
		message.Handle(&telebot.Btn{Text: b.deps.dictionary.Text(lang, btnSell)}, b.sellHandler)
	}
}

func (b *Bot) setupCallbackRoutes() {
	callback := b.Telebot.Group()

	callback.Handle(&telebot.Btn{Unique: cbkLanguage}, b.setLanguageHandler)
	callback.Handle(&telebot.Btn{Unique: cbkCheckSubscription}, b.checkSubscriptionHandler)
	callback.Handle(&telebot.Btn{Unique: cbkInstrumentsPage}, b.instrumentsListHandler)
	callback.Handle(&telebot.Btn{Unique: cbkInstrument}, b.instrumentHandler)
	callback.Handle(&telebot.Btn{Unique: cbkTopUsersPage}, b.topUsersHandler)
	callback.Handle(&telebot.Btn{Unique: cbkOperationsPage}, b.operationsHandler)
}

func (b *Bot) Start() {
	b.Telebot.Start()
}

func (b *Bot) Stop() {
	b.Telebot.Stop()
}

func (b *Bot) mustUser(c telebot.Context) *domain.User {
	tgID := c.Sender().ID

	user, ok := b.cache.Get(tgID)
	if !ok {
		log.Fatal("user not found in cache", zap.String("username", c.Sender().Username))
	}

	return user.(*domain.User)
}
