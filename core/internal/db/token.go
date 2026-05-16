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
	tokenTableName = "api_tokens"
)

var (
	ErrTokenDoesntExist  = errors.New("requested token doesn't exist")
	ErrTokenAlreadyExist = errors.New("token already exist")
)

func tokenColumns() []string {
	return []string{
		"id",
		"user_id",
		"name",
		"token",
		"created_at",
	}
}

func (c *Client) GetToken(ctx context.Context, tokenId uuid.UUID) (token *schema.ApiToken, err error) {
	query, _, err := sq.Select(tokenColumns()...).From(tokenTableName).Where(sq.Eq{TblColumn("id", tokenTableName): tokenId}).Limit(1).PlaceholderFormat(sq.Dollar).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build query: %w", err)
	}

	rows, err := c.pgConnPool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("exec query: %w", err)
	}
	defer rows.Close()

	token, err = pgx.CollectOneRow(rows, pgx.RowToAddrOfStructByNameLax[schema.ApiToken])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTokenDoesntExist
		}
		return nil, fmt.Errorf("scan rows to struct: %w", err)
	}

	return token, nil
}

func (c *Client) CreateToken(ctx context.Context, token *schema.ApiToken) error {
	query, args, err := sq.Insert(tokenTableName).Columns(
		"id",
		"user_id",
		"token",
	).Values(
		token.Id,
		token.UserId,
		token.Token,
	).Suffix("RETURNING created_at").PlaceholderFormat(sq.Dollar).ToSql()
	if err != nil {
		return fmt.Errorf("build query: %w", err)
	}

	err = c.pgConnPool.QueryRow(ctx, query, args...).Scan(&token.CreatedAt)
	if err != nil {
		if IsUniqueViolation(err, "api_tokens_token_key") {
			return ErrTokenAlreadyExist
		}
		return fmt.Errorf("exec query: %w", err)
	}

	return nil
}
