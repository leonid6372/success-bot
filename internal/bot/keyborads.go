package bot

import (
	"fmt"

	"gopkg.in/telebot.v4"
)

func (b *Bot) mainMenuKeyboard(lang string) *telebot.ReplyMarkup {
	markup := &telebot.ReplyMarkup{}

	btnProfile := telebot.Btn{Text: b.deps.dictionary.Text(lang, btnProfile)}
	btnPortfolio := telebot.Btn{Text: b.deps.dictionary.Text(lang, btnPortfolio)}
	btnTickersList := telebot.Btn{Text: b.deps.dictionary.Text(lang, btnTickersList)}
	btnTickersSearch := telebot.Btn{Text: b.deps.dictionary.Text(lang, btnTickersSearch)}
	btnFAQ := telebot.Btn{Text: b.deps.dictionary.Text(lang, btnFAQ)}
	btnEnterPromocode := telebot.Btn{Text: b.deps.dictionary.Text(lang, btnEnterPromocode)}

	rows := []telebot.Row{
		{btnProfile, btnPortfolio},
		{btnTickersList, btnTickersSearch},
		{btnEnterPromocode, btnFAQ},
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
