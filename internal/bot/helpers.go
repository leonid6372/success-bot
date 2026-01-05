package bot

import (
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/leonid6372/success-bot/internal/common/domain"
	"github.com/leonid6372/success-bot/pkg/errs"
	"github.com/leonid6372/success-bot/pkg/log"
	"go.uber.org/zap"
	"gopkg.in/telebot.v4"
)

func (b *Bot) setupBalancesAndTopUpdater() {
	for {
		select {
		case <-b.ctx.Done():
			log.Info("balances and top users updater shutting down...")
			return

		default:
			usersCount, err := b.deps.usersRepository.GetUsersCount(b.ctx)
			if err != nil {
				log.Error("failed to get users count", zap.Error(err))
				continue
			}

			usersInstrumentsCount, err := b.deps.portfoliosRepository.GetUsersInstrumentsCount(b.ctx)
			if err != nil {
				log.Error("failed to get users instruments count", zap.Error(err))
				continue
			}

			instruments := make(map[string]*domain.Instrument, usersInstrumentsCount)

			topUsersData, err := b.deps.usersRepository.GetTopUsersData(b.ctx)
			if err != nil {
				log.Error("failed to get top users data", zap.Error(err))
				continue
			}

			mapTopUsers := make(map[string]*domain.TopUser, usersCount)

			for _, data := range topUsersData {
				if _, ok := mapTopUsers[data.Username]; !ok {
					mapTopUsers[data.Username] = &domain.TopUser{
						ID:                 data.ID,
						Username:           data.Username,
						LanguageCode:       data.LanguageCode,
						AvailableBalance:   data.AvailableBalance,
						BlockedBalance:     data.BlockedBalance,
						BlockedBalanceDiff: data.BlockedBalance,
						MarginCall:         data.MarginCall,
					}
				}

				if data.Ticker == "" {
					continue
				}

				if _, ok := instruments[data.Ticker]; !ok {
					instrument, err := b.deps.finam.GetInstrumentPrices(b.ctx, data.Ticker)
					if err != nil {
						log.Error("failed to get ticker info", zap.String("ticker", data.Ticker), zap.Error(err))
					}

					instruments[data.Ticker] = instrument
				}

				if data.Count >= 0 {
					mapTopUsers[data.Username].TotalBalance += instruments[data.Ticker].Last * float64(data.Count)
					continue
				}

				mapTopUsers[data.Username].BlockedBalanceDiff +=
					instruments[data.Ticker].Last * float64(data.Count) * 0.5 // 50% guarantee coverage
			}

			topUsers := make([]*domain.TopUser, 0, len(mapTopUsers))
			for _, topUser := range mapTopUsers {
				topUser.AvailableBalance += topUser.BlockedBalanceDiff
				topUser.BlockedBalance -= topUser.BlockedBalanceDiff

				if !topUser.MarginCall && topUser.AvailableBalance < 0 {
					topUser.MarginCall = true

					b.Telebot.Send(
						&telebot.User{ID: topUser.ID},
						b.deps.dictionary.Text(topUser.LanguageCode, msgMarginCall),
						&telebot.SendOptions{ParseMode: telebot.ModeHTML},
					)
				}

				if topUser.MarginCall && topUser.AvailableBalance >= 0 {
					topUser.MarginCall = false
				}

				topUser.TotalBalance += topUser.AvailableBalance + topUser.BlockedBalance

				topUsers = append(topUsers, topUser)

				// Update data in repository
				if err := b.deps.usersRepository.UpdateUserBalancesAndMarginCall(
					b.ctx, topUser.ID, topUser.AvailableBalance, &topUser.BlockedBalanceDiff, &topUser.MarginCall,
				); err != nil {
					log.Error("failed to update user balances and margin call", zap.Int64("user_id", topUser.ID), zap.Error(err))
				}

				// Update data in cache
				rawUser, ok := b.cache.Get(topUser.ID)
				if ok {
					user := rawUser.(*domain.User)
					user.AvailableBalance = topUser.AvailableBalance
					user.BlockedBalance -= topUser.BlockedBalance
					user.MarginCall = topUser.MarginCall
				}
			}

			// Sort by balance descending
			sort.Slice(topUsers, func(i, j int) bool {
				return topUsers[i].TotalBalance > topUsers[j].TotalBalance
			})

			b.mu.Lock()
			b.topUsers = make([]*domain.TopUser, len(topUsers))
			copy(b.topUsers, topUsers)
			b.mu.Unlock()
		}

		time.Sleep(1 * time.Minute)
	}
}

