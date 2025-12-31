package bot

import (
	"context"
	"fmt"
	"math/rand/v2"
	"strconv"
	"time"

	"github.com/leonid6372/success-bot/internal/common/domain"
	"github.com/leonid6372/success-bot/pkg/dictionary"
	"github.com/leonid6372/success-bot/pkg/errs"
	"github.com/leonid6372/success-bot/pkg/log"
	"go.uber.org/zap"
	"gopkg.in/telebot.v4"
)

func (b *Bot) notSubscribedHandler(c telebot.Context) error {
	defer c.Respond()

	user := b.mustUser(c)

	text := b.deps.dictionary.Text(user.LanguageCode, msgNeedSubscribe)

	markup := b.subscribeKeyboard(user.LanguageCode, b.cfg.SubscribeChannelURL)

	if err := c.Send(text, &telebot.SendOptions{ReplyMarkup: markup}); err != nil {
		return errs.NewStack(err)
	}

	return nil
}

func (b *Bot) startHandler(c telebot.Context) error {
	ctx := c.Get(ctxContext).(context.Context)
	user := b.mustUser(c)

	if user.Metadata.InstrumentDone != nil {
		b.closeInstrument(user)
	}

	if user == nil {
		if err := b.deps.userRepository.CreateUser(ctx, &domain.User{
			ID:        c.Sender().ID,
			Username:  c.Sender().Username,
			FirstName: c.Sender().FirstName,
			LastName:  c.Sender().LastName,
			IsPremium: c.Sender().IsPremium,
		}); err != nil {
			return errs.NewStack(err)
		}

		return b.selectLanguageHandler(c)
	}

	return b.startMsg(c)
}

func (b *Bot) selectLanguageHandler(c telebot.Context) error {
	user := b.mustUser(c)

	if user.Metadata.InstrumentDone != nil {
		b.closeInstrument(user)
	}

	text := b.deps.dictionary.Text(dictionary.DefaultLanguage, msgLanguage)

	markup := b.languagesKeyboard()

	if err := c.Send(text, &telebot.SendOptions{ReplyMarkup: markup}); err != nil {
		return errs.NewStack(fmt.Errorf("failed to send message: %v", err))
	}

	return nil
}

func (b *Bot) startMsg(c telebot.Context) error {
	user := b.mustUser(c)

	if user.Metadata.InstrumentDone != nil {
		b.closeInstrument(user)
	}

	data := map[string]any{
		"ButtonInstrumentsList": b.deps.dictionary.Text(user.LanguageCode, btnInstrumentsList),
	}

	text := b.deps.dictionary.Text(user.LanguageCode, msgStart, data)

	if err := c.Send(text, &telebot.SendOptions{
		ReplyMarkup: b.mainMenuKeyboard(user.LanguageCode),
		ParseMode:   telebot.ModeHTML,
	}); err != nil {
		return errs.NewStack(fmt.Errorf("failed to send message: %v", err))
	}

	return nil
}

func (b *Bot) setLanguageHandler(c telebot.Context) error {
	defer c.Respond()

	ctx := c.Get(ctxContext).(context.Context)
	tgID := c.Sender().ID
	args := c.Args()

	if len(args) != 1 {
		return errs.NewStack(fmt.Errorf("failed to parse data: param language not found"))
	}

	langCode := args[0]

	// Update user in repository
	if err := b.deps.userRepository.UpdateUserLanguage(ctx, tgID, langCode); err != nil {
		return errs.NewStack(fmt.Errorf("failed to update user language_code in repository: %w", err))
	}

	// Update user in cache
	user := b.mustUser(c)
	user.LanguageCode = langCode
	//b.cache.SetDefault(user.ID, user) todo: check if needed because we use pointerss

	return b.startMsg(c)
}

func (b *Bot) checkSubscriptionHandler(c telebot.Context) error {
	defer c.Respond()

	sender := c.Sender()

	subscribed := true
	var err error

	if b.cfg.SubscribeChannelID != 0 {
		subscribed, err = b.checkSubscription(b.cfg.SubscribeChannelID, sender.ID)
		if err != nil {
			return errs.NewStack(fmt.Errorf("failed to get subscribed: %w", err))
		}
	}

	user := b.mustUser(c)

	if subscribed {
		if err := c.Delete(); err != nil {
			log.Warn("Failed to delete message", zap.Error(err))
		}

		text := b.deps.dictionary.Text(user.LanguageCode, msgSubscriptionSuccess)
		_, err := c.Bot().Send(c.Chat(), text)
		if err != nil {
			return errs.NewStack(fmt.Errorf("failed to send confirmation: %w", err))
		}
	} else {
		text := b.deps.dictionary.Text(user.LanguageCode, msgSubscriptionFailed)
		if c.Callback() != nil {
			return c.Respond(&telebot.CallbackResponse{Text: text})
		}
	}

	return nil
}

