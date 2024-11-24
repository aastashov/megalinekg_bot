package model

import "time"

type Account struct {
	ID           int `gorm:"primaryKey"`
	UserID       int
	Number       string `gorm:"unique"`
	BillingFrom  time.Time
	BillingTo    time.Time
	TariffAmount int
	Balance      float64
}
