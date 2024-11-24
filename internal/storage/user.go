package storage

import (
	"context"

	"gorm.io/gorm"

	"github.com/aastashov/megalinekg_bot/internal/model"
)

type UserStorage struct {
	db *gorm.DB
}

func NewUserStorage(db *gorm.DB) *UserStorage {
	return &UserStorage{db: db}
}

func (s *UserStorage) GetOrCreateByTelegramID(ctx context.Context, userID int64) (*model.User, bool, error) {
	var user model.User
	if err := s.db.WithContext(ctx).Where("telegram_id = ?", userID).Preload("Accounts").First(&user).Error; err != nil {
		if err = s.db.WithContext(ctx).Create(&model.User{TelegramID: userID}).Error; err != nil {
			return nil, false, err
		}

		return &user, true, nil
	}

	return &user, false, nil
}

func (s *UserStorage) Save(ctx context.Context, user *model.User) error {
	return s.db.WithContext(ctx).Save(user).Error
}

func (s *UserStorage) DeleteByTelegramID(ctx context.Context, userID int64) error {
	return s.db.WithContext(ctx).Where("telegram_id = ?", userID).Delete(&model.User{}).Error
}
