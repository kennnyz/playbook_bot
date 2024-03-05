package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/shopspring/decimal"
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

	if err := validatePair(update.Message.Text); err != nil {
		if _, err := b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   err.Error(),
		}); err != nil {
			log.Println("error sending msg ", getChatID(update), err)
			return
		}
		return
	}

	if err := Repository.savePair(chatID, strings.ToUpper(update.Message.Text)); err != nil {
		if _, err := b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   "Ошибка сохранение пары",
		}); err != nil {
			log.Println("error sending msg ", getChatID(update), err)
			return
		}
		return
	}
	usersStates[chatID] = StateIdle

	if _, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chatID,
		Text:   "Успешно сохранили вашу пару: " + update.Message.Text + " ✅",
	}); err != nil {
		log.Println("error sending msg ", getChatID(update), err)
		return
	}

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

	amount, err := validatePrice(update.Message.Text)
	if err != nil {
		if _, err := b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   err.Error(),
		}); err != nil {
			log.Println("error sending msg ", getChatID(update), err)
			return
		}
		return
	}

	usersPendingDeal[chatID].Amount = amount
	if _, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chatID,
		Text:   "Укажите цену покупки:",
	}); err != nil {
		log.Printf("can't send message to %v, error: %v", chatID, err)
		return
	}

	usersStates[chatID] = StateAwaitingBuyPrice
	log.Printf("update user %v state for %v  ", chatID, StateAwaitingBuyPrice)
}

func addPairCallbackHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.CallbackQuery != nil {
		if _, err := b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
			CallbackQueryID: update.CallbackQuery.ID,
			ShowAlert:       false,
		}); err != nil {
			log.Println("error sending msg ", getChatID(update), err)
			return
		}
	}

	chatID := getChatID(update)
	if chatID == 0 {
		return
	}

	if _, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chatID,
		Text:   "Введите название актива/пары (напр. Amazon, BTC/USD): ",
	}); err != nil {
		log.Println("error sending msg ", getChatID(update), err)
		return
	}

	usersStates[getChatID(update)] = StateAwaitingSavePair
	log.Printf("update user %v state for %v  ", chatID, StateAwaitingSavePair)
}

func addDealCallbackHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	chatID := getChatID(update)
	if chatID == 0 {
		return
	}

	// Получаем пользователя
	userPairs, err := Repository.getPairs(chatID)
	if err != nil {
		log.Println("Error getting pairs: ", err)
		return
	}

	log.Println("adding deal to ", chatID)

	if len(userPairs) == 0 {
		if _, err := b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   "У вас пока нет ни одной пары для добавления сделки. Вы можете добавить их с помощью кнопки 'Добавить пару'.",
		}); err != nil {
			log.Println("error sending msg ", getChatID(update), err)
			return
		}

		return
	}

	var keyboard [][]models.InlineKeyboardButton
	for _, pair := range userPairs {
		keyboard = append(keyboard, []models.InlineKeyboardButton{{Text: pair, CallbackData: pair}})
	}

	if _, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      chatID,
		Text:        "Выберите пару для добавления сделки:",
		ReplyMarkup: models.InlineKeyboardMarkup{InlineKeyboard: keyboard},
	}); err != nil {
		log.Println("error sending msg ", getChatID(update), err)
		return
	}

	usersStates[chatID] = StateAwaitingDealPair
	log.Printf("update user %v state for %v  ", chatID, StateAwaitingDealPair)
}

func handlePairSelection(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.CallbackQuery == nil {
		return
	}

	chatID := getChatID(update)
	if chatID == 0 {
		return
	}

	ok, err := Repository.getPair(chatID, update.CallbackQuery.Data)
	if err != nil {
		log.Println("Error getting pair: ", err)
		return
	}
	if !ok {
		if _, err := b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   "no such pair",
		}); err != nil {
			log.Println("error sending msg ", getChatID(update), err)
			return
		}
	}

	usersPendingDeal[chatID] = &Deal{Pair: update.CallbackQuery.Data}

	if _, err := b.SendMessage(ctx, &bot.SendMessageParams{ChatID: chatID, Text: "Укажите количесвто:"}); err != nil {
		log.Println("error sending msg ", getChatID(update), err)
		return
	}

	usersStates[chatID] = StateAwaitingAmount
	log.Printf("update user %v state for %v  ", chatID, StateAwaitingAmount)
}

func handleBuyPrice(ctx context.Context, b *bot.Bot, update *models.Update) {
	chatID := getChatID(update)
	if chatID == 0 {
		return
	}

	buyPrice, err := validatePrice(update.Message.Text)
	if err != nil {
		if _, err := b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   err.Error(),
		}); err != nil {
			log.Println("error sending msg ", getChatID(update), err)
			return
		}
		return
	}
	usersPendingDeal[chatID].BuyPrice = buyPrice

	if _, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chatID,
		Text:   "Укажите цену продажи:",
	}); err != nil {
		log.Println("error sending msg ", getChatID(update), err)
		return
	}

	usersStates[chatID] = StateAwaitingSellPrice
	log.Printf("update user %v state for %v  ", chatID, StateAwaitingSellPrice)
}

