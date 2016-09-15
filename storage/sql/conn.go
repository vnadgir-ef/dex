package sql

import (
	"database/sql"
	"regexp"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

// Match a postgres query binds. E.g. "$1", "$12", etc.
var bindRegexp = regexp.MustCompile(`\$\d+`)

func matchLiteral(s string) *regexp.Regexp {
	return regexp.MustCompile(`\b` + regexp.QuoteMeta(s) + `\b`)
}

var (
	// The "github.com/lib/pq" driver is the default flavor. All others are
	// translations of this.
	postgres = flavor{}

	sqlite3 = flavor{
		queryReplacers: []replacer{
			{bindRegexp, "?"},
			// Translate for booleans to integers.
			{matchLiteral("true"), "1"},
			{matchLiteral("false"), "0"},
			{matchLiteral("boolean"), "integer"},
			// Translate types.
			{matchLiteral("bytea"), "blob"},
			{matchLiteral("timestamp"), "integer"},
			{regexp.MustCompile(`\bnow\(\)`), "date('now')"},
		},
	}

	// Incomplete.
	mysql = flavor{
		queryReplacers: []replacer{
			{bindRegexp, "?"},
		},
	}
)

type replacer struct {
	re   *regexp.Regexp
	with string
}

// flavor represents a specific SQL implementation.
type flavor struct {
	queryReplacers []replacer

	// Optional function to create and finish a transaction. This is mainly for
	// cockroach db support which requires special retry logic provided by their
	// client package.
	//
	// https://github.com/cockroachdb/docs/blob/63761c2e/_includes/app/txn-sample.go#L41-L44
	executeTx func(db *sql.DB, fn func(*sql.Tx) error) error
}

func (f flavor) translate(query string) string {
	// TODO(ericchiang): heavy cashing
	for _, r := range f.queryReplacers {
		query = r.re.ReplaceAllString(query, r.with)
	}
	return query
}

type conn struct {
	db     *sql.DB
	flavor flavor
}

func (c *conn) Close() error {
	return c.db.Close()
}

func (c *conn) Exec(query string, args ...interface{}) (sql.Result, error) {
	query = c.flavor.translate(query)
	return c.db.Exec(query, args...)
}

func (c *conn) Query(query string, args ...interface{}) (*sql.Rows, error) {
	query = c.flavor.translate(query)
	return c.db.Query(query, args...)
}

func (c *conn) QueryRow(query string, args ...interface{}) *sql.Row {
	query = c.flavor.translate(query)
	return c.db.QueryRow(query, args...)
}

func (c *conn) ExecTx(fn func(tx *trans) error) error {
	if c.flavor.executeTx != nil {
		return c.flavor.executeTx(c.db, func(sqlTx *sql.Tx) error {
			return fn(&trans{sqlTx, c.flavor})
		})
	}

	sqlTx, err := c.db.Begin()
	if err != nil {
		return err
	}
	if err := fn(&trans{sqlTx, c.flavor}); err != nil {
		sqlTx.Rollback()
		return err
	}
	return sqlTx.Commit()
}

type trans struct {
	tx     *sql.Tx
	flavor flavor
}

func (t *trans) Exec(query string, args ...interface{}) (sql.Result, error) {
	query = t.flavor.translate(query)
	return t.tx.Exec(query, args...)
}

func (t *trans) Query(query string, args ...interface{}) (*sql.Rows, error) {
	query = t.flavor.translate(query)
	return t.tx.Query(query, args...)
}

func (t *trans) QueryRow(query string, args ...interface{}) *sql.Row {
	query = t.flavor.translate(query)
	return t.tx.QueryRow(query, args...)
}
