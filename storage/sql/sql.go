// Package sql provides a SQL implementations of the storage type.
//
// The package currently support Sqlite3, MySQL, Postgres, and CockroachDB.
package sql

// func (c *conn) CreateAuthRequest(a storage.AuthRequest) error
// func (c *conn) CreateClient(c storage.Client) error
// func (c *conn) CreateAuthCode(c storage.AuthCode) error
// func (c *conn) CreateRefresh(r storage.RefreshToken) error
// func (c *conn) GetAuthRequest(id string) (storage.AuthRequest, error)
// func (c *conn) GetAuthCode(id string) (storage.AuthCode, error)
// func (c *conn) GetClient(id string) (storage.Client, error)
// func (c *conn) GetKeys() (storage.Keys, error)
// func (c *conn) GetRefresh(id string) (storage.RefreshToken, error)
// func (c *conn) ListClients() ([]storage.Client, error)
// func (c *conn) ListRefreshTokens() ([]storage.RefreshToken, error)
// func (c *conn) DeleteAuthRequest(id string) error
// func (c *conn) DeleteAuthCode(code string) error
// func (c *conn) DeleteClient(id string) error
// func (c *conn) DeleteRefresh(id string) error
// func (c *conn) UpdateClient(id string, updater func(old storage.Client) (storage.Client, error)) error
// func (c *conn) UpdateKeys(updater func(old storage.Keys) (storage.Keys, error)) error
// func (c *conn) UpdateAuthRequest(id string, updater func(a storage.AuthRequest) (storage.AuthRequest, error)) error