func (b *Bot) setupStopOutProcessor() {
	moscow, _ := time.LoadLocation("Europe/Moscow")
	t := time.Now().In(moscow)

	for {
		startAt := time.Date(t.Year(), t.Month(), t.Day(), 23, 45, 0, 0, moscow)
		startCh := time.NewTimer(time.Until(startAt))

	outerLoop:
		for {
			select {
			case <-b.ctx.Done():
				log.Info("stop-out processor shutting down...")
				return

			case <-startCh.C:
				b.mu.RLock()
				for _, topUser := range b.topUsers {
					if topUser.MarginCall {
						userShort, err := b.deps.portfoliosRepository.GetUserMostExpensiveShort(b.ctx, topUser.ID)
						if err != nil {
							log.Error("failed to get user most expensive short",
								zap.Int64("user_id", topUser.ID),
								zap.Error(err),
							)

							continue
						}

						instrument, err := b.deps.finam.GetInstrumentPrices(b.ctx, userShort.Ticker)
						if err != nil {
							log.Error("failed to get instrument prices",
								zap.String("ticker", userShort.Ticker),
								zap.Error(err),
							)

							continue
						}

						closeCount := int64(0)
						for i := int64(1); i <= userShort.Count; i++ {
							if float64(instrument.Last)*float64(i)*0.5*0.997 >= -topUser.AvailableBalance { // 50% guarantee coverage and 3% fee for buying
								closeCount = i
								break
							}
						}

						if err := b.deps.portfoliosRepository.BuyInstrument(
							b.ctx, topUser.ID, userShort.ID, closeCount, instrument.Last,
						); err != nil {
							log.Error("failed to buy instrument",
								zap.Int64("user_id", topUser.ID),
								zap.String("ticker", userShort.Ticker),
								zap.Error(err),
							)

							continue
						}
					}
				}

				b.mu.RUnlock()
			}

			t = t.Add(24 * time.Hour)

			break outerLoop
		}
	}
}

func (b *Bot) closeInstrument(c telebot.Context, user *domain.User) error {
	close(*user.Metadata.InstrumentDone)
	user.Metadata.InstrumentDone = nil

	text := b.deps.dictionary.Text(user.LanguageCode, msgInstrumentExit)

	if err := c.Send(text, &telebot.SendOptions{
		ReplyMarkup: b.mainMenuKeyboard(user.LanguageCode),
	}); err != nil {
		return errs.NewStack(fmt.Errorf("failed to send message: %v", err))
	}

	return nil
}

func (b *Bot) getCurrentPage(c telebot.Context) (int64, error) {
	args := c.Args()

	if len(args) == 1 {
		currentPage, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return 0, errs.NewStack(fmt.Errorf("failed to parse current page: %v", err))
		}

		return currentPage, nil
	}

	return 1, nil
}

func (b *Bot) addPaginationCbkButtons(
	rows []telebot.Row, lang, cbkName string, currentPage, pagesCount int64,
) []telebot.Row {
	markup := &telebot.ReplyMarkup{}

	if pagesCount < 2 {
		return rows
	}

	if currentPage == 1 {
		rows = append(rows, telebot.Row{
			markup.Data(
				b.deps.dictionary.Text(lang, btnNextPage),
				fmt.Sprintf("%s|%d", cbkName, currentPage+1),
			),
		})
	}

	if currentPage == pagesCount {
		rows = append(rows, telebot.Row{
			markup.Data(
				b.deps.dictionary.Text(lang, btnPreviousPage),
				fmt.Sprintf("%s|%d", cbkName, currentPage-1),
			),
		})
	}

	if currentPage > 1 && currentPage < pagesCount {
		rows = append(rows, telebot.Row{
			markup.Data(
				b.deps.dictionary.Text(lang, btnPreviousPage),
				fmt.Sprintf("%s|%d", cbkName, currentPage-1),
			),
			markup.Data(
				b.deps.dictionary.Text(lang, btnNextPage),
				fmt.Sprintf("%s|%d", cbkName, currentPage+1),
			),
		})
	}

	return rows
}
