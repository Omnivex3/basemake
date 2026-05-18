package db

import "errors"

var (
	ErrNoConnection = errors.New("no active database connection — run 'dbai connect' first")
	ErrUnsupported  = errors.New("unsupported database driver")
)
