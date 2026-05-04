package db

import (
	"github.com/jackc/pgx/v5/pgxpool"
)

type Client struct {
	pgConnPool *pgxpool.Pool
}

func NewClient(dbConnPool *pgxpool.Pool) *Client {
	return &Client{
		pgConnPool: dbConnPool,
	}
}
