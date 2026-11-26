//go:build !solution

package ledger

import (
	"context"
	"database/sql"

	_ "github.com/jackc/pgx/v4/stdlib"
)

type ledgerImpl struct {
	db *sql.DB
}

type Execer interface {
	ExecContext(
		ctx context.Context,
		query string,
		args ...interface{},
	) (sql.Result, error)
}

func (l *ledgerImpl) Close() error {
	return l.db.Close()
}

func (l *ledgerImpl) CreateAccount(ctx context.Context, id ID) error {
	_, err := l.db.ExecContext(ctx, `
		INSERT INTO accounts (id, balance)
		VALUES ($1, $2)
	`, id, 0)

	return err
}

func (l *ledgerImpl) GetBalance(ctx context.Context, id ID) (Money, error) {
	var balance Money
	err := l.db.QueryRowContext(ctx, `
		SELECT balance
		FROM accounts
		WHERE id = $1
	`, id).Scan(&balance)

	if err != nil {
		return 0, err
	}

	return balance, nil
}

func deposit(ctx context.Context, db Execer, id ID, amount Money) error {
	_, err := db.ExecContext(ctx, `
		UPDATE accounts
		SET balance = balance + $1
		WHERE id = $2
	`, amount, id)

	return err
}

func (l *ledgerImpl) Deposit(ctx context.Context, id ID, amount Money) error {
	return deposit(ctx, l.db, id, amount)
}

func withdraw(ctx context.Context, db Execer, id ID, amount Money) error {
	res, err := db.ExecContext(ctx, `
		UPDATE accounts
		SET balance = balance - $1
		WHERE id = $2 AND balance >= $1
	`, amount, id)

	if err != nil {
		return err
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if affected == 0 {
		return ErrNoMoney
	}

	return nil
}

func (l *ledgerImpl) Withdraw(ctx context.Context, id ID, amount Money) error {
	return withdraw(ctx, l.db, id, amount)
}

// bad transfer
// func (l *ledgerImpl) Transfer(ctx context.Context, from, to ID, amount Money) error {
// 	tx, err := l.db.BeginTx(ctx, nil)
// 	if err != nil {
// 		return err
// 	}
// 	defer tx.Rollback()

// 	if err := withdraw(ctx, tx, from, amount); err != nil {
// 		return err
// 	}

// 	if err := deposit(ctx, tx, to, amount); err != nil {
// 		return err
// 	}

// 	return tx.Commit()
// }

func (l *ledgerImpl) Transfer(ctx context.Context, from, to ID, amount Money) error {
	tx, err := l.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// блокируем счета на время транзакции
	rows, err := tx.QueryContext(ctx, `
		SELECT id, balance
		FROM accounts
		WHERE id IN ($1, $2)
		ORDER BY id
		FOR UPDATE
	`, from, to)

	if err != nil {
		return err
	}

	balances := make(map[ID]Money)

	for rows.Next() {
		var id ID
		var balance Money

		if err := rows.Scan(&id, &balance); err != nil {
			rows.Close()
			return err
		}

		balances[id] = balance
	}

	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	if err := rows.Close(); err != nil {
		return err
	}

	if _, ok := balances[from]; !ok {
		return sql.ErrNoRows
	}
	if _, ok := balances[to]; !ok {
		return sql.ErrNoRows
	}
	if balances[from] < amount {
		return ErrNoMoney
	}

	if err := withdraw(ctx, tx, from, amount); err != nil {
		return err
	}

	if err := deposit(ctx, tx, to, amount); err != nil {
		return err
	}

	return tx.Commit()
}

func New(ctx context.Context, dsn string) (Ledger, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, err
	}

	_, err = db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS accounts (
			id TEXT PRIMARY KEY,
			balance BIGINT NOT NULL
		)
	`)
	if err != nil {
		db.Close()
		return nil, err
	}

	return &ledgerImpl{db: db}, nil
}
