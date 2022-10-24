package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgtype"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type MoneyStore struct {
	pool *pgxpool.Pool
}

func (store *MoneyStore) hasBalance(ctx context.Context, userID int) (bool, error) {
	const query = `SELECT EXISTS (SELECT id FROM deposit WHERE user_id = $1),
		EXISTS (SELECT id FROM money_send WHERE "to" = $1)`

	var depositExists, sendExists bool
	if err := store.pool.QueryRow(context.Background(), query, userID).Scan(&depositExists, &sendExists); err != nil {
		fmt.Printf("db error: %v", err)
		return false, err
	}

	return depositExists || sendExists, nil
}

func (store *MoneyStore) GetBalance(ctx context.Context, userID int) (int, error) {
	const query = `SELECT
		(SELECT COALESCE(SUM(amount), 0) FROM deposit WHERE user_id = $1) +
		(SELECT COALESCE(SUM(amount), 0) FROM money_send WHERE "to" = $1) -
		(SELECT COALESCE(SUM(amount), 0) FROM money_send WHERE "from" = $1) -
		(SELECT COALESCE(SUM(amount), 0) FROM payment WHERE user_id = $1 AND canceled_at IS NULL) AS balance`

	var balance int
	if err := store.pool.QueryRow(ctx, query, userID).Scan(&balance); err != nil {
		fmt.Printf("db error: %v", err)
		return 0, err
	}

	return balance, nil
}

func (store *MoneyStore) Deposit(ctx context.Context, deposit Deposit) error {
	const query = "INSERT INTO deposit (user_id, amount) VALUES ($1, $2)"

	res, err := store.pool.Exec(context.Background(), query, deposit.UserID, deposit.Amount)
	if err != nil {
		fmt.Printf("db error: %v", err)
		return err
	}

	if res.RowsAffected() == 0 {
		return errors.New("zero rows affected")
	}

	return nil
}

func (store *MoneyStore) Reserve(ctx context.Context, payment Payment) error {
	const query = "INSERT INTO payment (user_id, service_id, order_id, amount) VALUES ($1, $2, $3, $4)"

	res, err := store.pool.Exec(context.Background(), query, payment.UserID, payment.ServiceID, payment.OrderID, payment.Amount)
	if err != nil {
		fmt.Printf("db error: %v", err)
		return err
	}

	if res.RowsAffected() == 0 {
		return errors.New("zero rows affected")
	}

	return nil
}

func (store *MoneyStore) GetPayment(ctx context.Context, payment Payment) (int, pgtype.Timestamptz, pgtype.Timestamptz, error) {
	const query = "SELECT id, confirmed_at, canceled_at FROM payment WHERE user_id=$1 AND service_id=$2 AND order_id=$3"

	var paymentID int
	var paymentConfirmedAt pgtype.Timestamptz
	var paymentCanceledAt pgtype.Timestamptz

	if err := store.pool.QueryRow(context.Background(), query, payment.UserID, payment.ServiceID, payment.OrderID).Scan(&paymentID, &paymentConfirmedAt, &paymentCanceledAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			fmt.Printf("db error: %v", err)
			return 0, paymentConfirmedAt, paymentCanceledAt, errors.New("payment not found")
		}
		return 0, paymentConfirmedAt, paymentCanceledAt, err
	}

	return paymentID, paymentConfirmedAt, paymentConfirmedAt, nil
}

func (store *MoneyStore) Confirm(ctx context.Context, paymentID int) error {
	const query = "UPDATE payment SET confirmed_at=now() WHERE id=$1"

	res, err := store.pool.Exec(context.Background(), query, paymentID)
	if err != nil {
		fmt.Printf("db error: %v", err)
		return err
	}

	if res.RowsAffected() == 0 {
		return errors.New("zero rows affected")
	}

	return nil
}

func (store *MoneyStore) Cancel(ctx context.Context, paymentID int) error {
	const query = "UPDATE payment SET canceled_at=now() WHERE id=$1"

	res, err := store.pool.Exec(context.Background(), query, paymentID)
	if err != nil {
		fmt.Printf("db error: %v", err)
		return err
	}

	if res.RowsAffected() == 0 {
		return errors.New("zero rows affected")
	}

	return nil
}

func (store *MoneyStore) SendMoney(ctx context.Context, moneySend MoneySend) error {
	const query = `INSERT INTO money_send ("from", "to", amount) VALUES ($1, $2, $3);`

	res, err := store.pool.Exec(context.Background(), query, moneySend.From, moneySend.To, moneySend.Amount)
	if err != nil {
		fmt.Printf("db error: %v", err)
		return err
	}

	if res.RowsAffected() == 0 {
		return errors.New("zero rows affected")
	}

	return nil
}

func (store *MoneyStore) Transactions(ctx context.Context, transaction Transaction) ([]map[string]interface{}, error) {
	var query = `SELECT * FROM
		(SELECT 'deposit' "type", '' "comment", amount, deposit_at ts FROM deposit WHERE user_id = $1
		UNION ALL
		SELECT 'money_received' "type", 'user#' || "from" "comment", amount, sent_at ts FROM money_send WHERE "to" = $1
		UNION ALL
		SELECT 'payment_canceled' "type", 'order#' || order_id || ' service#' || service_id "comment", amount, canceled_at ts FROM payment WHERE "user_id" = $1 AND canceled_at IS NOT NULL
		UNION ALL
		SELECT 'money_sent' "type", 'user#' || "to" "comment", -amount, sent_at ts FROM money_send WHERE "from" = $1
		UNION ALL
		SELECT 'payment' "type", 'order#' || order_id || ' service#' || service_id "comment", -amount, reserved_at ts FROM payment WHERE "user_id" = $1) stats`

	var parameters []interface{}
	parameters = append(parameters, transaction.UserID)

	if transaction.DateSortBy != "" {
		if transaction.DateSortBy == "ASC" {
			query += " ORDER BY ts ASC"
		} else if transaction.DateSortBy == "DESC" {
			query += " ORDER BY ts DESC"
		}
	} else if transaction.AmountSortBy != "" {
		if transaction.AmountSortBy == "ASC" {
			query += " ORDER BY amount ASC"
		} else if transaction.AmountSortBy == "DESC" {
			query += " ORDER BY amount DESC"
		}
	} else {
		query += " ORDER BY ts ASC"
	}

	query += " LIMIT $2 OFFSET $3"
	parameters = append(parameters, transaction.Limit)
	parameters = append(parameters, transaction.Offset)

	res, err := store.pool.Query(context.Background(), query, parameters...)
	if err != nil {
		fmt.Printf("db error: %v", err)
		return nil, err
	}

	result := make([]map[string]interface{}, 0)

	for res.Next() {
		var t_type string
		var comment string
		var amount int
		var ts pgtype.Timestamptz
		r := make(map[string]interface{})

		res.Scan(&t_type, &comment, &amount, &ts)

		r["type"] = t_type
		r["comment"] = comment
		r["amount"] = amount
		r["ts"] = ts

		result = append(result, r)
	}

	return result, nil
}
