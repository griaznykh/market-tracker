package db

import (
	"context"
	"errors"
	"fmt"
	"market-service/internal/schema"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	sq "github.com/Masterminds/squirrel"
)

const (
	userTableName = "users"
)

var (
	ErrUserDoesntExist  = errors.New("requested user doesn't exist")
	ErrUserAlreadyExist = errors.New("user already exist")
)

func userColumns() []string {
	return []string{
		"id",
		"email",
		"password",
		"created_at",
	}
}

func (c *Client) GetUser(ctx context.Context, userId uuid.UUID) (user *schema.User, err error) {
	query, _, err := sq.Select(userColumns()...).From(userTableName).Where(sq.Eq{TblColumn("id", userTableName): userId}).Limit(1).PlaceholderFormat(sq.Dollar).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build query: %w", err)
	}

	rows, err := c.pgConnPool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("exec query: %w", err)
	}
	defer rows.Close()

	user, err = pgx.CollectOneRow(rows, pgx.RowToAddrOfStructByNameLax[schema.User])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserDoesntExist
		}
		return nil, fmt.Errorf("scan rows to struct: %w", err)
	}

	return user, nil
}

func (c *Client) GetUserByEmail(ctx context.Context, email string) (user *schema.User, err error) {
	query, args, err := sq.Select(userColumns()...).From(userTableName).Where(sq.Eq{TblColumn("email", userTableName): email}).Limit(1).PlaceholderFormat(sq.Dollar).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build query: %w", err)
	}

	rows, err := c.pgConnPool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("exec query: %w", err)
	}
	defer rows.Close()

	user, err = pgx.CollectOneRow(rows, pgx.RowToAddrOfStructByNameLax[schema.User])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserDoesntExist
		}
		return nil, fmt.Errorf("scan rows to struct: %w", err)
	}

	return user, nil
}

func (c *Client) CreateUser(ctx context.Context, user *schema.User) error {
	query, args, err := sq.Insert(userTableName).Columns(
		"id",
		"email",
		"password",
	).Values(
		user.Id,
		user.Email,
		user.Password,
	).Suffix("RETURNING created_at").PlaceholderFormat(sq.Dollar).ToSql()
	if err != nil {
		return fmt.Errorf("build query: %w", err)
	}

	err = c.pgConnPool.QueryRow(ctx, query, args...).Scan(&user.CreatedAt)
	if err != nil {
		if IsUniqueViolation(err, "users_email_key") {
			return ErrUserAlreadyExist
		}
		return fmt.Errorf("exec query: %w", err)
	}

	return nil
}
