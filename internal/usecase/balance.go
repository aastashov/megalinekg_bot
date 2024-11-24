package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"

	"github.com/aastashov/megalinekg_bot/internal/model"
)

var (
	dateRe = regexp.MustCompile(`\b(\d{2})\.(\d{2})\.(\d{4})\b`)
)

type userStorage interface {
	GetOrCreateByTelegramID(ctx context.Context, userID int64) (*model.User, bool, error)
	Save(ctx context.Context, user *model.User) error
}

type accountStorage interface {
	Save(ctx context.Context, account *model.Account) error
}

type megaLine interface {
	Login(ctx context.Context, username, password string) ([]byte, string, error)
	GetAccountsDetail(ctx context.Context, session, account string) ([]byte, error)
}

type BalanceUseCase struct {
	logger         *slog.Logger
	userStorage    userStorage
	accountStorage accountStorage
	megaLine       megaLine
}

func NewBalanceUseCase(logger *slog.Logger, userStorage userStorage, accountStorage accountStorage, megaLine megaLine) *BalanceUseCase {
	return &BalanceUseCase{
		logger:         logger.With("use_case", "BalanceUseCase"),
		userStorage:    userStorage,
		accountStorage: accountStorage,
		megaLine:       megaLine,
	}
}

func (uc *BalanceUseCase) UpdateBalance(ctx context.Context, userID int64) error {
	log := uc.logger.With("method", "UpdateBalance", "user_id", userID)

	user, _, err := uc.userStorage.GetOrCreateByTelegramID(ctx, userID)
	if err != nil {
		log.Error("get user by telegram ID", "error", err)
		return fmt.Errorf("get user by telegram ID: %w", err)
	}

	if user.AuthUsername == "" || user.AuthPassword == "" {
		log.Error("user not authorized")
		return errors.New("user not authorized")
	}

	if user.Session == "" {
		body, sessionID, err := uc.megaLine.Login(ctx, user.AuthUsername, user.AuthPassword)
		if err != nil {
			log.Error("login", "error", err, "response.body", string(body))
			return fmt.Errorf("login: %w", err)
		}

		if !strings.Contains(string(body), "Лицевой счет №") {
			log.Error("login failed", "response.body", string(body))
			return errors.New("login failed")
		}

		user.Session = sessionID

		doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
		if err != nil {
			log.Error("parse login response", "error", err)
			return fmt.Errorf("parse login response: %w", err)
		}

		doc.Find(".account_selector").Find("option").Each(func(i int, s *goquery.Selection) {
			user.Accounts = append(user.Accounts, model.Account{Number: strings.TrimSpace(s.Text()), UserID: user.ID})
		})
	}

	if err = uc.userStorage.Save(ctx, user); err != nil {
		log.Error("save user", "error", err)
		return fmt.Errorf("save user: %w", err)
	}

	for _, account := range user.Accounts {
		body, err := uc.megaLine.GetAccountsDetail(ctx, user.Session, account.Number)
		if err != nil {
			log.Error("get account detail", "error", err)
			continue
		}

		body, err = uc.megaLine.GetAccountsDetail(ctx, user.Session, account.Number)
		if err != nil {
			log.Error("get account detail", "error", err, "response.body", string(body))
			continue
		}

		doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
		if err != nil {
			log.Error("parse account detail", "error", err)
			continue
		}

		doc.Find(".account_info").Find(".span100").Each(func(i int, s *goquery.Selection) {
			switch strings.TrimSpace(s.Find(".desc").Text()) {
			case "Баланс":
				s.Find(".value").Each(func(i int, s *goquery.Selection) {
					balance := strings.TrimSpace(s.Text())
					balance = strings.ReplaceAll(balance, " ", "")
					balance = strings.ReplaceAll(balance, "сом", "")
					balance = strings.ReplaceAll(balance, ",", ".")

					balanceFloat, err := strconv.ParseFloat(strings.TrimSpace(balance), 64)
					if err != nil {
						log.Error("Parse balance failed", "error", err, "balance", balance)
						return
					}

					account.Balance = balanceFloat
				})
			case "Расчетный период:":
				s.Find(".value").Each(func(i int, s *goquery.Selection) {
					period := strings.TrimSpace(s.Text())
					matches := dateRe.FindAllStringSubmatch(period, -1)

					if len(matches) != 2 {
						log.Error("Parse period failed", "period", period)
						return
					}

					parsedDate, err := time.Parse("02.01.2006", matches[0][0])
					if err != nil {
						log.Error("Parse period failed", "error", err, "period", period)
						return
					}

					account.BillingFrom = parsedDate

					parsedDate, err = time.Parse("02.01.2006", matches[1][0])
					if err != nil {
						log.Error("Parse period failed", "error", err, "period", period)
						return
					}

					account.BillingTo = parsedDate
				})
			case "Оплата за период:":
				s.Find(".value").Each(func(i int, s *goquery.Selection) {
					payment := strings.TrimSpace(s.Text())
					payment = strings.ReplaceAll(payment, " ", "")
					payment = strings.ReplaceAll(payment, "сом", "")

					paymentInt, err := strconv.Atoi(strings.TrimSpace(payment))
					if err != nil {
						log.Error("Parse payment failed", "error", err, "payment", payment)
						return
					}

					account.TariffAmount = paymentInt
				})
			}
		})

		if err = uc.accountStorage.Save(ctx, &account); err != nil {
			log.Error("save account", "error", err)
			continue
		}
	}

	return nil
}
