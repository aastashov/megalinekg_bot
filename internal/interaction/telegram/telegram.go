package telegram

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	telegramBot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"github.com/aastashov/megalinekg_bot/internal/model"
)

type useCase interface {
	UpdateBalance(ctx context.Context, userID int64) error
}

type userStorage interface {
	GetOrCreateByTelegramID(ctx context.Context, userID int64) (*model.User, bool, error)
	Save(ctx context.Context, user *model.User) error
	DeleteByTelegramID(ctx context.Context, userID int64) error
}

type Connector struct {
	logger *slog.Logger
	tgBot  *telegramBot.Bot

	userStorage userStorage
	useCase     useCase

	waitingForLogin map[int64]struct{}
}

func NewConnector(logger *slog.Logger, token string, userStorage userStorage, useCase useCase) *Connector {
	cnt := &Connector{
		logger:          logger.With("component", "telegram"),
		userStorage:     userStorage,
		useCase:         useCase,
		waitingForLogin: make(map[int64]struct{}),
	}

	opts := []telegramBot.Option{
		telegramBot.WithSkipGetMe(),
		telegramBot.WithDefaultHandler(cnt.handler),
	}

	b, _ := telegramBot.New(token, opts...)
	b.RegisterHandler(telegramBot.HandlerTypeMessageText, "/start", telegramBot.MatchTypeExact, cnt.handlerStart)
	b.RegisterHandler(telegramBot.HandlerTypeMessageText, "/about", telegramBot.MatchTypeExact, cnt.handlerAbout)
	b.RegisterHandler(telegramBot.HandlerTypeMessageText, "/delete", telegramBot.MatchTypeExact, cnt.handlerDelete)
	b.RegisterHandler(telegramBot.HandlerTypeMessageText, "/save", telegramBot.MatchTypeExact, cnt.handlerSave)
	b.RegisterHandler(telegramBot.HandlerTypeMessageText, "/balance", telegramBot.MatchTypeExact, cnt.handlerBalance)

	cnt.tgBot = b
	return cnt
}

func (that *Connector) Start(ctx context.Context) {
	that.tgBot.Start(ctx)
}

func (that *Connector) handlerStart(ctx context.Context, bot *telegramBot.Bot, update *models.Update) {
	log := that.logger.With("method", "handlerStart", "user_id", update.Message.From.ID)

	_, created, err := that.userStorage.GetOrCreateByTelegramID(ctx, update.Message.From.ID)
	if err != nil {
		log.Error("Error getting or creating user", "error", err)
		return
	}

	message := "Кажется мы уже знакомы. Напишите /about чтобы узнать больше обо мне."
	if created {
		// If user was created, we should send welcome message
		message = "Привет. Я неофициальный бот MegaLineBalanceBot для отображения баланса. Давай начнем с команды /about."
	}

	_, err = bot.SendMessage(ctx, &telegramBot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   message,
	})

	if err != nil {
		log.Error("Error sending message", "error", err)
		return
	}
}

func (that *Connector) handlerAbout(ctx context.Context, bot *telegramBot.Bot, update *models.Update) {
	log := that.logger.With("method", "handlerAbout", "user_id", update.Message.From.ID)

	const aboutMessage = `*MegaLineBalanceBot* \- ваш помощник для удобного отслеживания баланса в личном кабинете MegaLine\.

✨ Я уважаю вашу конфиденциальность и использую данные только для того, чтобы напоминать вам о балансе\.
🛡️ Храню только ту информацию, которая необходима для работы, и ничего лишнего\.
💻 Мой код открыт для всех и доступен на GitHub: [GitHub](https://github\.com/aastashov/megalinekg_bot)\.
🧹 Если захотите удалить свои данные, просто используйте команду \/delete — всё удалится полностью\.

📥 Чтобы сохранить логин и пароль от личного кабинета, используйте команду \/save\. Эти данные будут храниться только для получения актуального баланса и расчетного периода для напоминания\.

Спасибо, что доверяете мне\! 😊`

	disabled := true
	_, err := bot.SendMessage(ctx, &telegramBot.SendMessageParams{
		ChatID:             update.Message.Chat.ID,
		Text:               aboutMessage,
		ParseMode:          models.ParseModeMarkdown,
		LinkPreviewOptions: &models.LinkPreviewOptions{IsDisabled: &disabled},
	})

	if err != nil {
		log.Error("Error sending message", "error", err)
		return
	}
}

