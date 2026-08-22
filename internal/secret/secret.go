// Package secret stores credentials chit needs (currently just Linear's
// personal API key) in the OS's native secret store — Secret Service on
// Linux, Keychain on macOS, Credential Manager on Windows — falling back to
// an encrypted file vault when no OS backend is available. GitHub
// credentials are never handled here: chit shells out to the `gh` CLI,
// which owns its own token storage entirely.
package secret

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/99designs/keyring"

	"github.com/bjcorder/chit/internal/config"
)

const serviceName = "chit"

// ErrNotFound is returned by Store.Get when key has no stored value.
var ErrNotFound = errors.New("secret: key not found")

// Store is a minimal wrapper over keyring.Keyring, narrowed to the get/set/
// delete operations chit actually needs.
type Store struct {
	kr keyring.Keyring
}

// newStore wraps an already-open keyring.Keyring — used by Open, and by
// tests to inject keyring.NewArrayKeyring instead of a real OS backend.
func newStore(kr keyring.Keyring) *Store {
	return &Store{kr: kr}
}

// Open opens the OS-native secret store, preferring the Secret Service /
// Keychain / Credential Manager backend and falling back to an encrypted
// file vault under chit's XDG data directory if none is available.
func Open() (*Store, error) {
	dataDir, err := config.DataDir()
	if err != nil {
		return nil, fmt.Errorf("resolving data dir: %w", err)
	}

	kr, err := keyring.Open(keyring.Config{
		ServiceName: serviceName,
		AllowedBackends: []keyring.BackendType{
			keyring.SecretServiceBackend,
			keyring.KeychainBackend,
			keyring.WinCredBackend,
			keyring.FileBackend,
		},
		LibSecretCollectionName: serviceName,
		FileDir:                 filepath.Join(dataDir, "keyring"),
		FilePasswordFunc:        keyring.TerminalPrompt,
	})
	if err != nil {
		return nil, fmt.Errorf("opening secret store: %w", err)
	}
	return newStore(kr), nil
}

// Get returns the stored value for key, or ErrNotFound if none exists.
func (s *Store) Get(key string) (string, error) {
	item, err := s.kr.Get(key)
	if err != nil {
		if errors.Is(err, keyring.ErrKeyNotFound) {
			return "", ErrNotFound
		}
		return "", fmt.Errorf("getting %q: %w", key, err)
	}
	return string(item.Data), nil
}

// Set stores value under key, overwriting any existing value.
func (s *Store) Set(key, value string) error {
	if err := s.kr.Set(keyring.Item{
		Key:   key,
		Data:  []byte(value),
		Label: fmt.Sprintf("%s: %s", serviceName, key),
	}); err != nil {
		return fmt.Errorf("setting %q: %w", key, err)
	}
	return nil
}

// Delete removes key. It is not an error if key doesn't exist.
func (s *Store) Delete(key string) error {
	if err := s.kr.Remove(key); err != nil && !errors.Is(err, keyring.ErrKeyNotFound) {
		return fmt.Errorf("deleting %q: %w", key, err)
	}
	return nil
}
