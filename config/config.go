package config

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Database Database `yaml:"database"`
	MegaLine MegaLine `yaml:"megaline"`
	Telegram Telegram `yaml:"telegram"`
	Log      Log      `yaml:"log"`
}

type Database struct {
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Name     string `yaml:"name"`
}

// GetConnectionURL returns the Connection URL or DSN for the database
func (db Database) GetConnectionURL() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable", db.User, db.Password, db.Host, db.Port, db.Name)
}

// GetConnectionString returns the Connection String or Key-Value Connection String for the database
func (db Database) GetConnectionString() string {
	return fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=disable", db.Host, db.User, db.Password, db.Name, db.Port)
}

type MegaLine struct {
	Timeout time.Duration `yaml:"timeout"`
}

type Telegram struct {
	Token string `yaml:"token"`
}

type Log struct {
	Level string `yaml:"level"`
}

func (lg Log) GetLevel() slog.Level {
	switch lg.Level {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func MustLoad(configPath string) *Config {
	cfg := &Config{}

	if err := cleanenv.ReadConfig(configPath, cfg); err != nil {
		panic(fmt.Errorf("cannot read config: %w", err))
	}

	return cfg
}
