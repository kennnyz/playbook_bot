package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/shopspring/decimal"
)

// Перечисление для состояний пользователя
type UserState int

const (
	StateIdle UserState = iota
	StateAwaitingSavePair
	StateAwaitingDealPair
	StateAwaitingAmount
	StateAwaitingBuyPrice
	StateAwaitingSellPrice
)

// Мапа для отслеживания состояний пользователей
var usersStates = make(map[int64]UserState)

var users = make(map[int64]*User)

type User struct {
	Name        string
	UserPairs   map[string]struct{}
	UserDeals   []*Deal
	PendingDeal *Deal
}

type Deal struct {
	Pair          string
	Amount        decimal.Decimal
	BuyPrice      decimal.Decimal
	SellPrice     decimal.Decimal
	Profit        decimal.Decimal
	ProfitPercent decimal.Decimal
	Date          time.Time
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	opts := []bot.Option{
		bot.WithDefaultHandler(defaultHandler),
		bot.WithCallbackQueryDataHandler("/add_pair", bot.MatchTypeExact, addPair_callbackHandler),
		bot.WithCallbackQueryDataHandler("/add_deal", bot.MatchTypeExact, addDeal_callbackHandler),
		bot.WithCallbackQueryDataHandler("/get_history", bot.MatchTypeExact, getHistory_callbackHandler),
		bot.WithMiddlewares(showMessageWithUserName),
	}

	b, err := bot.New("6791125665:AAHKZBXRdFjppkhzmPgmDp4oh2MoInIj3Go", opts...)
	if err != nil {
		panic(err)
	}

	b.RegisterHandler(bot.HandlerTypeMessageText, "/start", bot.MatchTypeExact, startCommand)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/add_deal", bot.MatchTypeExact, addDeal_callbackHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/add_pair", bot.MatchTypeExact, addPair_callbackHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/get_history", bot.MatchTypeExact, getHistory_callbackHandler)

	b.Start(ctx)
}

func defaultHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	chatID := getChatID(update)
	currentState := usersStates[chatID]

	switch currentState {
	case StateAwaitingSavePair:
		log.Println("saved")
		if update.Message.Text == "" {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: chatID,
				Text:   "введите текстом. Или если очень хочется эмодзи :))",
			})
			return
		}
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   "Успешно сохранили вашу пару: " + update.Message.Text + " ✅",
		})
		users[getChatID(update)].UserPairs[strings.ToUpper(update.Message.Text)] = struct{}{}
		usersStates[update.Message.Chat.ID] = StateIdle
		err := showStandardButtons(ctx, b, update)
		if err != nil {
			log.Printf("can't send message to %v, error : %v", chatID, err)
		}
	case StateAwaitingDealPair:
		handlePairSelection(ctx, b, update)
	case StateAwaitingAmount:
		handleAmount(ctx, b, update)
	case StateAwaitingBuyPrice:
		handleBuyPrice(ctx, b, update)
	case StateAwaitingSellPrice:
		handleSellPrice(ctx, b, update)
	default:
		err := showStandardButtons(ctx, b, update)
		if err != nil {
			log.Printf("can't send message to %v, error : %v", chatID, err)
		}
	}
}

func handleAmount(ctx context.Context, b *bot.Bot, update *models.Update) {
	userChatID := getChatID(update)
	if userChatID == 0 {
		return
	}

	user, ok := users[userChatID]
	if !ok {
		log.Println("User not found")
		return
	}

	amount, err := decimal.NewFromString(update.Message.Text)
	if err != nil {
		log.Println("Error parsing amount:", err)
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: userChatID,
			Text:   "невалидный формат",
		})
		return
	}

	user.PendingDeal.Amount = amount
	// Запрашиваем цену продажи
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: userChatID,
		Text:   "Укажите цену покупки:",
	})

	// Устанавливаем состояние пользователя в ожидание ввода количества
	usersStates[userChatID] = StateAwaitingBuyPrice
}

func addPair_callbackHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	usersStates[update.CallbackQuery.Message.Message.Chat.ID] = StateAwaitingSavePair
	log.Printf("update user {%v} state for %v  ", update.CallbackQuery.Message.Message.Chat.Username, StateAwaitingSavePair)

	b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: update.CallbackQuery.ID,
		ShowAlert:       false,
	})

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.CallbackQuery.Message.Message.Chat.ID,
		Text:   "Введите название актива/пары (напр. Amazon, BTC/USDT): ",
	})
}

