package bot

import (
	"context"
	"errors"
	"fmt"
	"strconv"
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
		if err := b.closeInstrument(c, user); err != nil {
			return errs.NewStack(err)
		}
	}

	if user == nil {
		user := &domain.User{
			ID:               c.Sender().ID,
			Username:         c.Sender().Username,
			FirstName:        c.Sender().FirstName,
			LastName:         c.Sender().LastName,
			IsPremium:        c.Sender().IsPremium,
			AvailableBalance: 250000,
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
		if err := b.closeInstrument(c, user); err != nil {
			return errs.NewStack(err)
		}
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
		if err := b.closeInstrument(c, user); err != nil {
			return errs.NewStack(err)
		}
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
			log.Warn("Failed to delete message", zap.String("username", sender.Username), zap.Error(err))
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
		if err := b.closeInstrument(c, user); err != nil {
			return errs.NewStack(err)
		}
	}

	text := b.deps.dictionary.Text(user.LanguageCode, msgMainMenu)

	if err := c.Send(text, &telebot.SendOptions{
		ReplyMarkup: b.mainMenuKeyboard(user.LanguageCode),
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
		if err := b.closeInstrument(c, user); err != nil {
			return errs.NewStack(err)
		}
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
		var warning string
		if user.MarginCall {
			warning = b.deps.dictionary.Text(user.LanguageCode, msgMarginCallWarning)
		}

		text = b.deps.dictionary.Text(user.LanguageCode, msgPortfolio, map[string]any{
			"Warning":          warning,
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

	if err := c.Send(text, &telebot.SendOptions{
		ReplyMarkup: markup,
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
		if err := b.closeInstrument(c, user); err != nil {
			return errs.NewStack(err)
		}
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

	if err := c.Send(text, &telebot.SendOptions{
		ReplyMarkup: markup,
		ParseMode:   telebot.ModeHTML,
	}); err != nil {
		return errs.NewStack(fmt.Errorf("failed to send message: %v", err))
	}

	return nil
}

func (b *Bot) instrumentHandler(c telebot.Context) error {
	defer c.Respond()

	user := b.mustUser(c)

	if user.Metadata.InstrumentDone != nil {
		if err := b.closeInstrument(c, user); err != nil {
			return errs.NewStack(err)
		}
	}

	args := c.Args()

	if len(args) != 1 {
		return errs.NewStack(fmt.Errorf("failed to parse data: param ticker not found"))
	}

	ticker := args[0]

	instrument, err := b.deps.instrumentsRepository.GetInstrumentByTicker(b.ctx, ticker)
	if err != nil {
		return errs.NewStack(fmt.Errorf("failed to get instrument by ticker: %v", err))
	}

	doneCh := make(chan struct{})
	user.Metadata.InstrumentDone = &doneCh

	go func(user *domain.User) {
		text := b.deps.dictionary.Text(user.LanguageCode, msgInstrument, map[string]any{
			"InstrumentName":   instrument.Name,
			"InstrumentTicker": instrument.Ticker,
		})

		if err := c.Send(text, &telebot.SendOptions{ParseMode: telebot.ModeHTML}); err != nil {
			log.Error("failed to send message", zap.String("username", user.Username), zap.Error(err))
			return
		}

		var prevPrice float64

		user.Metadata.InstrumentTicker = ticker

		for {
			time.Sleep(500 * time.Millisecond)

			log.Info("ticker price circle", zap.String("username", user.Username), zap.String("ticker", ticker))
			select {
			case <-doneCh:
				return
			default:
				instrumentPrices, err := b.deps.finam.GetInstrumentPrices(b.ctx, ticker)
				if err != nil {
					log.Error("failed to get instrument info", zap.String("username", user.Username), zap.Error(err))
				}

				instrumentInfo, err := b.deps.finam.GetInstrumentInfo(b.ctx, ticker)
				if err != nil {
					log.Error("failed to get instrument info", zap.String("username", user.Username), zap.Error(err))
					continue
				}

				// Skip if price didn't change
				if prevPrice == instrumentPrices.Last {
					continue
				}

				var color string

				if instrumentPrices.Last > prevPrice {
					color = "🟢"
				} else {
					color = "🔴"
				}

				text := b.deps.dictionary.Text(user.LanguageCode, msgLastPrice, map[string]any{
					"Color": color,
					"Price": fmt.Sprintf(`%.*f`, instrumentInfo.Decimals, instrumentPrices.Last),
				})

				if instrumentPrices.Ask == 0 && instrumentPrices.Bid == 0 {
					text += "\n\n" + b.deps.dictionary.Text(user.LanguageCode, msgClosedExchange)

					if err := c.Send(text, &telebot.SendOptions{ParseMode: telebot.ModeHTML}); err != nil {
						log.Error("failed to send message", zap.String("username", user.Username), zap.Error(err))
					}

					if err := b.closeInstrument(c, user); err != nil {
						log.Error("failed to close instrument", zap.String("username", user.Username), zap.Error(err))
					}

					return
				}

				prevPrice = instrumentPrices.Last

				user.Metadata.InstrumentBuyPrice = instrumentPrices.Ask
				user.Metadata.InstrumentSellPrice = instrumentPrices.Bid

				markup := b.instrumentKeyboard(user.LanguageCode, instrumentPrices)

				if err := c.Send(text, &telebot.SendOptions{ReplyMarkup: markup}); err != nil {
					log.Error("failed to send message", zap.String("username", user.Username), zap.Error(err))
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
		case 0:
			b.mu.RUnlock()
		case 1:
			top1Username = b.topUsers[0].Username
			top1Balance = b.topUsers[0].TotalBalance

			b.mu.RUnlock()
		case 2:
			top1Username = b.topUsers[0].Username
			top1Balance = b.topUsers[0].TotalBalance
			top2Username = b.topUsers[1].Username
			top2Balance = b.topUsers[1].TotalBalance

			b.mu.RUnlock()
		case 3:
			top1Username = b.topUsers[0].Username
			top1Balance = b.topUsers[0].TotalBalance
			top2Username = b.topUsers[1].Username
			top2Balance = b.topUsers[1].TotalBalance
			top3Username = b.topUsers[2].Username
			top3Balance = b.topUsers[2].TotalBalance

			b.mu.RUnlock()
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

	if err := c.Send(text, &telebot.SendOptions{
		ReplyMarkup: markup,
		ParseMode:   telebot.ModeHTML,
	}); err != nil {
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
	case domain.InputTypeCount:
		switch user.Metadata.InstrumentOperation {
		case domain.OperationTypeBuy:
			return b.inputCountToBuy(c)
		case domain.OperationTypeSell:
			return b.inputCountToSell(c)
		default:
			log.Error("invalid user operation type",
				zap.String("username", user.Username),
				zap.String("operation_type", user.Metadata.InstrumentOperation),
			)

			user.Metadata.InputType = ""
			user.Metadata.InstrumentOperation = ""

			return c.Send(b.deps.dictionary.Text(user.LanguageCode, msgDefaultError))
		}
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

func (b *Bot) inputCountToBuy(c telebot.Context) error {
	ctx := c.Get(ctxContext).(context.Context)
	user := b.mustUser(c)

	txtCount := c.Text()

	count, err := strconv.ParseInt(txtCount, 10, 64)
	if err != nil {
		text := b.deps.dictionary.Text(user.LanguageCode, msgInvalidCount)

		if err := c.Send(text); err != nil {
			return errs.NewStack(fmt.Errorf("failed to send message: %v", err))
		}

		return errs.NewStack(fmt.Errorf("failed to parse count input: %v", err))
	}

	instrument, err := b.deps.instrumentsRepository.GetInstrumentByTicker(ctx, user.Metadata.InstrumentTicker)
	if err != nil {
		return errs.NewStack(fmt.Errorf("failed to get instrument by ticker: %v", err))
	}

	var text string

	err = b.deps.portfoliosRepository.BuyInstrument(ctx, user.ID, instrument.ID, count, user.Metadata.InstrumentBuyPrice)
	switch {
	case errors.Is(err, boterrs.ErrInsufficientFunds):
		text = b.deps.dictionary.Text(user.LanguageCode, msgInsufficientFunds)
	case err == nil:
		text = b.deps.dictionary.Text(user.LanguageCode, msgSuccessfulBuy, map[string]any{
			"Count":          count,
			"InstrumentName": instrument.Name,
			"Price":          user.Metadata.InstrumentBuyPrice,
		})
	default:
		return errs.NewStack(fmt.Errorf("failed to apply promocode: %v", err))
	}

	user.Metadata.InputType = ""
	user.Metadata.InstrumentTicker = ""
	user.Metadata.InstrumentOperation = ""

	if err := c.Send(text); err != nil {
		return errs.NewStack(fmt.Errorf("failed to send message: %v", err))
	}

	return nil
}

func (b *Bot) inputCountToSell(c telebot.Context) error {
	ctx := c.Get(ctxContext).(context.Context)
	user := b.mustUser(c)

	txtCount := c.Text()

	count, err := strconv.ParseInt(txtCount, 10, 64)
	if err != nil {
		text := b.deps.dictionary.Text(user.LanguageCode, msgInvalidCount)

		if err := c.Send(text); err != nil {
			return errs.NewStack(fmt.Errorf("failed to send message: %v", err))
		}

		return errs.NewStack(fmt.Errorf("failed to parse count input: %v", err))
	}

	instrument, err := b.deps.instrumentsRepository.GetInstrumentByTicker(ctx, user.Metadata.InstrumentTicker)
	if err != nil {
		return errs.NewStack(fmt.Errorf("failed to get instrument by ticker: %v", err))
	}

	var text string

	err = b.deps.portfoliosRepository.SellInstrument(ctx, user.ID, instrument.ID, count, user.Metadata.InstrumentSellPrice)
	switch {
	case errors.Is(err, boterrs.ErrInsufficientFunds):
		text = b.deps.dictionary.Text(user.LanguageCode, msgInsufficientFunds)
	case err == nil:
		text = b.deps.dictionary.Text(user.LanguageCode, msgSuccessfulSell, map[string]any{
			"Count":          count,
			"InstrumentName": instrument.Name,
			"Price":          user.Metadata.InstrumentSellPrice,
		})
	default:
		return errs.NewStack(fmt.Errorf("failed to apply promocode: %v", err))
	}

	user.Metadata.InputType = ""
	user.Metadata.InstrumentTicker = ""
	user.Metadata.InstrumentOperation = ""

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

	pagesCount, err := b.deps.operationsRepository.GetOperationsPagesCount(ctx, user.ID)
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
				"Name":        strings.Split(op.InstrumentName, " ")[1], // cut instrument emoji
				"Amount":      op.TotalAmount,
			}))

		case domain.OperationTypeSell:
			text.WriteString(b.deps.dictionary.Text(user.LanguageCode, msgOperationSell, map[string]any{
				"OperationID": op.ID,
				"Count":       op.Count,
				"Name":        strings.Split(op.InstrumentName, " ")[1], // cut instrument emoji
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
			log.Error("unknown operation type", zap.String("username", user.Username), zap.String("type", string(op.Type)))
		}
	}

	markup := b.paginationKeyboard(user.LanguageCode, cbkOperationsPage, currentPage, pagesCount)

	if err := c.Send(text.String(), &telebot.SendOptions{
		ReplyMarkup: markup,
		ParseMode:   telebot.ModeHTML,
	}); err != nil {
		return errs.NewStack(fmt.Errorf("failed to send message: %v", err))
	}

	return nil
}

func (b *Bot) buyHandler(c telebot.Context) error {
	user := b.mustUser(c)

	if user.Metadata.InstrumentTicker == "" {
		return errs.NewStack(boterrs.ErrEmptyTickerToBuy)
	}

	if user.Metadata.InstrumentDone != nil {
		if err := b.closeInstrument(c, user); err != nil {
			return errs.NewStack(err)
		}
	}

	user.Metadata.InputType = domain.InputTypeCount
	user.Metadata.InstrumentOperation = domain.OperationTypeBuy

	maxCount, err := b.deps.portfoliosRepository.GetMaxInstrumentCountToBuy(
		b.ctx, user.ID, user.Metadata.InstrumentTicker, user.Metadata.InstrumentBuyPrice,
	)
	if err != nil {
		return errs.NewStack(fmt.Errorf("failed to get max count to buy: %v", err))
	}

	text := b.deps.dictionary.Text(user.LanguageCode, msgEnterCountToBuy, map[string]any{
		"Price":    user.Metadata.InstrumentBuyPrice,
		"MaxCount": maxCount,
	})

	if err := c.Send(text); err != nil {
		return errs.NewStack(fmt.Errorf("failed to send message: %v", err))
	}

	return nil
}

func (b *Bot) sellHandler(c telebot.Context) error {
	user := b.mustUser(c)

	if user.Metadata.InstrumentTicker == "" {
		return errs.NewStack(boterrs.ErrEmptyTickerToSell)
	}

	if user.Metadata.InstrumentDone != nil {
		if err := b.closeInstrument(c, user); err != nil {
			return errs.NewStack(err)
		}
	}

	user.Metadata.InputType = domain.InputTypeCount
	user.Metadata.InstrumentOperation = domain.OperationTypeSell

	maxCount, err := b.deps.portfoliosRepository.GetMaxInstrumentCountToSell(
		b.ctx, user.ID, user.Metadata.InstrumentTicker, user.Metadata.InstrumentSellPrice,
	)
	if err != nil {
		return errs.NewStack(fmt.Errorf("failed to get max count to sell: %v", err))
	}

	text := b.deps.dictionary.Text(user.LanguageCode, msgEnterCountToSell, map[string]any{
		"Price":    user.Metadata.InstrumentSellPrice,
		"MaxCount": maxCount,
	})

	if err := c.Send(text); err != nil {
		return errs.NewStack(fmt.Errorf("failed to send message: %v", err))
	}

	return nil
}
