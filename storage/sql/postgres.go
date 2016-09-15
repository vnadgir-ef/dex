package sql

import "time"

// Postgres options for creating an SQL db.
type Postgres struct {
	Database string
	User     string
	Password string
	Host     string

	SSLCAFile string

	SSLKeyFile  string
	SSLCertFile string

	ConnectionTimeout time.Duration
}
