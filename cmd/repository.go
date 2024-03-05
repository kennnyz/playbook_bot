package main

import (
	"database/sql"
	"errors"
	"log"

	_ "github.com/lib/pq"
)

type repository struct {
	conn *sql.DB
}

func newRepository(dsn string) (*repository, error) {
	conn, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}

	// Проверяем соединение с базой
	if err := conn.Ping(); err != nil {
		return nil, err
	}

	return &repository{conn: conn}, nil
}

func (r *repository) saveUser(u *User) error {
	query := `
        INSERT INTO Users (username, chat_id)
        VALUES ($1, $2)
    `
	_, err := r.conn.Exec(query, u.Name, u.ChatID)
	if err != nil {
		return err
	}

	return nil
}

func (r *repository) getUser(id int64) (*User, error) {
	query := `
		SELECT username, chat_id
		FROM Users
		WHERE chat_id = $1
	`

	var user User
	if err := r.conn.QueryRow(query, id).Scan(&user.Name, &user.ChatID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return &user, nil
}

func (r *repository) saveDeal(d *Deal, userID int64) error {
	var pairID int64
	err := r.conn.QueryRow("SELECT pair_id FROM PAIRS WHERE pair_name = $1", d.Pair).Scan(&pairID)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO Deals (user_id, pair_id, buy_price, sell_price, profit, profit_percent, deal_date)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err = r.conn.Exec(query, userID, pairID, d.BuyPrice, d.SellPrice, d.Profit, d.ProfitPercent, d.Date)
	if err != nil {
		return err
	}

	return nil
}

func (r *repository) getDeals(userID int64) ([]*Deal, error) {
	query := `
        SELECT d.deal_id, p.pair_name, d.buy_price, d.sell_price, d.profit, d.profit_percent, d.deal_date
        FROM Deals AS d
        JOIN Pairs AS p ON d.pair_id = p.pair_id
        WHERE d.user_id = $1
    `

	rows, err := r.conn.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	deals := []*Deal{}

	for rows.Next() {
		var deal Deal
		if err := rows.Scan(&deal.ID, &deal.Pair, &deal.BuyPrice, &deal.SellPrice, &deal.Profit, &deal.ProfitPercent, &deal.Date); err != nil {
			return nil, err
		}
		deals = append(deals, &deal)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return deals, nil
}

func (r *repository) savePair(userID int64, pair string) error {
	// Проверяем, существует ли уже такая пара в таблице PAIRS
	var pairID int64
	err := r.conn.QueryRow("SELECT pair_id FROM PAIRS WHERE pair_name = $1", pair).Scan(&pairID)
	if errors.Is(err, sql.ErrNoRows) { // Если пары нет, создаем ее
		_, err := r.conn.Exec("INSERT INTO PAIRS (pair_name) VALUES ($1)", pair)
		if err != nil {
			log.Println(err)
			return err
		}

		// Получаем ChatID созданной пары
		err = r.conn.QueryRow("SELECT pair_id FROM PAIRS WHERE pair_name = $1", pair).Scan(&pairID)
		if err != nil {
			log.Println(err)
			return err
		}
	} else if err != nil {
		log.Println(err)
		return err
	}

	// Добавляем запись о паре для пользователя в таблицу UserPairs
	_, err = r.conn.Exec("INSERT INTO UserPairs (user_id, pair_id) VALUES ($1, $2)", userID, pairID)
	if err != nil {
		log.Println(err)
		return err
	}

	return nil
}

func (r *repository) getPairs(id int64) ([]string, error) {
	query := `
		SELECT p.pair_name
		FROM UserPairs AS up
		JOIN Pairs AS p ON up.pair_id = p.pair_id
		WHERE up.user_id = $1
	`

	rows, err := r.conn.Query(query, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pairs []string

	for rows.Next() {
		var pair string
		if err := rows.Scan(&pair); err != nil {
			return nil, err
		}
		pairs = append(pairs, pair)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return pairs, nil
}

func (r *repository) getPair(id int64, pair string) (bool, error) {
	query := `
		SELECT EXISTS (
			SELECT 1
			FROM UserPairs AS up
			JOIN Pairs AS p ON up.pair_id = p.pair_id
			WHERE up.user_id = $1 AND p.pair_name = $2
		)
	`

	var exists bool
	if err := r.conn.QueryRow(query, id, pair).Scan(&exists); err != nil {
		return false, err
	}

	return exists, nil
}
