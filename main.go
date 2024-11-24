package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	"github.com/aastashov/megalinekg_bot/config"
	"github.com/aastashov/megalinekg_bot/internal/interaction/megaline"
	"github.com/aastashov/megalinekg_bot/internal/interaction/telegram"
	"github.com/aastashov/megalinekg_bot/internal/storage"
	"github.com/aastashov/megalinekg_bot/internal/usecase"
)

const defaultRequestTimeout = 5 * time.Second

func main() {
	baseDir, err := os.Getwd()
	if err != nil {
		panic(fmt.Errorf("cannot get current working directory: %w", err))
	}

	cnf := config.MustLoad(filepath.Join(baseDir, "./config.yml"))
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cnf.Log.GetLevel()}))

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	// Initialize database
	connection, err := storage.NewPostgresDB(logger, cnf.Log.GetLevel(), cnf.Database.GetConnectionString())
	if err != nil {
		panic(fmt.Errorf("open db: %w", err))
	}

	defer func() {
		db, err := connection.DB()
		if err != nil {
			logger.Error("close db", "error", err)
			return
		}

		if err = db.Close(); err != nil {
			logger.Error("close db", "error", err)
			return
		}
	}()

	if err = storage.Migration(connection); err != nil {
		panic(fmt.Errorf("migrate db: %w", err))
	}

	// Initialize storage
	userStorage := storage.NewUserStorage(connection)
	accountStorage := storage.NewAccountStorage(connection)

	// Initialize interaction with MegaLine
	megaLineConnector := megaline.NewConnector(http.Client{Timeout: defaultRequestTimeout})

	// Initialize use case
	balanceUseCase := usecase.NewBalanceUseCase(userStorage, accountStorage, megaLineConnector)

	// Initialize interaction with Telegram
	telegramConnector := telegram.NewConnector(logger, cnf.Telegram.Token, userStorage, balanceUseCase)
	telegramConnector.Start(ctx)
}
