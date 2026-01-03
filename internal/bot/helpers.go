package bot

import (
	"fmt"
	"sort"
	"time"

	"github.com/leonid6372/success-bot/internal/common/domain"
	"github.com/leonid6372/success-bot/pkg/log"
	"go.uber.org/zap"
	"gopkg.in/telebot.v4"
)

func (b *Bot) closeInstrument(user *domain.User) {
	close(*user.Metadata.InstrumentDone)
	user.Metadata.InstrumentDone = nil
}

func (b *Bot) setupTopUsersUpdater() {
	for {
		usersCount, err := b.deps.usersRepository.GetUsersCount(b.deps.ctx)
		if err != nil {
			log.Error("failed to get users count", zap.Error(err))
			continue
		}

		topUsersData, err := b.deps.usersRepository.GetTopUsersData(b.deps.ctx)
		if err != nil {
			log.Error("failed to get top users data", zap.Error(err))
			continue
		}

		mapTopUsers := make(map[string]float64, usersCount)

		for _, data := range topUsersData {
			if _, ok := mapTopUsers[data.Username]; !ok {
				mapTopUsers[data.Username] = data.Balance
			}

			if data.Ticker == "" {
				continue
			}

			instrument, err := b.deps.finam.GetInstrumentPrices(b.deps.ctx, data.Ticker)
			if err != nil {
				log.Error("failed to get ticker info", zap.String("ticker", data.Ticker), zap.Error(err))
			}

			mapTopUsers[data.Username] += instrument.Price.Last * float64(data.Count)
		}

		topUsers := make([]*domain.TopUser, 0, len(mapTopUsers))
		for username, balance := range mapTopUsers {
			topUsers = append(topUsers, &domain.TopUser{
				Username: username,
				Balance:  balance,
			})
		}

		// sort by balance descending
		sort.Slice(topUsers, func(i, j int) bool {
			return topUsers[i].Balance > topUsers[j].Balance
		})

		b.mu.Lock()
		b.topUsers = make([]*domain.TopUser, len(topUsers))
		copy(b.topUsers, topUsers)
		b.mu.Unlock()

		time.Sleep(5 * time.Minute)
	}
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
