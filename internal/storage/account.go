package storage

import (
	"context"

	"gorm.io/gorm"

	"github.com/aastashov/megalinekg_bot/internal/model"
)

type AccountStorage struct {
	db *gorm.DB
}

func NewAccountStorage(db *gorm.DB) *AccountStorage {
	return &AccountStorage{db: db}
}

func (s *AccountStorage) Save(ctx context.Context, user *model.Account) error {
	return s.db.WithContext(ctx).Save(user).Error
}
