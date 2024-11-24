package storage

import (
	"fmt"
	"log/slog"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	slogGorm "github.com/orandin/slog-gorm"

	"github.com/aastashov/megalinekg_bot/internal/model"
)

func NewPostgresDB(logger *slog.Logger, logLevel slog.Level, connectionString string) (*gorm.DB, error) {
	gormLogger := slogGorm.New(
		slogGorm.WithHandler(logger.Handler()),
		slogGorm.WithTraceAll(),
		slogGorm.SetLogLevel(slogGorm.ErrorLogType, slog.LevelError),
		slogGorm.SetLogLevel(slogGorm.SlowQueryLogType, slog.LevelWarn),
		slogGorm.SetLogLevel(slogGorm.DefaultLogType, logLevel),
	)

	db, err := gorm.Open(postgres.Open(connectionString), &gorm.Config{Logger: gormLogger})
	if err != nil {
		return nil, fmt.Errorf("open connection: %w", err)
	}

	return db, nil
}

func Migration(db *gorm.DB) error {
	err := db.AutoMigrate(
		model.User{},
		model.Account{},
	)

	if err != nil {
		return fmt.Errorf("migrate models: %w", err)
	}

	return nil
}
