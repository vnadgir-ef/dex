package sql

import "testing"

func TestMigrate(t *testing.T) {
	s := SQLite{File: ":memory:"}
	conn, err := s.open()
	if err != nil {
		t.Fatal(err)
	}
	conn.Close()
}
