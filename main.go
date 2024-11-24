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
	connection := storage.MustNewPostgresDB(logger, cnf.Log.GetLevel(), cnf.Database.GetConnectionString())
	defer connection.MustClose()

	connection.MustMigration()

	// Initialize storage
	userStorage := storage.NewUserStorage(connection.DB)
	accountStorage := storage.NewAccountStorage(connection.DB)

	// Initialize interaction with MegaLine
	megaLineConnector := megaline.NewConnector(http.Client{Timeout: cnf.MegaLine.Timeout * time.Second})

	// Initialize use case
	balanceUseCase := usecase.NewBalanceUseCase(logger, userStorage, accountStorage, megaLineConnector)

	// Initialize interaction with Telegram
	telegramConnector := telegram.NewConnector(logger, cnf.Telegram.Token, userStorage, balanceUseCase)
	telegramConnector.Start(ctx)
}
