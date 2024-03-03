package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

var users = make(map[int64]User)

type User struct {
	UserPairs map[string]struct{}
	UserDeals []*Deal
}

type Deal struct {
	Pair          string
	BuyPrice      float64
	SellPrice     float64
	Profit        float64
	ProfitPercent float64
	Date          time.Time
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	opts := []bot.Option{
		bot.WithDefaultHandler(defaultHandler),
		bot.WithCallbackQueryDataHandler("button", bot.MatchTypePrefix, callbackHandler),
		bot.WithMiddlewares(showMessageWithUserName),
	}

	b, err := bot.New("6791125665:AAHKZBXRdFjppkhzmPgmDp4oh2MoInIj3Go", opts...)
	if err != nil {
		panic(err)
	}

	b.RegisterHandler(bot.HandlerTypeMessageText, "/start", bot.MatchTypeExact, helloWorld)

	b.Start(ctx)
}

func handler(ctx context.Context, b *bot.Bot, update *models.Update) {
	//b.SendMessage(ctx, &bot.SendMessageParams{
	//	ChatID: update.Message.Chat.ID,
	//	Text:   update.Message.Text,
	//})
}

func defaultHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	kb := &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{Text: "Button 1", CallbackData: "button_1"},
				{Text: "Button 2", CallbackData: "button_2"},
			}, {
				{Text: "Button 3", CallbackData: "button_3"},
			},
		},
	}

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      update.Message.Chat.ID,
		Text:        "Click by button",
		ReplyMarkup: kb,
	})
}

func callbackHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	// answering callback query first to let Telegram know that we received the callback query,
	// and we're handling it. Otherwise, Telegram might retry sending the update repetitively
	// as it thinks the callback query doesn't reach to our application. learn more by
	// reading the footnote of the https://core.telegram.org/bots/api#callbackquery type.
	b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: update.CallbackQuery.ID,
		ShowAlert:       false,
	})
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.CallbackQuery.Message.Message.Chat.ID,
		Text:   "You selected the button: " + update.CallbackQuery.Data,
	})

	// Remove the buttons by editing the message's reply markup
	b.EditMessageReplyMarkup(ctx, &bot.EditMessageReplyMarkupParams{
		ChatID:    update.CallbackQuery.Message.Message.Chat.ID,
		MessageID: update.CallbackQuery.Message.Message.ID,
		ReplyMarkup: &models.InlineKeyboardMarkup{
			InlineKeyboard: [][]models.InlineKeyboardButton{},
		},
	})
}

func addDealCommand(ctx context.Context, b *bot.Bot, update *models.Update) {

}

// addPairCommand adding new pair to the list
func addPairCommand(ctx context.Context, b *bot.Bot, update *models.Update) {

}

func showMessageWithUserName(next bot.HandlerFunc) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		if update.Message != nil {
			log.Printf("%s say: %s", update.Message.From.Username, update.Message.Text)
		}
		next(ctx, b, update)
	}
}

func helloWorld(ctx context.Context, b *bot.Bot, update *models.Update) {
	message := "Привет 👋\nЭто Бот с помощью которого можно вести учет ваших сделок📖\n\nПоддерживаемые команды:\n/add_deal - добавить новую сделку\n/get_history - получить историю сделок"
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   message,
	})
}
