package main

import "database/sql"

type repository struct {
	conn *sql.DB
}

func newRepository(dsn string) (*repository, error) {
	conn, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}

	return &repository{conn: conn}, nil
}

func (r *repository) saveUser(u *User) error {
	return nil
}

func (r *repository) getUser(id int64) (*User, error) {
	return nil, nil
}

func (r *repository) saveDeal(d *Deal) error {
	return nil
}

func (r *repository) getDeals(id int64) ([]*Deal, error) {
	return nil, nil
}

func (r *repository) savePair(id int64, pair string) error {
	return nil
}

func (r *repository) getPairs(id int64) ([]string, error) {
	return nil, nil
}
