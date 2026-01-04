package bot

import (
	"fmt"

	"github.com/leonid6372/success-bot/internal/common/domain"
	"gopkg.in/telebot.v4"
)

func (b *Bot) mainMenuKeyboard(lang string) *telebot.ReplyMarkup {
	markup := &telebot.ReplyMarkup{}

	btnPortfolio := telebot.Btn{Text: b.deps.dictionary.Text(lang, btnPortfolio)}
	btnOperations := telebot.Btn{Text: b.deps.dictionary.Text(lang, btnOperations)}
	btnInstrumentsList := telebot.Btn{Text: b.deps.dictionary.Text(lang, btnInstrumentsList)}
	btnInstrumentsSearch := telebot.Btn{Text: b.deps.dictionary.Text(lang, btnInstrumentsSearch)}
	btnEnterPromocode := telebot.Btn{Text: b.deps.dictionary.Text(lang, btnEnterPromocode)}
	btnFAQ := telebot.Btn{Text: b.deps.dictionary.Text(lang, btnFAQ)}
	btnTopUsers := telebot.Btn{Text: b.deps.dictionary.Text(lang, btnTopUsers)}

	rows := []telebot.Row{
		{btnPortfolio, btnOperations},
		{btnInstrumentsList, btnInstrumentsSearch},
		{btnEnterPromocode, btnFAQ},
		{btnTopUsers},
	}

	markup.Reply(rows...)
	markup.ResizeKeyboard = true
	return markup
}

func (b *Bot) subscribeKeyboard(lang string, channelLink string) *telebot.ReplyMarkup {
	markup := &telebot.ReplyMarkup{}

	subscribeBtn := markup.URL(b.deps.dictionary.Text(lang, btnSubscribe), channelLink)
	checkBtn := markup.Data(b.deps.dictionary.Text(lang, btnSubscribed), cbkCheckSubscription)

	rows := []telebot.Row{
		{subscribeBtn},
		{checkBtn},
	}

	markup.Inline(rows...)

	return markup
}

func (b *Bot) languagesKeyboard() *telebot.ReplyMarkup {
	markup := &telebot.ReplyMarkup{}
	var rows []telebot.Row

	for _, lang := range b.cfg.Languages {
		text := b.deps.dictionary.Text(lang, btnLanguage)
		callbackData := fmt.Sprintf("%s|%s", cbkLanguage, lang)

		btn := markup.Data(text, callbackData)
		rows = append(rows, telebot.Row{btn})
	}

	markup.Inline(rows...)
	return markup
}

func (b *Bot) portfolioInstrumentsListByPageKeyboard(
	lang string, instruments []*domain.UserInstrument, currentPage, pagesCount int64,
) *telebot.ReplyMarkup {
	markup := &telebot.ReplyMarkup{}
	var rows []telebot.Row

	rows = b.addPaginationCbkButtons(rows, lang, cbkPortfolioPage, currentPage, pagesCount)

	for _, instrument := range instruments {
		text := b.deps.dictionary.Text(lang, btnPortfolioInstrument, map[string]any{
			"Name":              instrument.Name,
			"Count":             instrument.Count,
			"AvgPrice":          instrument.AvgPrice,
			"PercentDifference": instrument.Last/instrument.AvgPrice*100 - 100,
		})
		callbackData := fmt.Sprintf("%s|%s", cbkInstrument, instrument.Ticker)

		btn := markup.Data(text, callbackData)
		rows = append(rows, telebot.Row{btn})
	}

	markup.Inline(rows...)
	return markup
}

func (b *Bot) instrumentsListByPageKeyboard(
	lang string, instruments []*domain.Instrument, currentPage, pagesCount int64,
) *telebot.ReplyMarkup {
	markup := &telebot.ReplyMarkup{}
	var rows []telebot.Row

	rows = b.addPaginationCbkButtons(rows, lang, cbkInstrumentsPage, currentPage, pagesCount)

	for i := 0; i < len(instruments); i += 2 {
		end := min(i+2, len(instruments))

		var rowBtns []telebot.Btn
		for _, instrument := range instruments[i:end] {
			text := instrument.Name
			callbackData := fmt.Sprintf("%s|%s", cbkInstrument, instrument.Ticker)

			rowBtns = append(rowBtns, markup.Data(text, callbackData))
		}
		rows = append(rows, rowBtns)
	}

	markup.Inline(rows...)
	return markup
}

func (b *Bot) instrumentKeyboard(lang string, instrument *domain.Instrument) *telebot.ReplyMarkup {
	markup := &telebot.ReplyMarkup{}

	btnBuy := telebot.Btn{Text: b.deps.dictionary.Text(lang, btnBuy, map[string]any{
		"Price": instrument.Ask,
	})}
	btnSell := telebot.Btn{Text: b.deps.dictionary.Text(lang, btnSell, map[string]any{
		"Price": instrument.Bid,
	})}
	btnPortfolio := telebot.Btn{Text: b.deps.dictionary.Text(lang, btnPortfolio)}
	btnInstrumentsList := telebot.Btn{Text: b.deps.dictionary.Text(lang, btnInstrumentsList)}
	btnInstrumentsSearch := telebot.Btn{Text: b.deps.dictionary.Text(lang, btnInstrumentsSearch)}
	btnMainMenu := telebot.Btn{Text: b.deps.dictionary.Text(lang, btnMainMenu)}

	rows := []telebot.Row{
		{btnBuy, btnSell},
		{btnPortfolio, btnInstrumentsList},
		{btnMainMenu, btnInstrumentsSearch},
	}

	markup.Reply(rows...)
	markup.ResizeKeyboard = true
	return markup
}

func (b *Bot) paginationKeyboard(lang string, callback string, currentPage, pagesCount int64) *telebot.ReplyMarkup {
	markup := &telebot.ReplyMarkup{}
	var rows []telebot.Row

	rows = b.addPaginationCbkButtons(rows, lang, callback, currentPage, pagesCount)

	markup.Inline(rows...)
	return markup
}
