package bot

import (
	"fmt"

	"github.com/Ruvad39/go-finam-rest"
	"github.com/leonid6372/success-bot/internal/common/domain"
	"gopkg.in/telebot.v4"
)

func (b *Bot) mainMenuKeyboard(lang string) *telebot.ReplyMarkup {
	markup := &telebot.ReplyMarkup{}

	btnProfile := telebot.Btn{Text: b.deps.dictionary.Text(lang, btnProfile)}
	btnPortfolio := telebot.Btn{Text: b.deps.dictionary.Text(lang, btnPortfolio)}
	btnInstrumentsList := telebot.Btn{Text: b.deps.dictionary.Text(lang, btnInstrumentsList)}
	btnInstrumentsSearch := telebot.Btn{Text: b.deps.dictionary.Text(lang, btnInstrumentsSearch)}
	btnEnterPromocode := telebot.Btn{Text: b.deps.dictionary.Text(lang, btnEnterPromocode)}
	btnFAQ := telebot.Btn{Text: b.deps.dictionary.Text(lang, btnFAQ)}
	btnTopUsers := telebot.Btn{Text: b.deps.dictionary.Text(lang, btnTopUsers)}

	rows := []telebot.Row{
		{btnProfile, btnPortfolio},
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

func (b *Bot) instrumentsListByPageKeyboard(
	lang string, instruments []*domain.Instrument, currentPage, pagesCount int64,
) *telebot.ReplyMarkup {
	markup := &telebot.ReplyMarkup{}
	var rows []telebot.Row

	if currentPage == 1 {
		rows = append(rows, telebot.Row{
			markup.Data(
				b.deps.dictionary.Text(lang, btnNextPage),
				fmt.Sprintf("%s|%d", cbkInstrumentsListPage, currentPage+1),
			),
		})
	}

	if currentPage == pagesCount {
		rows = append(rows, telebot.Row{
			markup.Data(
				b.deps.dictionary.Text(lang, btnPreviousPage),
				fmt.Sprintf("%s|%d", cbkInstrumentsListPage, currentPage-1),
			),
		})
	}

	if currentPage > 1 && currentPage < pagesCount {
		rows = append(rows, telebot.Row{
			markup.Data(
				b.deps.dictionary.Text(lang, btnPreviousPage),
				fmt.Sprintf("%s|%d", cbkInstrumentsListPage, currentPage-1),
			),
			markup.Data(
				b.deps.dictionary.Text(lang, btnNextPage),
				fmt.Sprintf("%s|%d", cbkInstrumentsListPage, currentPage+1),
			),
		})
	}

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

func (b *Bot) instrumentKeyboard(lang string, info *finam.QuoteResponse) *telebot.ReplyMarkup {
	markup := &telebot.ReplyMarkup{}

	btnBuy := telebot.Btn{Text: b.deps.dictionary.Text(lang, btnBuy, map[string]any{
		"Price": info.Quote.Ask,
	})}
	btnSold := telebot.Btn{Text: b.deps.dictionary.Text(lang, btnSold, map[string]any{
		"Price": info.Quote.Bid,
	})}
	btnInstrumentsList := telebot.Btn{Text: b.deps.dictionary.Text(lang, btnInstrumentsList)}
	btnMainMenu := telebot.Btn{Text: b.deps.dictionary.Text(lang, btnMainMenu)}

	rows := []telebot.Row{
		{btnBuy, btnSold},
		{btnInstrumentsList, btnMainMenu},
	}

	markup.Reply(rows...)
	markup.ResizeKeyboard = true
	return markup
}