func handleSellPrice(ctx context.Context, b *bot.Bot, update *models.Update) {
	chatID := getChatID(update)
	if chatID == 0 {
		return
	}

	// Получаем цену покупки
	sellPrice, err := validatePrice(update.Message.Text)
	if err != nil {
		if _, err := b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   err.Error(),
		}); err != nil {
			log.Println("error sending msg ", getChatID(update), err)
			return
		}
		return
	}
	usersPendingDeal[chatID].SellPrice = sellPrice

	completeDeal(ctx, b, chatID, usersPendingDeal[chatID])

	if err := showStandardButtons(ctx, b, update); err != nil {
		log.Printf("can't send message to %v, error: %v", chatID, err)
		return
	}

	usersStates[chatID] = StateIdle
	log.Printf("update user %v state for %v  ", chatID, StateIdle)
}

func completeDeal(ctx context.Context, b *bot.Bot, chatID int64, PendingDeal *Deal) {
	// Профит = (цена продажи - цена покупки) * количество
	PendingDeal.Profit = PendingDeal.SellPrice.Sub(PendingDeal.BuyPrice).Mul(PendingDeal.Amount)
	// Процент прибыли = (цена продажи - цена покупки) / цена покупки * 100
	PendingDeal.ProfitPercent = PendingDeal.SellPrice.Sub(PendingDeal.BuyPrice).Div(PendingDeal.BuyPrice).Mul(decimal.NewFromInt(100))

	PendingDeal.Date = time.Now()

	err := Repository.saveDeal(PendingDeal, chatID)
	if err != nil {
		log.Println("Error saving deal: ", err)
		if _, err := b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:    chatID,
			Text:      "Ошибка сохранения сделки",
			ParseMode: "HTML",
		}); err != nil {
			log.Printf("can't send message to %v, error: %v", chatID, err)
		}

		return
	}

	dealText := "<b> Сделка успешно добавлена 🎉 Ваша сделка:</b>\n" +
		"<b>Пара:</b> " + PendingDeal.Pair + "\n" +
		"<b>Количество:</b> " + PendingDeal.Amount.String() + "\n" +
		"<b>Покупка:</b> " + PendingDeal.BuyPrice.String() + "\n" +
		"<b>Продажа:</b> " + PendingDeal.SellPrice.String() + "\n" +
		"<b>Прибыль:</b> " + PendingDeal.Profit.String() + "$\n" +
		"<b>Процент прибыли:</b> " + PendingDeal.ProfitPercent.Truncate(3).String() + "%\n"
	fmt.Printf("%v deal: \nbuy price %v\nsell price %v \nprofit %v\nprofit percentage %v\n", chatID, PendingDeal.BuyPrice, PendingDeal.SellPrice, PendingDeal.Profit, PendingDeal.ProfitPercent)

	if _, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    chatID,
		Text:      dealText,
		ParseMode: "HTML",
	}); err != nil {
		log.Printf("can't send message to %v, error: %v", chatID, err)
	}

	usersPendingDeal[chatID] = nil
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
		log.Printf("can't send message to %s, error : %v", getUserName(update), err)
		return err
	}

	return nil
}

func startCommand(ctx context.Context, b *bot.Bot, update *models.Update) {
	message := "Привет 👋\nЭто Бот с помощью которого можно вести учет ваших сделок📖\n\nПоддерживаемые команды:\n/add_deal - добавить новую сделку\n/add_pair - добавить актив/пару\n/get_history - получить историю сделок"
	if _, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   message,
	}); err != nil {
		log.Printf("can't send message to %s, error : %v", getUserName(update), err)
	}

	if user, err := Repository.getUser(getChatID(update)); errors.Is(err, sql.ErrNoRows) || user == nil {
		// Пользователя нет в базе, создаем нового
		err = Repository.saveUser(&User{
			Name:   getUserName(update),
			ChatID: getChatID(update),
		})

		log.Println("Saved new user: ", getUserName(update))
	}

	if err := showStandardButtons(ctx, b, update); err != nil {
		log.Printf("can't send message to %v, error : %v", update.Message.Chat.ID, err)
	}
}

func showMessageWithUserName(next bot.HandlerFunc) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		chatID := getChatID(update)
		if user, err := Repository.getUser(chatID); errors.Is(err, sql.ErrNoRows) || user == nil {
			err = Repository.saveUser(&User{
				Name:   getUserName(update),
				ChatID: chatID,
			})

			log.Println("Saved new user: ", getUserName(update))
		}

		user, err := Repository.getUser(chatID)
		if err != nil || user == nil {
			log.Println("Error getting user: ", err)
			return
		}

		if update.Message != nil {
			log.Printf("%s say: %s", user.Name, update.Message.Text)
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

func getUserName(update *models.Update) string {
	if update.Message != nil {
		return update.Message.Chat.Username
	} else if update.CallbackQuery != nil {
		return update.CallbackQuery.Message.Message.Chat.Username
	}

	return ""
}

func validatePrice(price string) (decimal.Decimal, error) {
	if price == "" {
		return decimal.Decimal{}, fmt.Errorf("empty price")
	}

	if strings.Contains(price, ",") {
		price = strings.ReplaceAll(price, ",", ".")
	}

	if len(price) > 12 {
		return decimal.Decimal{}, fmt.Errorf("слишком большое число")
	}

	res, err := decimal.NewFromString(price)
	if err != nil {
		return decimal.Decimal{}, fmt.Errorf("невалидный формат")
	}

	if res.LessThan(decimal.NewFromInt(0)) {
		return decimal.Decimal{}, fmt.Errorf("не может быть отрицательным числом")
	}

	return res, nil
}

func validatePair(pair string) error {
	if pair == "" {
		return fmt.Errorf("empty pair")
	}

	if len(pair) > 20 {
		return fmt.Errorf("слишком длинное название")
	}

	return nil
}
