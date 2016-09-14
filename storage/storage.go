package storage

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base32"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	jose "gopkg.in/square/go-jose.v2"
)

var (
	// stubbed out for testing
	now = time.Now
)

// ErrNotFound is the error returned by storages if a resource cannot be found.
var ErrNotFound = errors.New("not found")

// Kubernetes only allows lower case letters for names.
//
// TODO(ericchiang): refactor ID creation onto the storage.
var encoding = base32.NewEncoding("abcdefghijklmnopqrstuvwxyz234567")

// NewID returns a random string which can be used as an ID for objects.
func NewID() string {
	buff := make([]byte, 8) // 64 bit random ID.
	if _, err := io.ReadFull(rand.Reader, buff); err != nil {
		panic(err)
	}
	// Trim padding
	return strings.TrimRight(encoding.EncodeToString(buff), "=")
}

// Storage is the storage interface used by the server. Implementations, at minimum
// require compare-and-swap atomic actions.
//
// Implementations are expected to perform their own garbage collection of
// expired objects (expect keys, which are handled by the server).
type Storage interface {
	Close() error

	// TODO(ericchiang): Let the storages set the IDs of these objects.
	CreateAuthRequest(a AuthRequest) error
	CreateClient(c Client) error
	CreateAuthCode(c AuthCode) error
	CreateRefresh(r RefreshToken) error

	// TODO(ericchiang): return (T, bool, error) so we can indicate not found
	// requests that way instead of using ErrNotFound.
	GetAuthRequest(id string) (AuthRequest, error)
	GetAuthCode(id string) (AuthCode, error)
	GetClient(id string) (Client, error)
	GetKeys() (Keys, error)
	GetRefresh(id string) (RefreshToken, error)

	ListClients() ([]Client, error)
	ListRefreshTokens() ([]RefreshToken, error)

	// Delete methods MUST be atomic.
	DeleteAuthRequest(id string) error
	DeleteAuthCode(code string) error
	DeleteClient(id string) error
	DeleteRefresh(id string) error

	// Update functions are assumed to be a performed within a single object transaction.
	UpdateClient(id string, updater func(old Client) (Client, error)) error
	UpdateKeys(updater func(old Keys) (Keys, error)) error
	UpdateAuthRequest(id string, updater func(a AuthRequest) (AuthRequest, error)) error
}

// Client represents an OAuth2 client.
//
// For further reading see:
//   * Trusted peers: https://developers.google.com/identity/protocols/CrossClientAuth
//   * Public clients: https://developers.google.com/api-client-library/python/auth/installed-app
type Client struct {
	// Client ID and secret used to identify the client.
	ID     string `json:"id" yaml:"id"`
	Secret string `json:"secret" yaml:"secret"`

	// A registered set of redirect URIs. When redirecting from dex to the client, the URI
	// requested to redirect to MUST match one of these values, unless the client is "public".
	RedirectURIs []string `json:"redirectURIs" yaml:"redirectURIs"`

	// TrustedPeers are a list of peers which can issue tokens on this client's behalf using
	// the dynamic "oauth2:server:client_id:(client_id)" scope. If a peer makes such a request,
	// this client's ID will appear as the ID Token's audience.
	//
	// Clients inherently trust themselves.
	TrustedPeers []string `json:"trustedPeers" yaml:"trustedPeers"`

	// Public clients must use either use a redirectURL 127.0.0.1:X or "urn:ietf:wg:oauth:2.0:oob"
	Public bool `json:"public" yaml:"public"`

	// Name and LogoURL used when displaying this client to the end user.
	Name    string `json:"name" yaml:"name"`
	LogoURL string `json:"logoURL" yaml:"logoURL"`
}

// Claims represents the ID Token claims supported by the server.
type Claims struct {
	UserID        string
	Username      string
	Email         string
	EmailVerified bool

	Groups []string
}

