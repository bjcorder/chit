package secret

import (
	"errors"
	"testing"

	"github.com/99designs/keyring"
)

func TestSetGetRoundTrips(t *testing.T) {
	s := newStore(keyring.NewArrayKeyring(nil))

	if err := s.Set("linear-api-key", "lin_api_abc123"); err != nil {
		t.Fatalf("Set: %v", err)
	}

	got, err := s.Get("linear-api-key")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != "lin_api_abc123" {
		t.Errorf("Get() = %q, want %q", got, "lin_api_abc123")
	}
}

func TestGetMissingKeyReturnsErrNotFound(t *testing.T) {
	s := newStore(keyring.NewArrayKeyring(nil))

	if _, err := s.Get("does-not-exist"); !errors.Is(err, ErrNotFound) {
		t.Errorf("Get() error = %v, want ErrNotFound", err)
	}
}

func TestSetOverwritesExistingValue(t *testing.T) {
	s := newStore(keyring.NewArrayKeyring(nil))

	if err := s.Set("linear-api-key", "old"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if err := s.Set("linear-api-key", "new"); err != nil {
		t.Fatalf("Set (overwrite): %v", err)
	}

	got, err := s.Get("linear-api-key")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != "new" {
		t.Errorf("Get() = %q, want %q", got, "new")
	}
}

func TestDeleteRemovesValue(t *testing.T) {
	s := newStore(keyring.NewArrayKeyring(nil))

	if err := s.Set("linear-api-key", "value"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if err := s.Delete("linear-api-key"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := s.Get("linear-api-key"); !errors.Is(err, ErrNotFound) {
		t.Errorf("Get() after Delete error = %v, want ErrNotFound", err)
	}
}

func TestDeleteMissingKeyIsNotAnError(t *testing.T) {
	s := newStore(keyring.NewArrayKeyring(nil))

	if err := s.Delete("does-not-exist"); err != nil {
		t.Errorf("Delete of missing key returned error: %v", err)
	}
}
