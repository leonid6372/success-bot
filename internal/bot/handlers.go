package bot

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/leonid6372/success-bot/internal/boterrs"
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

	if user != nil && user.Metadata.InstrumentDone != nil {
		b.closeInstrument(user)
	}

	if user == nil {
		user := &domain.User{
			ID:        c.Sender().ID,
			Username:  c.Sender().Username,
			FirstName: c.Sender().FirstName,
			LastName:  c.Sender().LastName,
			IsPremium: c.Sender().IsPremium,
		}

		if err := b.deps.usersRepository.CreateUser(ctx, user); err != nil {
			return errs.NewStack(err)
		}

		b.cache.SetDefault(user.ID, user)

		return b.selectLanguageHandler(c)
	}

	return b.startMsg(c)
}

func (b *Bot) selectLanguageHandler(c telebot.Context) error {
	user := b.mustUser(c)
	if user == nil {
		return errs.NewStack(fmt.Errorf("user not found in cache"))
	}

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

	text := b.deps.dictionary.Text(user.LanguageCode, msgStart, map[string]any{
		"ButtonInstrumentsList": b.deps.dictionary.Text(user.LanguageCode, btnInstrumentsList),
	})

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
	if err := b.deps.usersRepository.UpdateUserLanguage(ctx, tgID, langCode); err != nil {
		return errs.NewStack(fmt.Errorf("failed to update user language_code in repository: %w", err))
	}

	// Update user in cache
	user := b.mustUser(c)
	if user == nil {
		return errs.NewStack(fmt.Errorf("user not found in cache"))
	}

	user.LanguageCode = langCode

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

	text := b.deps.dictionary.Text(user.LanguageCode, msgMainMenu)

	if err := c.Send(text, &telebot.SendOptions{
		ReplyMarkup: b.mainMenuKeyboard(user.LanguageCode),
		ParseMode:   telebot.ModeHTML,
	}); err != nil {
		return errs.NewStack(fmt.Errorf("failed to send message: %v", err))
	}

	return nil
}

