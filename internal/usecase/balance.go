package usecase

import (
	"context"
	"errors"
	"fmt"
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
	userStorage    userStorage
	accountStorage accountStorage
	megaLine       megaLine
}

func NewBalanceUseCase(userStorage userStorage, accountStorage accountStorage, megaLine megaLine) *BalanceUseCase {
	return &BalanceUseCase{
		userStorage:    userStorage,
		accountStorage: accountStorage,
		megaLine:       megaLine,
	}
}

func (uc *BalanceUseCase) UpdateBalance(ctx context.Context, userID int64) error {
	user, _, err := uc.userStorage.GetOrCreateByTelegramID(ctx, userID)
	if err != nil {
		return fmt.Errorf("get user by telegram ID: %w", err)
	}

	if user.AuthUsername == "" || user.AuthPassword == "" {
		return errors.New("user not authorized")
	}

	if user.Session == "" {
		body, sessionID, err := uc.megaLine.Login(ctx, user.AuthUsername, user.AuthPassword)
		if err != nil {
			return fmt.Errorf("login: %w", err)
		}

		if !strings.Contains(string(body), "Лицевой счет №") {
			return errors.New("login failed")
		}

		user.Session = sessionID

		doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
		if err != nil {
			return fmt.Errorf("parse login response: %w", err)
		}

		doc.Find(".account_selector").Find("option").Each(func(i int, s *goquery.Selection) {
			user.Accounts = append(user.Accounts, model.Account{Number: strings.TrimSpace(s.Text()), UserID: user.ID})
		})
	}

	if err = uc.userStorage.Save(ctx, user); err != nil {
		return fmt.Errorf("save user: %w", err)
	}

	for _, account := range user.Accounts {
		body, err := uc.megaLine.GetAccountsDetail(ctx, user.Session, account.Number)
		if err != nil {
			fmt.Println("Get account detail failed:", err)
			continue
		}

		body, err = uc.megaLine.GetAccountsDetail(ctx, user.Session, account.Number)
		if err != nil {
			fmt.Println("Get account detail failed:", err)
			continue
		}

		doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
		if err != nil {
			fmt.Println("Parse account detail failed:", err)
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
						fmt.Println("Parse balance failed:", err)
						return
					}

					account.Balance = balanceFloat
				})
			case "Расчетный период:":
				s.Find(".value").Each(func(i int, s *goquery.Selection) {
					period := strings.TrimSpace(s.Text())
					matches := dateRe.FindAllStringSubmatch(period, -1)

					if len(matches) != 2 {
						fmt.Println("Parse period failed")
						return
					}

					parsedDate, err := time.Parse("02.01.2006", matches[0][0])
					if err != nil {
						fmt.Println("Parse period failed:", err)
						return
					}

					account.BillingFrom = parsedDate

					parsedDate, err = time.Parse("02.01.2006", matches[1][0])
					if err != nil {
						fmt.Println("Parse period failed:", err)
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
						fmt.Println("Parse payment failed:", err)
						return
					}

					account.TariffAmount = paymentInt
				})
			}
		})

		if err = uc.accountStorage.Save(ctx, &account); err != nil {
			fmt.Println("Save account failed:", err)
			continue
		}
	}

	return nil
}
