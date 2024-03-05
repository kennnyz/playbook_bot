-- +goose Up
-- +goose StatementBegin
CREATE TABLE Users (
                       user_id SERIAL PRIMARY KEY,
                       username TEXT,
                       chat_id BIGINT UNIQUE
);

CREATE TABLE PAIRS (
                       pair_id SERIAL PRIMARY KEY,
                       pair_name TEXT
);

CREATE TABLE UserPairs (
                           user_id INT REFERENCES Users(chat_id),
                           pair_id INT REFERENCES PAIRS(pair_id),
                           PRIMARY KEY (user_id, pair_id)
);

CREATE TABLE Deals (
                       deal_id SERIAL PRIMARY KEY,
                       user_id INT REFERENCES Users(chat_id),
                       pair_id INT REFERENCES PAIRS(pair_id),
                       buy_price DECIMAL,
                       sell_price DECIMAL,
                       profit DECIMAL,
                       profit_percent DECIMAL,
                       deal_date TIMESTAMP
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE Deals;
DROP TABLE UserPairs;
DROP TABLE PAIRS;
DROP TABLE Users;
-- +goose StatementEnd
