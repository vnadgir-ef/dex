package sql

import (
	"database/sql"
	"fmt"
)

// SQLite
type SQLite struct {
	// File to
	File string `yaml:"file"`
}

func (s *SQLite) open() (*conn, error) {
	db, err := sql.Open("sqlite3", s.File)
	if err != nil {
		return nil, err
	}
	if s.File == ":memory:" {
		// sqlite3 uses file locks to coordinate concurrent access. In memory
		// doesn't support this, so limit the number of connections to 1.
		db.SetMaxOpenConns(1)
	}
	c := &conn{db, sqlite3}
	if _, err := c.migrate(); err != nil {
		return nil, fmt.Errorf("failed to perform migrations: %v", err)
	}
	return c, nil
}
