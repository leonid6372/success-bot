package bot

import (
	"context"
	"errors"
	"strings"

	"github.com/leonid6372/success-bot/internal/boterrs"
	"github.com/leonid6372/success-bot/internal/common/domain"
	"github.com/leonid6372/success-bot/pkg/dictionary"
	"github.com/leonid6372/success-bot/pkg/errs"
	"github.com/leonid6372/success-bot/pkg/log"
	"go.uber.org/zap"
	"gopkg.in/telebot.v4"
)

const (
	ctxContext        = "context"
	ctxUserSubscribed = "subscribed"
)

func (b *Bot) recoveryMiddleware(next telebot.HandlerFunc) telebot.HandlerFunc {
	return func(c telebot.Context) error {
		defer func() {
			if r := recover(); r != nil {
				log.Error("recovered from panic",
					zap.String("username", c.Sender().Username),
					zap.Any("panic", r),
					zap.Stack("stack"),
				)
			}
		}()

		return next(c)
	}
}

func (b *Bot) timeoutMiddleware(next telebot.HandlerFunc) telebot.HandlerFunc {
	return func(c telebot.Context) error {
		log.Info("request", zap.String("username", c.Sender().Username), zap.String("text", c.Text()))

		ctx, cancel := context.WithTimeout(context.Background(), b.cfg.Timeout)
		defer cancel()

		c.Set(ctxContext, ctx)

		return next(c)
	}
}

func (b *Bot) updateUserInfoMiddleware(next telebot.HandlerFunc) telebot.HandlerFunc {
	return func(c telebot.Context) error {
		ctx := c.Get(ctxContext).(context.Context)
		sender := c.Sender()

		user := &domain.User{
			ID:        sender.ID,
			Username:  sender.Username,
			FirstName: sender.FirstName,
			LastName:  sender.LastName,
			IsPremium: sender.IsPremium,
		}

		if err := b.deps.usersRepository.UpdateUserTGData(ctx, user); err != nil {
			log.Error("failed to update user info", zap.String("username", user.Username), zap.Error(err))
		}

		return next(c)
	}
}

func (b *Bot) subscribeMiddleware(next telebot.HandlerFunc) telebot.HandlerFunc {
	return func(c telebot.Context) error {
		message := c.Message()

		if strings.Contains(message.Text, "/start") ||
			strings.Contains(message.Text, cbkLanguage) {
			return next(c)
		}

		subscribed := true
		var err error

		if b.cfg.SubscribeChannelID != 0 {
			subscribed, err = b.checkSubscription(b.cfg.SubscribeChannelID, message.Chat.ID)
			if err != nil {
				return errs.NewStack(err)
			}
		}

		c.Set(ctxUserSubscribed, subscribed)

		if !subscribed {
			return b.notSubscribedHandler(c)
		}

		return next(c)
	}
}

func (b *Bot) checkSubscription(channelID int64, userID int64) (bool, error) {
	chat := &telebot.Chat{ID: channelID}
	user := &telebot.User{ID: userID}

	member, err := b.Telebot.ChatMemberOf(chat, user)
	if err != nil {
		if strings.Contains(err.Error(), "user not found") {
			return false, nil
		}

		return false, errs.NewStack(err)
	}

	return member.Role == telebot.Creator ||
		member.Role == telebot.Administrator ||
		member.Role == telebot.Member, nil
}

func (b *Bot) selectUserMiddleware(next telebot.HandlerFunc) telebot.HandlerFunc {
	return func(c telebot.Context) error {
		ctx := c.Get(ctxContext).(context.Context)
		tgID := c.Sender().ID

		user, ok := b.cache.Get(tgID)
		if ok {
			b.cache.SetDefault(tgID, user)

			return next(c)
		}

		user, err := b.deps.usersRepository.GetUserByID(ctx, tgID)
		if err != nil {
			if errors.Is(err, boterrs.ErrUserNotFound) {
				b.cache.SetDefault(tgID, user)

				return b.startHandler(c)
			}

			return errs.NewStack(err)
		}

		b.cache.SetDefault(tgID, user)

		return next(c)
	}
}

func (b *Bot) defaultErrorMiddleware(next telebot.HandlerFunc) telebot.HandlerFunc {
	return func(c telebot.Context) error {
		if err := next(c); err != nil {
			return b.defaultErrorHandler(c)
		}

		return nil
	}
}

func (b *Bot) defaultErrorHandler(c telebot.Context) error {
	text := b.deps.dictionary.Text(dictionary.DefaultLanguage, msgDefaultError)

	if err := c.Send(text); err != nil {
		return errs.NewStack(err)
	}

	return nil
}
