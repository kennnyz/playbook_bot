package main

import (
	"context"
	"github.com/go-telegram/bot"
	"os"
	"os/signal"
)

// Мапа для отслеживания состояний пользователей
var usersStates = make(map[int64]UserState)

// tmp database
var users = make(map[int64]*User)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	opts := []bot.Option{
		bot.WithDefaultHandler(defaultHandler),
		bot.WithCallbackQueryDataHandler("/add_pair", bot.MatchTypeExact, addPairCallbackHandler),
		bot.WithCallbackQueryDataHandler("/add_deal", bot.MatchTypeExact, addDealCallbackHandler),
		bot.WithCallbackQueryDataHandler("/get_history", bot.MatchTypeExact, getHistoryCallbackHandler),
		bot.WithMiddlewares(showMessageWithUserName),
	}

	token := os.Getenv("API_KEY")
	if token == "" {
		panic("API key not provided!")
	}

	b, err := bot.New(token, opts...)
	if err != nil {
		panic(err)
	}

	b.RegisterHandler(bot.HandlerTypeMessageText, "/start", bot.MatchTypeExact, startCommand)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/add_deal", bot.MatchTypeExact, addDealCallbackHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/add_pair", bot.MatchTypeExact, addPairCallbackHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/get_history", bot.MatchTypeExact, getHistoryCallbackHandler)

	b.Start(ctx)
}
