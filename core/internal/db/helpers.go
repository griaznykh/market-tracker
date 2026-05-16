package db

import (
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
)

func IsUniqueViolation(
	err error,
	constraint string,
) bool {

	var pgErr *pgconn.PgError

	if !errors.As(err, &pgErr) {
		return false
	}

	return pgErr.Code == "23505" &&
		pgErr.ConstraintName == constraint
}

func TblColumn(column, tlbName string, suffix ...string) string {
	r := fmt.Sprintf("%s.%s", tlbName, column)
	if len(suffix) > 0 {
		r += " " + strings.Join(suffix, " ")
	}
	return r
}
