package main

import (
	"context"
	"flag"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/leonid6372/success-bot/internal/common/config"
	"github.com/leonid6372/success-bot/internal/common/repositories/postgres"
	"github.com/leonid6372/success-bot/pkg/log"
	"go.uber.org/zap"
	"gopkg.in/telebot.v4"
)

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "debug.yaml", "bot config path")
	flag.Parse()

	ctx := context.TODO()

	cfg := config.GetConfig(configPath)

	log.Info("bot starting...")

	log.Info("init postgres...")
	pool, err := pgxpool.New(ctx, cfg.GetPostgresURL())
	if err != nil {
		log.Fatal("postgres init failed", zap.Error(err))
	}

	userRepository := postgres.NewUsersRepository(pool)

	b, err := telebot.NewBot(telebot.Settings{
		Token:  cfg.Bot.APIKey,
		Poller: &telebot.LongPoller{Timeout: cfg.Bot.Timeout},
	})
	if err != nil {
		log.Fatal("telebot.NewBot", zap.Error(err))
	}

	go b.Start()

	users, err := userRepository.GetAllUsers(ctx)
	if err != nil {
		log.Fatal("userRepository.GetAllUsers", zap.Error(err))
	}

	for _, user := range users {
		markup := dailyRewardKeyboard()

		if _, err := b.Send(&telebot.User{ID: user.ID},
			"🎁 Повторная ежедневная награда 🎁\nСкорее забирай 👇",
			&telebot.SendOptions{ReplyMarkup: markup, ParseMode: telebot.ModeHTML},
		); err != nil {
			log.Error("failed to send message", zap.String("username", user.Username), zap.Error(err))
		}
	}

	b.Stop()

	log.Info("finish")
}

func dailyRewardKeyboard() *telebot.ReplyMarkup {
	markup := &telebot.ReplyMarkup{}

	btnDailyReward := telebot.Btn{Text: "💰 Забрать награду"}

	rows := []telebot.Row{{btnDailyReward}}

	markup.Reply(rows...)
	markup.ResizeKeyboard = true
	return markup
}
