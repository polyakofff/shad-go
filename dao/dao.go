//go:build !solution

package dao

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type postgresDao struct {
	db *sql.DB
}

func CreateDao(ctx context.Context, dsn string) (Dao, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, err
	}

	_, err = db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS users (
			id SERIAL PRIMARY KEY,
			name TEXT NOT NULL
		)
	`)
	if err != nil {
		db.Close()
		return nil, err
	}

	return &postgresDao{db: db}, nil
}

func (d *postgresDao) Create(ctx context.Context, u *User) (UserID, error) {
	var id UserID

	err := d.db.QueryRowContext(ctx, `
		INSERT INTO users (name)
		VALUES ($1)
		RETURNING id
	`, u.Name).Scan(&id)

	if err != nil {
		return 0, fmt.Errorf("create user: %w", err)
	}

	u.ID = id
	return id, nil
}

func (d *postgresDao) Update(ctx context.Context, u *User) error {
	res, err := d.db.ExecContext(ctx, `
		UPDATE users
		SET name = $1
		WHERE id = $2
	`, u.Name, u.ID)

	n, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if n == 0 {
		return sql.ErrNoRows
	}

	return nil
}

func (d *postgresDao) Delete(ctx context.Context, id UserID) error {
	res, err := d.db.ExecContext(ctx, `
		DELETE FROM users
		WHERE id = $1
	`, id)

	n, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if n == 0 {
		return sql.ErrNoRows
	}

	return nil
}

func (d *postgresDao) Lookup(ctx context.Context, id UserID) (User, error) {
	var user User

	err := d.db.QueryRowContext(ctx, `
		SELECT id, name
		FROM users
		WHERE id = $1
	`, id).Scan(&user.ID, &user.Name)

	if err != nil {
		return User{}, fmt.Errorf("lookup user: %w", err)
	}

	return user, nil
}

func (d *postgresDao) List(ctx context.Context) ([]User, error) {
	rows, err := d.db.QueryContext(ctx, `
		SELECT id, name
		FROM users
	`)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var user User

		if err := rows.Scan(&user.ID, &user.Name); err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}

		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate users: %w", err)
	}

	return users, nil
}

func (d *postgresDao) Close() error {
	return d.db.Close()
}
