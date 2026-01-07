package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/leonid6372/success-bot/internal/bot"
	"github.com/leonid6372/success-bot/internal/common/clients/finam"
	"github.com/leonid6372/success-bot/internal/common/config"
	"github.com/leonid6372/success-bot/internal/common/repositories/postgres"
	"github.com/leonid6372/success-bot/pkg/dictionary"
	"github.com/leonid6372/success-bot/pkg/goosemigrate"
	"github.com/leonid6372/success-bot/pkg/log"
	"go.uber.org/zap"
)

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "prod.yaml", "bot config path")
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())

	cfg := config.GetConfig(configPath)

	log.Info("bot starting...")

	log.Info("init dictionary...")
	dictionary, err := dictionary.New()
	if err != nil {
		log.Fatal("dictionary init failed", zap.Error(err))
	}

	log.Info("init postgres...")
	pool, err := pgxpool.New(ctx, cfg.GetPostgresURL())
	if err != nil {
		log.Fatal("postgres init failed", zap.Error(err))
	}

	if err := goosemigrate.NewMigrator(cfg.GetPostgresURL(), "migrations", cfg.Postgres.Schema).Up(); err != nil {
		log.Fatal("migrations up failed", zap.Error(err))
	}

	userRepository := postgres.NewUsersRepository(pool)
	instrumentsRepository := postgres.NewInstrumentsRepository(pool)
	promocodesRepository := postgres.NewPromocodesRepository(pool)
	operationsRepository := postgres.NewOperationsRepository(pool)
	portfoliosRepository := postgres.NewPortfolioRepository(pool)

	log.Info("init finam...")
	finam, err := finam.NewClient(ctx, cfg.Finam.Token, cfg.Finam.AccountID)
	if err != nil {
		log.Fatal("finam init failed", zap.Error(err))
	}

	log.Info("init telebot...")
	bot, err := bot.New(ctx,
		&cfg.Bot,
		finam,
		dictionary,
		userRepository,
		instrumentsRepository,
		promocodesRepository,
		operationsRepository,
		portfoliosRepository,
	)
	if err != nil {
		log.Fatal("bot starting failed", zap.Error(err))
	}

	go func() {
		bot.Start()
	}()

	log.Info("bot starting complete")

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	<-done
	log.Info("bot shutting down...")

	pool.Close()
	bot.Stop()

	if err := log.Sync(); err != nil {
		log.Error("log sync failed", zap.Error(err))
	}

	cancel()

	log.Info("bot shut down complete")
}
