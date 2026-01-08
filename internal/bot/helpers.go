package bot

import (
	"fmt"
	"sort"
	"strconv"
	"time"

	_ "time/tzdata"

	"github.com/leonid6372/success-bot/internal/common/domain"
	"github.com/leonid6372/success-bot/pkg/errs"
	"github.com/leonid6372/success-bot/pkg/log"
	"go.uber.org/zap"
	"gopkg.in/telebot.v4"
)

// setupCacheUpdater setups a goroutine that updates instruments cache every minute.
// Also updates user's blocked balances and top users list using actual instrument prices.
func (b *Bot) setupCacheUpdater() {
	for {
		select {
		case <-b.ctx.Done():
			log.Info("balances and top users updater shutting down...")
			return

		default:
			// update instuments cache
			tickers, err := b.deps.portfoliosRepository.GetUsersInstrumentTickers(b.ctx)
			if err != nil {
				log.Error("failed to get users instrument tickers", zap.Error(err))
				continue
			}

			for _, ticker := range tickers {
				instrument, err := b.deps.finam.GetInstrumentPrices(b.ctx, ticker)
				if err != nil {
					log.Error("failed to get instrument prices from finam", zap.String("ticker", ticker), zap.Error(err))
					continue
				}

				b.instruments.SetDefault(ticker, instrument)
			}

			usersCount, err := b.deps.usersRepository.GetUsersCount(b.ctx)
			if err != nil {
				log.Error("failed to get users count", zap.Error(err))
				continue
			}

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

				instrument, err := b.getInstrumentPrices(b.ctx, data.Ticker)
				if err != nil {
					log.Error("failed to get instrument prices", zap.String("ticker", data.Ticker), zap.Error(err))
					continue
				}

				if data.Count >= 0 {
					mapTopUsers[data.Username].TotalBalance += instrument.Last * float64(data.Count)
					continue
				}

				mapTopUsers[data.Username].BlockedBalanceDiff +=
					instrument.Last * float64(data.Count) * 0.5 // 50% guarantee coverage
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
				rawUser, ok := b.users.Get(topUser.ID)
				if ok {
					user := rawUser.(*domain.User)
					user.AvailableBalance = topUser.AvailableBalance
					user.BlockedBalance = topUser.BlockedBalance
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

// setupDailyProcessor setups a goroutine that processes daily tasks at 23:45 Moscow time.
// It processes stop-out for users with margin call and daily reward messages.
func (b *Bot) setupDailyProcessor() {
	moscow, _ := time.LoadLocation("Europe/Moscow")
	t := time.Now().In(moscow)

	dailyRewardT := t
	stopOutT := t

	for {
		dailyRewardAt := time.Date(dailyRewardT.Year(), dailyRewardT.Month(), dailyRewardT.Day(), 8, 0, 0, 0, moscow)
		dailyRewardCh := time.NewTimer(time.Until(dailyRewardAt))

		stopOutAt := time.Date(stopOutT.Year(), stopOutT.Month(), stopOutT.Day(), 23, 45, 0, 0, moscow)
		stopOutCh := time.NewTimer(time.Until(stopOutAt))

	outerLoop:
		for {
			select {
			case <-b.ctx.Done():
				log.Info("stop-out processor shutting down...")
				return

			case <-dailyRewardCh.C:
				users, err := b.deps.usersRepository.GetUsersClaimedDailyReward(b.ctx)
				if err != nil {
					log.Error("failed to get users claimed daily reward", zap.Error(err))
					continue
				}

				if err := b.deps.usersRepository.ResetDailyReward(b.ctx); err != nil {
					log.Error("failed to reset daily reward for all users", zap.Error(err))
					continue
				}

				for _, user := range users {
					text := b.deps.dictionary.Text(user.LanguageCode, msgDailyReward, map[string]any{
						"Amount": b.cfg.DailyReward,
					})

					markup := b.dailyRewardKeyboard(user.LanguageCode)

					if _, err := b.Telebot.Send(&telebot.User{ID: user.ID},
						text,
						&telebot.SendOptions{ReplyMarkup: markup, ParseMode: telebot.ModeHTML},
					); err != nil {
						log.Error("failed to send message", zap.String("username", user.Username), zap.Error(err))
					}
				}

				dailyRewardT = dailyRewardT.Add(24 * time.Hour)

				break outerLoop

			case <-stopOutCh.C:
				b.mu.RLock()
				for _, topUser := range b.topUsers {
					if topUser.MarginCall {
						userShort, err := b.deps.portfoliosRepository.GetUserMostExpensiveShort(b.ctx, topUser.ID)
						if err != nil {
							log.Error("failed to get user most expensive short",
								zap.String("username", topUser.Username),
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
						for i := int64(1); i <= -userShort.Count; i++ {
							if instrument.Last*0.497*float64(i)-((instrument.Last-userShort.AvgPrice)*float64(i)) >= -topUser.AvailableBalance { // 50% guarantee coverage and 0,3% fee for buying
								closeCount = i
								break
							}
						}

						if err := b.deps.portfoliosRepository.BuyInstrument(
							b.ctx, topUser.ID, userShort.ID, closeCount, instrument.Last,
						); err != nil {
							log.Error("failed to buy instrument",
								zap.String("username", topUser.Username),
								zap.String("ticker", userShort.Ticker),
								zap.Error(err),
							)

							continue
						}
					}
				}

				b.mu.RUnlock()

				stopOutT = stopOutT.Add(24 * time.Hour)

				break outerLoop
			}
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
				b.deps.dictionary.Text(lang, btnNext),
				fmt.Sprintf("%s|%d", cbkName, currentPage+1),
			),
		})
	}

	if currentPage == pagesCount {
		rows = append(rows, telebot.Row{
			markup.Data(
				b.deps.dictionary.Text(lang, btnPrevious),
				fmt.Sprintf("%s|%d", cbkName, currentPage-1),
			),
		})
	}

	if currentPage > 1 && currentPage < pagesCount {
		rows = append(rows, telebot.Row{
			markup.Data(
				b.deps.dictionary.Text(lang, btnPrevious),
				fmt.Sprintf("%s|%d", cbkName, currentPage-1),
			),
			markup.Data(
				b.deps.dictionary.Text(lang, btnNext),
				fmt.Sprintf("%s|%d", cbkName, currentPage+1),
			),
		})
	}

	return rows
}