func addDeal_callbackHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	userChatID := getChatID(update)
	if userChatID == 0 {
		return
	}

	// Получаем пользователя
	user, ok := users[userChatID]
	if !ok {
		log.Println("User not found")
		return
	}
	log.Println("adding deal to ", user.Name)

	// Проверяем, есть ли у пользователя пары
	if len(user.UserPairs) == 0 {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: userChatID,
			Text:   "У вас пока нет ни одной пары для добавления сделки. Вы можете добавить их с помощью кнопки 'Добавить пару'.",
		})
		return
	}

	// Формируем клавиатуру с кнопками пар пользователя
	var keyboard [][]models.InlineKeyboardButton
	for pair := range user.UserPairs {
		keyboard = append(keyboard, []models.InlineKeyboardButton{{Text: pair, CallbackData: pair}})
	}

	// Отправляем сообщение с клавиатурой
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      userChatID,
		Text:        "Выберите пару для добавления сделки:",
		ReplyMarkup: models.InlineKeyboardMarkup{InlineKeyboard: keyboard},
	})
	usersStates[userChatID] = StateAwaitingDealPair
}

func handlePairSelection(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.CallbackQuery == nil {
		return
	}

	userChatID := getChatID(update)
	if userChatID == 0 {
		return
	}

	// Получаем пользователя
	user, ok := users[userChatID]
	if !ok {
		log.Println("User not found")
		return
	}

	// Получаем выбранную пользователем пару
	selectedPair := update.CallbackQuery.Data

	// Устанавливаем выбранную пару во временную сделку
	user.PendingDeal = &Deal{Pair: selectedPair}

	// Запрашиваем цену покупки
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: userChatID,
		Text:   "Укажите количесвто:",
	})

	// Устанавливаем состояние пользователя в ожидание ввода количества
	usersStates[userChatID] = StateAwaitingAmount
}

func handleBuyPrice(ctx context.Context, b *bot.Bot, update *models.Update) {
	userChatID := getChatID(update)
	if userChatID == 0 {
		return
	}

	// Получаем пользователя
	user, ok := users[userChatID]
	if !ok {
		log.Println("User not found")
		return
	}

	buyPrice, err := decimal.NewFromString(update.Message.Text)
	if err != nil {
		log.Println("Error parsing buy price:", err)
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: userChatID,
			Text:   "невалидный формат",
		})
		return
	}

	user.PendingDeal.BuyPrice = buyPrice

	// Запрашиваем цену продажи
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: userChatID,
		Text:   "Укажите цену продажи:",
	})
	// Устанавливаем состояние пользователя в ожидание ввода цены продажи
	usersStates[userChatID] = StateAwaitingSellPrice
}

func handleSellPrice(ctx context.Context, b *bot.Bot, update *models.Update) {
	userChatID := getChatID(update)
	if userChatID == 0 {
		return
	}

	// Получаем пользователя
	user, ok := users[userChatID]
	if !ok {
		log.Println("User not found")
		return
	}

	// Получаем цену покупки
	sellPrice, err := decimal.NewFromString(update.Message.Text)
	if err != nil {
		log.Println("Error parsing buy price:", err)
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: userChatID,
			Text:   "невалидный формат",
		})
		return
	}

	// Записываем цену покупки в структуру сделки
	user.PendingDeal.SellPrice = sellPrice

	// Устанавливаем состояние пользователя в ожидание ввода даты
	completeDeal(ctx, b, userChatID, user)
	showStandardButtons(ctx, b, update)
	usersStates[userChatID] = StateIdle
}