func (b *Bot) mainMenuHandler(c telebot.Context) error {
	user := b.mustUser(c)

	if user.Metadata.InstrumentDone != nil {
		b.closeInstrument(user)
	}

	if err := c.Send(c.Text(), &telebot.SendOptions{
		ReplyMarkup: b.mainMenuKeyboard(user.LanguageCode),
		ParseMode:   telebot.ModeHTML,
	}); err != nil {
		return errs.NewStack(fmt.Errorf("failed to send message: %v", err))
	}

	return nil
}

func (b *Bot) instrumentsListHandler(c telebot.Context) error {
	defer c.Respond()

	ctx := c.Get(ctxContext).(context.Context)
	user := b.mustUser(c)

	if user.Metadata.InstrumentDone != nil {
		b.closeInstrument(user)
	}

	var currentPage int64
	var err error

	args := c.Args()

	if len(args) == 1 {
		currentPage, err = strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return errs.NewStack(fmt.Errorf("failed to parse current page: %v", err))
		}
	} else {
		currentPage = 1
	}

	instrumentsListPagesCount, err := b.deps.instrumentsRepository.GetInstrumentsCount(ctx)
	if err != nil {
		return errs.NewStack(fmt.Errorf("failed to get instruments count: %v", err))
	}

	instruments, err := b.deps.instrumentsRepository.GetInstrumentsByPage(ctx, currentPage)

	data := map[string]any{
		"CurrentPage":             currentPage,
		"PagesCount":              instrumentsListPagesCount,
		"ButtonInstrumentsSearch": b.deps.dictionary.Text(user.LanguageCode, btnInstrumentsSearch),
	}

	text := b.deps.dictionary.Text(user.LanguageCode, msgInstrumentsList, data)

	markup := b.instrumentsListByPageKeyboard(
		user.LanguageCode, instruments, currentPage, instrumentsListPagesCount,
	)

	if err := c.Send(text, &telebot.SendOptions{ReplyMarkup: markup}); err != nil {
		return errs.NewStack(fmt.Errorf("failed to send message: %v", err))
	}

	return nil
}

func (b *Bot) instrumentHandler(c telebot.Context) error {
	defer c.Respond()

	user := b.mustUser(c)

	if user.Metadata.InstrumentDone != nil {
		b.closeInstrument(user)
	}

	args := c.Args()

	if len(args) != 1 {
		return errs.NewStack(fmt.Errorf("failed to parse data: param ticker not found"))
	}

	ticker := args[0]

	doneCh := make(chan struct{})
	user.Metadata.InstrumentDone = &doneCh
	//b.cache.SetDefault(user.ID, user) todo: check if needed because we use pointerss

	go func(user *domain.User) {
		data := map[string]any{
			"InstrumentTicker": ticker,
		}

		text := b.deps.dictionary.Text(user.LanguageCode, msgInstrument, data)
		if err := c.Send(text); err != nil {
			log.Error("failed to send message", zap.Error(err))
			return
		}

		for {
			select {
			case <-doneCh:
				return
			default:
				info, err := b.deps.finam.NewQuoteRequest(ticker).Do(b.deps.ctx)
				if err != nil {
					log.Error("failed to get instrument info", zap.Error(err))
				}

				var color string

				n := rand.IntN(2)
				//if info.Quote.Change.Float64() >= 0 {
				if n == 0 {
					color = "🟢"
				} else {
					color = "🔴"
				}

				text := b.deps.dictionary.Text(user.LanguageCode, btnLastPrice, map[string]any{
					"Color": color,
					"Price": info.Quote.Last,
				})

				markup := b.instrumentKeyboard(user.LanguageCode, &info)

				if err := c.Send(text, &telebot.SendOptions{ReplyMarkup: markup}); err != nil {
					log.Error("failed to send message", zap.Error(err))
				}
			}

			log.Info("ticker price circle")

			time.Sleep(500 * time.Millisecond)
		}
	}(user)

	return nil
}