// AuthRequest represents a OAuth2 client authorization request. It holds the state
// of a single auth flow up to the point that the user authorizes the client.
type AuthRequest struct {
	// ID used to identify the authorization request.
	ID string

	// ID of the client requesting authorization from a user.
	ClientID string

	// Values parsed from the initial request. These describe the resources the client is
	// requesting as well as values describing the form of the response.
	ResponseTypes []string
	Scopes        []string
	RedirectURI   string
	Nonce         string
	State         string

	// The client has indicated that the end user must be shown an approval prompt
	// on all requests. The server cannot cache their initial action for subsequent
	// attempts.
	ForceApprovalPrompt bool

	Expiry time.Time

	// Has the user proved their identity through a backing identity provider?
	//
	// If false, the following fields are invalid.
	LoggedIn bool

	// The identity of the end user. Generally nil until the user authenticates
	// with a backend.
	Claims Claims

	// The connector used to login the user and any data the connector wishes to persists.
	// Set when the user authenticates.
	ConnectorID   string
	ConnectorData []byte
}

// AuthCode represents a code which can be exchanged for an OAuth2 token response.
//
// This value is created once an end user has authorized a client, the server has
// redirect the end user back to the client, but the client hasn't exchanged the
// code for an access_token and id_token.
type AuthCode struct {
	// Actual string returned as the "code" value.
	ID string

	// The client this code value is valid for. When exchanging the code for a
	// token response, the client must use its client_secret to authenticate.
	ClientID string

	// As part of the OAuth2 spec when a client makes a token request it MUST
	// present the same redirect_uri as the initial redirect. This values is saved
	// to make this check.
	//
	// https://tools.ietf.org/html/rfc6749#section-4.1.3
	RedirectURI string

	// If provided by the client in the initial request, the provider MUST create
	// a ID Token with this nonce in the JWT payload.
	Nonce string

	// Scopes authorized by the end user for the client.
	Scopes []string

	// Authentication data provided by an upstream source.
	ConnectorID   string
	ConnectorData []byte
	Claims        Claims

	Expiry time.Time
}

// RefreshToken is an OAuth2 refresh token which allows a client to request new
// tokens on the end user's behalf.
type RefreshToken struct {
	// The actual refresh token.
	RefreshToken string

	// Client this refresh token is valid for.
	ClientID string

	// Authentication data provided by an upstream source.
	ConnectorID   string
	ConnectorData []byte
	Claims        Claims

	// Scopes present in the initial request. Refresh requests may specify a set
	// of scopes different from the initial request when refreshing a token,
	// however those scopes must be encompassed by this set.
	Scopes []string

	// Nonce value supplied during the initial redirect. This is required to be part
	// of the claims of any future id_token generated by the client.
	Nonce string
}

// VerificationKey is a rotated signing key which can still be used to verify
// signatures.
type VerificationKey struct {
	PublicKey *jose.JSONWebKey `json:"publicKey"`
	Expiry    time.Time        `json:"expiry"`
}

// Keys hold encryption and signing keys.
type Keys struct {
	// Key for creating and verifying signatures. These may be nil.
	SigningKey    *jose.JSONWebKey
	SigningKeyPub *jose.JSONWebKey

	// Old signing keys which have been rotated but can still be used to validate
	// existing signatures.
	VerificationKeys []VerificationKey

	// The next time the signing key will rotate.
	//
	// For caching purposes, implementations MUST NOT update keys before this time.
	NextRotation time.Time
}

// Sign creates a JWT using the signing key.
func (k Keys) Sign(payload []byte) (jws string, err error) {
	if k.SigningKey == nil {
		return "", fmt.Errorf("no key to sign payload with")
	}
	signingKey := jose.SigningKey{Key: k.SigningKey}

	switch key := k.SigningKey.Key.(type) {
	case *rsa.PrivateKey:
		// TODO(ericchiang): Allow different cryptographic hashes.
		signingKey.Algorithm = jose.RS256
	case *ecdsa.PrivateKey:
		switch key.Params() {
		case elliptic.P256().Params():
			signingKey.Algorithm = jose.ES256
		case elliptic.P384().Params():
			signingKey.Algorithm = jose.ES384
		case elliptic.P521().Params():
			signingKey.Algorithm = jose.ES512
		default:
			return "", errors.New("unsupported ecdsa curve")
		}
	}

	signer, err := jose.NewSigner(signingKey, &jose.SignerOptions{})
	if err != nil {
		return "", fmt.Errorf("new signier: %v", err)
	}
	signature, err := signer.Sign(payload)
	if err != nil {
		return "", fmt.Errorf("signing payload: %v", err)
	}
	return signature.CompactSerialize()
}
