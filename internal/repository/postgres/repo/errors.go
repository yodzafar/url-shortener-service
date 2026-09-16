package repo

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

const (
	codeUniqueViolation     = "23505"
	codeForeignKeyViolation = "23503"
	codeCheckViolation      = "23514"
)

func isUniqueViolation(err error) bool {
	return pgCode(err) == codeUniqueViolation
}

func isForeignKeyViolation(err error) bool {
	return pgCode(err) == codeForeignKeyViolation
}

func pgCode(err error) string {
	if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok {
		return pgErr.Code
	}
	return ""
}
