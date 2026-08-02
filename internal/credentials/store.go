// Package credentials stores the Bluff session in the operating system's
// native credential manager.
package credentials

import (
	"context"
	"errors"

	"github.com/zalando/go-keyring"
)

const service = "com.thsnkhn.bluff"

// ErrNotFound indicates that this device has no saved session.
var ErrNotFound = errors.New("saved Bluff session not found")

// Store persists a single bearer token in the native credential manager.
type Store struct {
	account string
}

// NewKeyringStore creates a credential store scoped to an API deployment.
func NewKeyringStore(account string) *Store {
	return &Store{account: account}
}

// Load retrieves a session token.
func (s *Store) Load(ctx context.Context) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	token, err := keyring.Get(service, s.account)
	if errors.Is(err, keyring.ErrNotFound) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", err
	}
	return token, nil
}

// Save persists a session token.
func (s *Store) Save(ctx context.Context, token string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return keyring.Set(service, s.account, token)
}

// Delete removes the saved session token.
func (s *Store) Delete(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	err := keyring.Delete(service, s.account)
	if errors.Is(err, keyring.ErrNotFound) {
		return nil
	}
	return err
}