func completeDeal(ctx context.Context, b *bot.Bot, userChatID int64, user *User) {
	// Профит = (цена продажи - цена покупки) * количество
	user.PendingDeal.Profit = user.PendingDeal.SellPrice.Sub(user.PendingDeal.BuyPrice).Mul(user.PendingDeal.Amount)
	// Процент прибыли = (цена продажи - цена покупки) / цена покупки * 100
	user.PendingDeal.ProfitPercent = user.PendingDeal.SellPrice.Sub(user.PendingDeal.BuyPrice).Div(user.PendingDeal.BuyPrice).Mul(decimal.NewFromInt(100))

	user.PendingDeal.Date = time.Now()

	user.UserDeals = append(user.UserDeals, user.PendingDeal)
	dealText := "<b> Сделка успешно добавлена 🎉 Ваша сделка:</b>\n" +
		"<b>Покупка:</b> " + user.PendingDeal.BuyPrice.String() + "\n" +
		"<b>Количество:</b> " + user.PendingDeal.Amount.String() + "\n" +
		"<b>Продажа:</b> " + user.PendingDeal.SellPrice.String() + "\n" +
		"<b>Прибыль:</b> " + user.PendingDeal.Profit.String() + "$\n" +
		"<b>Процент прибыли:</b> " + user.PendingDeal.ProfitPercent.String() + "%\n"
	fmt.Printf("%s deal: \nbuy price %v\nsell price %v \nprofit %v\nprofit percentage %v\n", user.Name, user.PendingDeal.BuyPrice, user.PendingDeal.SellPrice, user.PendingDeal.Profit, user.PendingDeal.ProfitPercent)
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    userChatID,
		Text:      dealText,
		ParseMode: "HTML",
	})
	user.PendingDeal = nil
}

func getHistory_callbackHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	chatID := getChatID(update)
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chatID,
		Text:   "пока не реализованная функция 🙈",
	})
	showStandardButtons(ctx, b, update)
}

func showMessageWithUserName(next bot.HandlerFunc) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		chatID := getChatID(update)
		if _, ok := users[chatID]; !ok {
			var deals []*Deal
			var uName string
			if update.Message != nil {
				uName = update.Message.Chat.Username
			} else if update.CallbackQuery != nil {
				uName = update.CallbackQuery.Message.Message.Chat.Username
			}
			users[chatID] = &User{
				Name:      uName,
				UserPairs: make(map[string]struct{}),
				UserDeals: deals,
			}

			log.Println("Saved new user: ", uName)
		}

		if update.Message != nil {
			log.Printf("%s say: %s", update.Message.From.Username, update.Message.Text)
		}
		next(ctx, b, update)
	}
}

func startCommand(ctx context.Context, b *bot.Bot, update *models.Update) {
	message := "Привет 👋\nЭто Бот с помощью которого можно вести учет ваших сделок📖\n\nПоддерживаемые команды:\n/add_deal - добавить новую сделку\n/add\n/get_history - получить историю сделок"
	_, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   message,
	})
	if err != nil {
		log.Printf("can't send message to %s, error : %v", update.Message.Chat.Username, err)
	}

	if _, ok := users[update.Message.Chat.ID]; !ok {
		var deals []*Deal
		users[update.Message.Chat.ID] = &User{
			Name:      update.Message.Chat.Username,
			UserPairs: make(map[string]struct{}),
			UserDeals: deals,
		}

		log.Println("Saved new user: ", update.Message.Chat.Username)
	}

	err = showStandardButtons(ctx, b, update)
	if err != nil {
		log.Printf("can't send message to %v, error : %v", update.Message.Chat.ID, err)
	}
}

func showStandardButtons(ctx context.Context, b *bot.Bot, update *models.Update) error {
	kb := &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{Text: "Добавить сделку", CallbackData: "/add_deal"},
				{Text: "Добавить пару", CallbackData: "/add_pair"},
			},
			{
				{Text: "История сделок", CallbackData: "/get_history"},
			},
		},
	}

	_, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      getChatID(update),
		Text:        "Выберите действие",
		ReplyMarkup: kb,
	})
	if err != nil {
		log.Printf("can't send message to %s, error : %v", update.Message.Chat.Username, err)
		return err
	}

	return nil
}

func getChatID(update *models.Update) int64 {
	if update.Message != nil {
		return update.Message.Chat.ID
	} else if update.CallbackQuery != nil {
		return update.CallbackQuery.Message.Message.Chat.ID
	}

	return 0
}

func truncateFloat(f float64, precision int) float64 {
	multiplier := math.Pow(10, float64(precision))
	truncated := math.Trunc(f*multiplier) / multiplier
	return truncated
}
