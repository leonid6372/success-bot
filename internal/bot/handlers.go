package bot

import (
	"fmt"

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
		return fmt.Errorf("failed to send subscription message: %w", err)
	}

	return nil
}

func (b *Bot) startMsg(c telebot.Context) error {
	user := b.mustUser(c)

	data := map[string]any{
		"ButtonTickersList": b.deps.dictionary.Text(user.LanguageCode, btnTickersList),
	}

	text := b.deps.dictionary.Text(user.LanguageCode, msgStart, data)

	if err := c.Send(text, &telebot.SendOptions{
		ReplyMarkup: b.mainMenuKeyboard(user.LanguageCode),
		ParseMode:   telebot.ModeHTML,
	}); err != nil {
		return fmt.Errorf("failed to send message: %v", err)
	}

	return nil
}

func (b *Bot) setLanguageHandler(c telebot.Context) error {
	defer c.Respond()

	tgID := c.Sender().ID
	args := c.Args()

	if len(args) != 1 {
		return fmt.Errorf("failed to parse data: param language not found")
	}

	langCode := args[0]

	// update user in repository
	if err := b.deps.userRepository.UpdateUserLanguage(b.ctx, tgID, langCode); err != nil {
		return fmt.Errorf("failed to update user language_code in repository: %w", err)
	}

	// update user in cache
	user := b.mustUser(c)
	user.LanguageCode = langCode
	b.cache.SetDefault(user.ID, user)

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
			return fmt.Errorf("failed to get subscribed: %w", err)
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
			return fmt.Errorf("failed to send confirmation: %w", err)
		}
	} else {
		text := b.deps.dictionary.Text(user.LanguageCode, msgSubscriptionFailed)
		if c.Callback() != nil {
			return c.Respond(&telebot.CallbackResponse{Text: text})
		}
	}

	return nil
}
