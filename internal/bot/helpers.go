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

			mapTopUsers[data.Username].BlockedBalance +=
				instruments[data.Ticker].Last * float64(data.Count) * 0.5 // 50% guarantee coverage
		}

		topUsers := make([]*domain.TopUser, 0, len(mapTopUsers))
		for _, topUser := range mapTopUsers {
			topUser.AvailableBalance += topUser.BlockedBalance

			marginCall := false
			if topUser.AvailableBalance < 0 {
				marginCall = true

				b.Telebot.Send(
					&telebot.User{ID: topUser.ID},
					b.deps.dictionary.Text(topUser.LanguageCode, msgMarginCall),
					&telebot.SendOptions{ParseMode: telebot.ModeHTML},
				)
			}

			topUser.TotalBalance += topUser.AvailableBalance + topUser.BlockedBalance

			topUsers = append(topUsers, topUser)

			// Update data in repository
			if err := b.deps.usersRepository.UpdateUserBalancesAndMarginCall(
				b.ctx, topUser.ID, topUser.AvailableBalance, &topUser.BlockedBalance, &marginCall,
			); err != nil {
				log.Error("failed to update user balances and margin call", zap.Int64("user_id", topUser.ID), zap.Error(err))
			}

			// Update data in cache
			rawUser, ok := b.cache.Get(topUser.ID)
			if ok {
				user := rawUser.(*domain.User)
				user.AvailableBalance = topUser.AvailableBalance
				user.BlockedBalance = topUser.BlockedBalance
				user.MarginCall = marginCall
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

		time.Sleep(5 * time.Minute)
	}
}

func (b *Bot) closeInstrument(user *domain.User) {
	close(*user.Metadata.InstrumentDone)
	user.Metadata.InstrumentDone = nil
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

	if pagesCount == 1 {
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