func (that *Connector) handlerDelete(ctx context.Context, bot *telegramBot.Bot, update *models.Update) {
	log := that.logger.With("method", "handlerDelete", "user_id", update.Message.From.ID)

	responseText := "Ваши данные удалены. Для начала работы заново, напишите /start."

	if err := that.userStorage.DeleteByTelegramID(ctx, update.Message.From.ID); err != nil {
		log.Error("Error deleting user", "error", err)
		responseText = "Произошла ошибка при удалении данных. Попробуйте позже."
	}

	_, err := bot.SendMessage(ctx, &telegramBot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   responseText,
	})

	if err != nil {
		log.Error("Error sending message", "error", err, "response_text", responseText)
		return
	}
}

func (that *Connector) handlerSave(ctx context.Context, bot *telegramBot.Bot, update *models.Update) {
	log := that.logger.With("method", "handlerSave", "user_id", update.Message.From.ID)

	_, err := bot.SendMessage(ctx, &telegramBot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   "Введите ваш логин и пароль через пробел",
	})

	if err != nil {
		log.Error("Error sending message", "error", err)
		return
	}

	// Set user as waiting for login
	that.waitingForLogin[update.Message.From.ID] = struct{}{}
}

func (that *Connector) handlerBalance(ctx context.Context, bot *telegramBot.Bot, update *models.Update) {
	log := that.logger.With("method", "handlerBalance", "user_id", update.Message.From.ID)

	if err := that.useCase.UpdateBalance(ctx, update.Message.From.ID); err != nil {
		_, err = bot.SendMessage(ctx, &telegramBot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "Произошла ошибка при получении баланса. Попробуйте позже.",
		})

		if err != nil {
			log.Error("Error sending message", "error", err)
			return
		}

		return
	}

	user, _, err := that.userStorage.GetOrCreateByTelegramID(ctx, update.Message.From.ID)
	if err != nil {
		log.Error("Error getting or creating user", "error", err)
		return
	}

	message := "*Ваш аккаунт MegaLine:*"
	if len(user.Accounts) > 1 {
		message = "*Ваши аккаунты MegaLine:*"
	}

	sep := "\n\n"
	const template = "%s📱 *Номер аккаунта*: %s\n💰 *Баланс*: %v KGS\n📅 *Дата оплаты*: %s\n💳 *Сумма тарифа*: %d KGS"

	for _, account := range user.Accounts {
		message += fmt.Sprintf(template, sep, account.Number, account.Balance, account.BillingTo.Format("02\\-01\\-2006"), account.TariffAmount)
	}

	message = strings.ReplaceAll(message, ".", "\\.")

	_, err = bot.SendMessage(ctx, &telegramBot.SendMessageParams{
		ChatID:    update.Message.Chat.ID,
		Text:      message,
		ParseMode: models.ParseModeMarkdown,
	})

	if err != nil {
		log.Error("Error sending message", "error", err)
		return
	}
}

func (that *Connector) handler(ctx context.Context, bot *telegramBot.Bot, update *models.Update) {
	log := that.logger.With("method", "handler", "user_id", update.Message.From.ID)
	log.Info("Handling message", "text", update.Message.Text)

	if _, ok := that.waitingForLogin[update.Message.From.ID]; ok {
		that.handleWaitingForLogin(ctx, bot, update)
		return
	}
}

func (that *Connector) handleWaitingForLogin(ctx context.Context, bot *telegramBot.Bot, update *models.Update) {
	log := that.logger.With("method", "handleWaitingForLogin", "user_id", update.Message.From.ID)

	// Handle login
	delete(that.waitingForLogin, update.Message.From.ID)

	login, password := "", ""
	// Parse login and password
	parts := strings.Split(strings.TrimSpace(update.Message.Text), " ")
	if len(parts) == 2 {
		login, password = parts[0], parts[1]
	}

	if login == "" || password == "" {
		_, err := bot.SendMessage(ctx, &telegramBot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "Неверный формат. Попробуйте еще раз.",
		})

		if err != nil {
			log.Error("Error sending message", "error", err)
			return
		}
		return
	}

	// Save login and password
	user, _, err := that.userStorage.GetOrCreateByTelegramID(ctx, update.Message.From.ID)
	if err != nil {
		log.Error("Error getting or creating user", "error", err)
		return
	}

	user.AuthUsername = login
	user.AuthPassword = password
	if err = that.userStorage.Save(ctx, user); err != nil {
		log.Error("Error saving user", "error", err)
		return
	}

	_, err = bot.SendMessage(ctx, &telegramBot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   "Данные сохранены. Теперь вы можете получать актуальный баланс.",
	})

	if err != nil {
		log.Error("Error sending message", "error", err)
		return
	}
}
