package main

import (
	"github.com/shopspring/decimal"
	"time"
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
	Name        string
	ID          int64
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
