package sql

import (
	"database/sql"
	"fmt"
)

func (c *conn) migrate() (int, error) {
	_, err := c.Exec(`create table if not exists migrations (
		num integer not null,
		at timestamp not null
	);`)
	if err != nil {
		return 0, fmt.Errorf("creating migration table: %v", err)
	}

	i := 0
	done := false
	for {
		err := c.ExecTx(func(tx *trans) error {
			// Within a transaction, perform a single migration.
			var (
				num sql.NullInt64
				n   int
			)
			if err := tx.QueryRow(`select max(num) from migrations;`).Scan(&num); err != nil {
				if err != sql.ErrNoRows {
					return fmt.Errorf("select max migration: %v", err)
				}
				n = 0
			} else if num.Valid {
				n = int(num.Int64)
			} else {
				n = 0
			}
			if n >= len(migrations) {
				done = true
				return nil
			}
			migrationNum := n + 1
			m := migrations[n]
			if _, err := tx.Exec(m); err != nil {
				return fmt.Errorf("migration %d failed: %v", migrationNum, err)
			}

			q := `insert into migrations (num, at) values ($1, now());`
			if _, err := tx.Exec(q, migrationNum); err != nil {
				return fmt.Errorf("update migration table: %v", err)
			}
			return nil
		})
		if err != nil {
			return i, err
		}
		if done {
			break
		}
		i++
	}

	return i, nil
}

// All SQL flavors share migration strategies.
//
// TODO(ericchiang): consider adding additional fields like "forDriver"
var migrations = []string{
	`create table client (
		id text not null primary key,
		secret text not null,
		redirect_uris bytea not null, -- JSON array of strings
		trusted_peers bytea not null, -- JSON array of strings
		public boolean not null,
		name text not null,
		logo_url text not null
	);

	create table auth_request (
		id text not null primary key,
		client_id text not null,
		response_types bytea not null, -- JSON array of strings
		scopes bytea not null, -- JSON array of strings
		redirect_uri text not null,
		nonce text not null,
		state text not null,
		force_approval_prompt boolean not null,

		logged_in boolean not null default false,

		claims_user_id string not null default "",
		claims_username string not null default "",
		claims_email string not null default "",
		claims_email_verified boolean not null default false,

		connector_id string not null default "",
		connector_data bytea,

		expiry timestamp not null
	);

	create table auth_code (
		id text not null primary key,
		client_id text not null,
		scopes bytea not null, -- JSON array of strings
		nonce text not null,

		claims_user_id text not null,
		claims_username text not null,
		claims_email text not null,
		claims_email_verified boolean not null,

		connector_id text not null,
		connector_data bytea,

		expiry timestamp not null
	);

	create table refresh_token (
		id text not null primary key,
		client_id text not null,
		scopes bytea not null, -- JSON array of strings
		nonce text not null,

		claims_user_id text not null,
		claims_username text not null,
		claims_email text not null,
		claims_email_verified boolean not null,

		connector_id text not null,
		connector_data bytea,

		expiry timestamp not null
	);
	`,
}
