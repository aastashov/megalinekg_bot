package storage

import (
	"fmt"
	"log/slog"

	slogGorm "github.com/orandin/slog-gorm"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/aastashov/megalinekg_bot/internal/model"
)

type Storage struct {
	DB *gorm.DB
}

func MustNewPostgresDB(logger *slog.Logger, connectionString string) *Storage {
	gormLogger := slogGorm.New(
		slogGorm.WithHandler(logger.Handler()),
		slogGorm.WithTraceAll(),
		slogGorm.SetLogLevel(slogGorm.ErrorLogType, slog.LevelError),
		slogGorm.SetLogLevel(slogGorm.SlowQueryLogType, slog.LevelWarn),
		slogGorm.SetLogLevel(slogGorm.DefaultLogType, slog.LevelInfo),
	)

	db, err := gorm.Open(postgres.Open(connectionString), &gorm.Config{Logger: gormLogger})
	if err != nil {
		panic(fmt.Errorf("open connection: %w", err))
	}

	return &Storage{DB: db}
}

func (s *Storage) MustClose() {
	connection, err := s.DB.DB()
	if err != nil {
		panic(fmt.Errorf("get db connection: %w", err))
	}

	if err = connection.Close(); err != nil {
		panic(fmt.Errorf("close connection: %w", err))
	}
}

func (s *Storage) MustMigration() {
	err := s.DB.AutoMigrate(
		model.User{},
		model.Account{},
	)

	if err != nil {
		panic(fmt.Errorf("migrate models: %w", err))
	}
}
