package main

import (
	"context"
	"database/sql"
	"errors"

	_ "github.com/mattn/go-sqlite3"
)

const schemaSQL = `
CREATE TABLE IF NOT EXISTS subscriptions (
	id         INTEGER PRIMARY KEY AUTOINCREMENT,
	name       TEXT NOT NULL,
	amount     REAL NOT NULL,
	currency   TEXT NOT NULL DEFAULT 'EUR',
	cycle      TEXT NOT NULL CHECK(cycle IN ('monthly', 'yearly', '2yearly')),
	start_date TEXT NOT NULL,
	notes      TEXT DEFAULT ''
);
`

var ErrNotFound = errors.New("subscription not found")

func OpenDB(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}

	if _, err := db.Exec(`PRAGMA foreign_keys = ON`); err != nil {
		db.Close()
		return nil, err
	}
	if err := Migrate(db); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}

func Migrate(db *sql.DB) error {
	_, err := db.Exec(schemaSQL)
	return err
}

func ListSubscriptions(ctx context.Context, db *sql.DB) ([]Subscription, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, name, amount, currency, cycle, start_date, COALESCE(notes, '')
		FROM subscriptions
		ORDER BY id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	subscriptions := []Subscription{}
	for rows.Next() {
		sub, err := scanSubscription(rows)
		if err != nil {
			return nil, err
		}
		subscriptions = append(subscriptions, sub)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return subscriptions, nil
}

func GetSubscription(ctx context.Context, db *sql.DB, id int) (Subscription, error) {
	row := db.QueryRowContext(ctx, `
		SELECT id, name, amount, currency, cycle, start_date, COALESCE(notes, '')
		FROM subscriptions
		WHERE id = ?
	`, id)

	sub, err := scanSubscription(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Subscription{}, ErrNotFound
	}
	if err != nil {
		return Subscription{}, err
	}

	return sub, nil
}

func InsertSubscription(ctx context.Context, db *sql.DB, sub Subscription) (Subscription, error) {
	result, err := db.ExecContext(ctx, `
		INSERT INTO subscriptions (name, amount, currency, cycle, start_date, notes)
		VALUES (?, ?, ?, ?, ?, ?)
	`, sub.Name, sub.Amount, sub.Currency, sub.Cycle, sub.StartDate, sub.Notes)
	if err != nil {
		return Subscription{}, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return Subscription{}, err
	}

	return GetSubscription(ctx, db, int(id))
}

func UpdateSubscription(ctx context.Context, db *sql.DB, id int, sub Subscription) (Subscription, error) {
	result, err := db.ExecContext(ctx, `
		UPDATE subscriptions
		SET name = ?, amount = ?, currency = ?, cycle = ?, start_date = ?, notes = ?
		WHERE id = ?
	`, sub.Name, sub.Amount, sub.Currency, sub.Cycle, sub.StartDate, sub.Notes, id)
	if err != nil {
		return Subscription{}, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return Subscription{}, err
	}
	if rowsAffected == 0 {
		return Subscription{}, ErrNotFound
	}

	return GetSubscription(ctx, db, id)
}

func DeleteSubscription(ctx context.Context, db *sql.DB, id int) error {
	result, err := db.ExecContext(ctx, `DELETE FROM subscriptions WHERE id = ?`, id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

type subscriptionScanner interface {
	Scan(dest ...any) error
}

func scanSubscription(scanner subscriptionScanner) (Subscription, error) {
	var sub Subscription
	err := scanner.Scan(
		&sub.ID,
		&sub.Name,
		&sub.Amount,
		&sub.Currency,
		&sub.Cycle,
		&sub.StartDate,
		&sub.Notes,
	)
	if err != nil {
		return Subscription{}, err
	}

	return sub, nil
}
