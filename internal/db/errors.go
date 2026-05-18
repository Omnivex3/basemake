package db

import "errors"

var (
	ErrNoConnection = errors.New("no active database connection — run 'basemake connect' first")
	ErrUnsupported  = errors.New("unsupported database driver")
)