func (b *Bot) portfolioHandler(c telebot.Context) error {
	defer c.Respond()

	ctx := c.Get(ctxContext).(context.Context)
	user := b.mustUser(c)

	if user.Metadata.InstrumentDone != nil {
		b.closeInstrument(user)
	}

	currentPage, err := b.getCurrentPage(c)
	if err != nil {
		return errs.NewStack(err)
	}

	pagesCount, err := b.deps.portfoliosRepository.GetUserPortfolioPagesCount(ctx, user.ID)
	if err != nil {
		return errs.NewStack(fmt.Errorf("failed to get portfolio pages count: %v", err))
	}

	instruments, err := b.deps.portfoliosRepository.GetUserPortfolioByPage(ctx, user.ID, currentPage)
	if err != nil {
		return errs.NewStack(fmt.Errorf("failed to get user portfolio by page: %v", err))
	}

	var text string

	if len(instruments) == 0 {
		text = b.deps.dictionary.Text(user.LanguageCode, msgEmptyPortfolio, map[string]any{
			"AvailableBalance": user.AvailableBalance,
		})
	} else {
		text = b.deps.dictionary.Text(user.LanguageCode, msgPortfolio, map[string]any{
			"CurrentPage":      currentPage,
			"PagesCount":       pagesCount,
			"AvailableBalance": user.AvailableBalance,
			"BlockedBalance":   user.BlockedBalance,
		})
	}

	for _, instrument := range instruments {
		price, err := b.deps.finam.GetInstrumentPrices(ctx, instrument.Ticker)
		if err != nil {
			return errs.NewStack(fmt.Errorf("failed to get instrument prices: %v", err))
		}

		instrument.InstrumentPrices = price.InstrumentPrices
	}

	markup := b.portfolioInstrumentsListByPageKeyboard(
		user.LanguageCode, instruments, currentPage, pagesCount,
	)

	if err := c.Send(text, &telebot.SendOptions{ReplyMarkup: markup}); err != nil {
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

	currentPage, err := b.getCurrentPage(c)
	if err != nil {
		return errs.NewStack(err)
	}

	pagesCount, err := b.deps.instrumentsRepository.GetInstrumentsPagesCount(ctx)
	if err != nil {
		return errs.NewStack(fmt.Errorf("failed to get instruments pages count: %v", err))
	}

	instruments, err := b.deps.instrumentsRepository.GetInstrumentsByPage(ctx, currentPage)
	if err != nil {
		return errs.NewStack(fmt.Errorf("failed to get instruments by page: %v", err))
	}

	text := b.deps.dictionary.Text(user.LanguageCode, msgInstrumentsList, map[string]any{
		"CurrentPage":             currentPage,
		"PagesCount":              pagesCount,
		"ButtonInstrumentsSearch": b.deps.dictionary.Text(user.LanguageCode, btnInstrumentsSearch),
	})

	markup := b.instrumentsListByPageKeyboard(
		user.LanguageCode, instruments, currentPage, pagesCount,
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

	go func(user *domain.User) {
		text := b.deps.dictionary.Text(user.LanguageCode, msgInstrument, map[string]any{
			"InstrumentTicker": ticker,
		})

		if err := c.Send(text, &telebot.SendOptions{ParseMode: telebot.ModeHTML}); err != nil {
			log.Error("failed to send message", zap.Error(err))
			return
		}

		var prevPrice float64

		for {
			log.Info("ticker price circle")

			time.Sleep(500 * time.Millisecond)

			select {
			case <-doneCh:
				return
			default:
				instrument, err := b.deps.finam.GetInstrumentPrices(b.ctx, ticker)
				if err != nil {
					log.Error("failed to get instrument info", zap.Error(err))
				}

				// Skip if price didn't change
				if prevPrice == instrument.Last {
					continue
				}

				prevPrice = instrument.Last
				var color string

				if instrument.Change >= 0 {
					color = "🟢"
				} else {
					color = "🔴"
				}

				text := b.deps.dictionary.Text(user.LanguageCode, btnLastPrice, map[string]any{
					"Color": color,
					"Price": instrument.Last,
				})

				markup := b.instrumentKeyboard(user.LanguageCode, instrument)

				if err := c.Send(text, &telebot.SendOptions{ReplyMarkup: markup}); err != nil {
					log.Error("failed to send message", zap.Error(err))
				}
			}
		}
	}(user)

	return nil
}

func (b *Bot) enterPromocodeHandler(c telebot.Context) error {
	user := b.mustUser(c)

	user.Metadata.InputType = domain.InputTypePromocode

	text := b.deps.dictionary.Text(user.LanguageCode, msgEnterPromocode)

	if err := c.Send(text); err != nil {
		return errs.NewStack(fmt.Errorf("failed to send message: %v", err))
	}

	return nil
}

func (b *Bot) topUsersHandler(c telebot.Context) error {
	defer c.Respond()

	user := b.mustUser(c)

	currentPage, err := b.getCurrentPage(c)
	if err != nil {
		return errs.NewStack(err)
	}

	b.mu.RLock()
	pagesCount := int64(len(b.topUsers)/domain.UsersPerPage) + 1

	var text, usersList string

	if currentPage == 1 {
		var top1Username, top2Username, top3Username string
		var top1Balance, top2Balance, top3Balance float64

		switch len(b.topUsers) {
		case 1:
			top1Username = b.topUsers[0].Username
			top1Balance = b.topUsers[0].TotalBalance
		case 2:
			top1Username = b.topUsers[0].Username
			top1Balance = b.topUsers[0].TotalBalance
			top2Username = b.topUsers[1].Username
			top2Balance = b.topUsers[1].TotalBalance
		case 3:
			top1Username = b.topUsers[0].Username
			top1Balance = b.topUsers[0].TotalBalance
			top2Username = b.topUsers[1].Username
			top2Balance = b.topUsers[1].TotalBalance
			top3Username = b.topUsers[2].Username
			top3Balance = b.topUsers[2].TotalBalance
		default:
			top1Username = b.topUsers[0].Username
			top1Balance = b.topUsers[0].TotalBalance
			top2Username = b.topUsers[1].Username
			top2Balance = b.topUsers[1].TotalBalance
			top3Username = b.topUsers[2].Username
			top3Balance = b.topUsers[2].TotalBalance

			for i := 3; i < min(domain.UsersPerPage, len(b.topUsers)); i++ {
				usersList += fmt.Sprintf("\n%d. %s %.2f L$", i+1, b.topUsers[i].Username, b.topUsers[i].TotalBalance)
			}
			b.mu.RUnlock()
		}

		text = b.deps.dictionary.Text(user.LanguageCode, msgTopUsersFirstPage, map[string]any{
			"CurrentPage":  currentPage,
			"PagesCount":   pagesCount,
			"Top1Username": top1Username,
			"Top1Balance":  top1Balance,
			"Top2Username": top2Username,
			"Top2Balance":  top2Balance,
			"Top3Username": top3Username,
			"Top3Balance":  top3Balance,
			"UsersList":    usersList,
		})
	} else {
		for i := domain.UsersPerPage * (currentPage - 1); i < min(domain.UsersPerPage*currentPage, int64(len(b.topUsers))); i++ {
			usersList += fmt.Sprintf("\n%d. %s %.2f L$", i+1, b.topUsers[i].Username, b.topUsers[i].TotalBalance)
		}
		b.mu.RUnlock()

		text = b.deps.dictionary.Text(user.LanguageCode, msgTopUsers, map[string]any{
			"CurrentPage": currentPage,
			"PagesCount":  pagesCount,
			"UsersList":   usersList,
		})
	}

	markup := b.paginationKeyboard(user.LanguageCode, cbkTopUsersPage, currentPage, pagesCount)

	if err := c.Send(text, &telebot.SendOptions{ReplyMarkup: markup, ParseMode: telebot.ModeHTML}); err != nil {
		return errs.NewStack(fmt.Errorf("failed to send message: %v", err))
	}

	return nil
}

func (b *Bot) textHandler(c telebot.Context) error {
	user := b.mustUser(c)

	if user.Metadata.InputType == "" {
		return b.mainMenuHandler(c)
	}

	switch user.Metadata.InputType {
	case domain.InputTypePromocode:
		return b.inputPromocode(c)
	// case domain.InputTypeTicker:
	// return b.inputTicker(c)
	default:
		user.Metadata.InputType = ""
		return c.Send(b.deps.dictionary.Text(user.LanguageCode, msgDefaultError))
	}
}

func (b *Bot) inputPromocode(c telebot.Context) error {
	ctx := c.Get(ctxContext).(context.Context)
	user := b.mustUser(c)
	promocodeValue := c.Text()

	var text string

	promocode, err := b.deps.promocodesRepository.ApplyPromocode(ctx, promocodeValue, user.ID)
	switch {
	case errors.Is(err, boterrs.ErrInvalidPromocode):
		text = b.deps.dictionary.Text(user.LanguageCode, msgInvalidPromocode)
	case errors.Is(err, boterrs.ErrUsedPromocode):
		text = b.deps.dictionary.Text(user.LanguageCode, msgPromocodeAlreadyUsed)
	case err == nil:
		text = b.deps.dictionary.Text(user.LanguageCode, msgSuccessfulPromocode, map[string]any{
			"Amount": promocode.BonusAmount,
		})
	default:
		return errs.NewStack(fmt.Errorf("failed to apply promocode: %v", err))
	}

	user.Metadata.InputType = ""

	if err := c.Send(text); err != nil {
		return errs.NewStack(fmt.Errorf("failed to send message: %v", err))
	}

	return nil
}

func (b *Bot) operationsHandler(c telebot.Context) error {
	defer c.Respond()

	ctx := c.Get(ctxContext).(context.Context)
	user := b.mustUser(c)

	currentPage, err := b.getCurrentPage(c)
	if err != nil {
		return errs.NewStack(err)
	}

	pagesCount, err := b.deps.operationsRepository.GetOperationsPagesCount(ctx)
	if err != nil {
		return errs.NewStack(fmt.Errorf("failed to get operations pages count: %v", err))
	}

	operations, err := b.deps.operationsRepository.GetOperationsByPage(ctx, user.ID, currentPage)
	if err != nil {
		return errs.NewStack(fmt.Errorf("failed to get operations by page: %v", err))
	}

	var text strings.Builder

	if len(operations) == 0 {
		text.WriteString(b.deps.dictionary.Text(user.LanguageCode, msgNoOperations))
	} else {
		text.WriteString(b.deps.dictionary.Text(user.LanguageCode, msgOperations, map[string]any{
			"CurrentPage": currentPage,
			"PagesCount":  pagesCount,
		}))
	}

	for _, op := range operations {
		switch op.Type {
		case domain.OperationTypeBuy:
			text.WriteString(b.deps.dictionary.Text(user.LanguageCode, msgOperationBuy, map[string]any{
				"OperationID": op.ID,
				"Count":       op.Count,
				"Name":        op.InstrumentName[5:], // cut instrument emoji
				"Amount":      op.TotalAmount,
			}))

		case domain.OperationTypeSell:
			text.WriteString(b.deps.dictionary.Text(user.LanguageCode, msgOperationSell, map[string]any{
				"OperationID": op.ID,
				"Count":       op.Count,
				"Name":        op.InstrumentName[5:], // cut instrument emoji
				"Amount":      op.TotalAmount,
			}))

		case domain.OperationTypeFee:
			text.WriteString(b.deps.dictionary.Text(user.LanguageCode, msgOperationFee, map[string]any{
				"OperationID": op.ParentID,
				"Amount":      op.TotalAmount,
			}))

		case domain.OperationTypePromocode:
			text.WriteString(b.deps.dictionary.Text(user.LanguageCode, msgOperationPromocode, map[string]any{
				"Name":   op.InstrumentName,
				"Amount": op.TotalAmount,
			}))

		default:
			log.Error("unknown operation type", zap.String("type", string(op.Type)))
		}
	}

	markup := b.paginationKeyboard(user.LanguageCode, cbkOperationsPage, currentPage, pagesCount)

	if err := c.Send(text.String(), &telebot.SendOptions{ReplyMarkup: markup, ParseMode: telebot.ModeHTML}); err != nil {
		return errs.NewStack(fmt.Errorf("failed to send message: %v", err))
	}

	return nil
}
