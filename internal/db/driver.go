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
	}
}

func detectDriver(dsn string) (driverConnector, error) {
	for _, d := range drivers {
		if strings.HasPrefix(dsn, d.Scheme()+"://") || strings.HasPrefix(dsn, d.Scheme()+":") {
			return d, nil
		}
	}
	return nil, ErrUnsupported
}
