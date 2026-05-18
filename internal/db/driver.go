package db

import (
	"strings"
)

// driverConnector opens a database connection for a given DSN scheme
type driverConnector interface {
	Connect(dsn string) (Database, error)
	Scheme() string
}

var drivers []driverConnector

func init() {
	drivers = []driverConnector{
		&postgresDriver{},
		&mysqlDriver{},
		&sqliteDriver{},
	}
}

func detectDriver(dsn string) (driverConnector, error) {
	// Check registered drivers
	for _, d := range drivers {
		if strings.HasPrefix(dsn, d.Scheme()+"://") || strings.HasPrefix(dsn, d.Scheme()+":") {
			return d, nil
		}
	}

	// Aliases
	if strings.HasPrefix(dsn, "postgresql://") || strings.HasPrefix(dsn, "postgresql:") {
		return &postgresDriver{}, nil
	}

	return nil, ErrUnsupported
}
