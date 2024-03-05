package main

import (
	"time"

	"github.com/shopspring/decimal"
)

type UserState int

const (
	StateIdle UserState = iota
	StateAwaitingSavePair
	StateAwaitingDealPair
	StateAwaitingAmount
	StateAwaitingBuyPrice
	StateAwaitingSellPrice
)

type User struct {
	Name   string
	ChatID int64
}

type Deal struct {
	Pair          string
	ID            int64
	Amount        decimal.Decimal
	BuyPrice      decimal.Decimal
	SellPrice     decimal.Decimal
	Profit        decimal.Decimal
	ProfitPercent decimal.Decimal
	Date          time.Time
}
