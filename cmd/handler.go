package main

import (
	"context"
	"fmt"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/shopspring/decimal"
	"log"
	"strings"
	"time"
)

func defaultHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	chatID := getChatID(update)
	currentState := usersStates[chatID]

	switch currentState {
	case StateAwaitingSavePair:
		handleSavePair(ctx, b, update)
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

func handleSavePair(ctx context.Context, b *bot.Bot, update *models.Update) {
	chatID := getChatID(update)
	if update.Message.Text == "" {
		if _, err := b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   "Введите текстом. Или если очень хочется эмодзи :))",
		}); err != nil {
			log.Println("error sending msg ", getChatID(update), err)
			return
		}
		return
	}

	user := users[chatID]
	if user == nil {
		log.Println("User not found")
		// TODO create
		return
	}

	user.UserPairs[strings.ToUpper(update.Message.Text)] = struct{}{}
	usersStates[chatID] = StateIdle

	if _, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chatID,
		Text:   "Успешно сохранили вашу пару: " + update.Message.Text + " ✅",
	}); err != nil {
		log.Println("error sending msg ", getChatID(update), err)
		return
	}
	log.Printf("%s saved pair %s", user.Name, strings.ToUpper(update.Message.Text))

	err := showStandardButtons(ctx, b, update)
	if err != nil {
		log.Printf("can't send message to %v, error: %v", chatID, err)
	}
}

func handleAmount(ctx context.Context, b *bot.Bot, update *models.Update) {
	chatID := getChatID(update)
	if chatID == 0 {
		return
	}

	user, ok := users[chatID]
	if !ok {
		log.Println("User not found")
		return
	}

	amount, err := decimal.NewFromString(update.Message.Text)
	if err != nil {
		log.Println("Error parsing amount:", err)
		if _, err := b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   "невалидный формат",
		}); err != nil {
			log.Printf("can't send message to %v, error: %v", chatID, err)
		}
		return
	}

	user.PendingDeal.Amount = amount
	if _, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chatID,
		Text:   "Укажите цену покупки:",
	}); err != nil {
		log.Printf("can't send message to %v, error: %v", chatID, err)
		return
	}

	usersStates[chatID] = StateAwaitingBuyPrice
}

func addPairCallbackHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	usersStates[getChatID(update)] = StateAwaitingSavePair
	log.Printf("update user {%v} state for %v  ", update.CallbackQuery.Message.Message.Chat.Username, StateAwaitingSavePair)

	if _, err := b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: update.CallbackQuery.ID,
		ShowAlert:       false,
	}); err != nil {
		log.Println("error sending msg ", getChatID(update), err)
		return
	}

	if _, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.CallbackQuery.Message.Message.Chat.ID,
		Text:   "Введите название актива/пары (напр. Amazon, BTC/USDT): ",
	}); err != nil {
		log.Println("error sending msg ", getChatID(update), err)
		return
	}
}

func addDealCallbackHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
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

	if len(user.UserPairs) == 0 {
		if _, err := b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: userChatID,
			Text:   "У вас пока нет ни одной пары для добавления сделки. Вы можете добавить их с помощью кнопки 'Добавить пару'.",
		}); err != nil {
			log.Println("error sending msg ", getChatID(update), err)
			return
		}

		return
	}

	var keyboard [][]models.InlineKeyboardButton
	for pair := range user.UserPairs {
		keyboard = append(keyboard, []models.InlineKeyboardButton{{Text: pair, CallbackData: pair}})
	}

	if _, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      userChatID,
		Text:        "Выберите пару для добавления сделки:",
		ReplyMarkup: models.InlineKeyboardMarkup{InlineKeyboard: keyboard},
	}); err != nil {
		log.Println("error sending msg ", getChatID(update), err)
		return
	}

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
	if _, ok := user.UserPairs[update.CallbackQuery.Data]; !ok {
		if _, err := b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: userChatID,
			Text:   "no such pair",
		}); err != nil {
			log.Println("error sending msg ", getChatID(update), err)
			return
		}
	}
	user.PendingDeal = &Deal{Pair: update.CallbackQuery.Data}

	if _, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: userChatID,
		Text:   "Укажите количесвто:",
	}); err != nil {
		log.Println("error sending msg ", getChatID(update), err)
		return
	}

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

	buyPrice, err := decimal.NewFromString(strings.ReplaceAll(update.Message.Text, ",", "."))
	if err != nil {
		log.Println("Error parsing buy price:", err)
		if _, err := b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: userChatID,
			Text:   "невалидный формат",
		}); err != nil {
			log.Println("error sending msg ", getChatID(update), err)
			return
		}

		return
	}
	user.PendingDeal.BuyPrice = buyPrice

	if _, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: userChatID,
		Text:   "Укажите цену продажи:",
	}); err != nil {
		log.Println("error sending msg ", getChatID(update), err)
		return
	}

	usersStates[userChatID] = StateAwaitingSellPrice
}

func handleSellPrice(ctx context.Context, b *bot.Bot, update *models.Update) {
	chatID := getChatID(update)
	if chatID == 0 {
		return
	}

	// Получаем пользователя
	user, ok := users[chatID]
	if !ok {
		log.Println("User not found")
		return
	}

	// Получаем цену покупки
	sellPrice, err := decimal.NewFromString(strings.ReplaceAll(update.Message.Text, ",", "."))
	if err != nil {
		log.Println("Error parsing buy price:", err)
		if _, err := b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   "невалидный формат",
		}); err != nil {
			log.Printf("can't send message to %v, error: %v", chatID, err)
			return
		}
		return
	}
	user.PendingDeal.SellPrice = sellPrice

	completeDeal(ctx, b, chatID, user)

	if err := showStandardButtons(ctx, b, update); err != nil {
		log.Printf("can't send message to %v, error: %v", chatID, err)
		return
	}

	usersStates[chatID] = StateIdle
}

func completeDeal(ctx context.Context, b *bot.Bot, chatID int64, user *User) {
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

	if _, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    chatID,
		Text:      dealText,
		ParseMode: "HTML",
	}); err != nil {
		log.Printf("can't send message to %v, error: %v", chatID, err)
	}
	user.PendingDeal = nil
}

func getHistoryCallbackHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	chatID := getChatID(update)
	if _, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chatID,
		Text:   "пока не реализованная функция 🙈",
	}); err != nil {
		log.Printf("can't send message to %v, error: %v", chatID, err)
		return
	}

	if err := showStandardButtons(ctx, b, update); err != nil {
		log.Printf("can't send message to %v, error: %v", chatID, err)
		return
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

func getChatID(update *models.Update) int64 {
	if update.Message != nil {
		return update.Message.Chat.ID
	} else if update.CallbackQuery != nil {
		return update.CallbackQuery.Message.Message.Chat.ID
	}

	return 0
}
