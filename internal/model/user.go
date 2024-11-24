package model

type User struct {
	ID           int    `gorm:"primaryKey"`
	TelegramID   int64  `gorm:"unique"`
	AuthUsername string `gorm:"unique"`
	AuthPassword string
	Session      string
	Accounts     []Account `gorm:"foreignKey:UserID"`
}
