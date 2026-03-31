package secrets

import (
	"path/filepath"
	"sort"
	"testing"
)

func testStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "secrets.enc")
	s, err := NewStore(path, "test-master-key")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	return s
}

func TestNewStore_MissingKey(t *testing.T) {
	t.Setenv(EnvSecretKey, "")
	_, err := NewStore("/tmp/test.enc", "")
	if err == nil {
		t.Error("expected error for missing key")
	}
}

func TestNewStore_EnvKey(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(EnvSecretKey, "env-key")
	s, err := NewStore(filepath.Join(dir, "secrets.enc"), "")
	if err != nil {
		t.Fatalf("expected success with env key, got: %v", err)
	}
	if s == nil {
		t.Fatal("expected non-nil store")
	}
}

func TestStore_SetGetDelete(t *testing.T) {
	s := testStore(t)

	if err := s.Set("api-key", "secret123"); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	val, ok := s.Get("api-key")
	if !ok {
		t.Fatal("expected to find api-key")
	}
	if val != "secret123" {
		t.Errorf("expected 'secret123', got '%s'", val)
	}

	_, ok = s.Get("nonexistent")
	if ok {
		t.Error("expected false for nonexistent key")
	}

	if err := s.Delete("api-key"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	_, ok = s.Get("api-key")
	if ok {
		t.Error("expected false after delete")
	}
}

func TestStore_List(t *testing.T) {
	s := testStore(t)
	s.Set("key1", "val1")
	s.Set("key2", "val2")

	list := s.List()
	sort.Strings(list)
	if len(list) != 2 || list[0] != "key1" || list[1] != "key2" {
		t.Errorf("expected [key1, key2], got %v", list)
	}
}

func TestStore_Persistence(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "secrets.enc")

	s1, _ := NewStore(path, "master-key")
	s1.Set("db-pass", "p@ss")

	// Reload from disk
	s2, err := NewStore(path, "master-key")
	if err != nil {
		t.Fatalf("failed to reload store: %v", err)
	}
	val, ok := s2.Get("db-pass")
	if !ok || val != "p@ss" {
		t.Errorf("expected 'p@ss' after reload, got '%s' (found=%v)", val, ok)
	}
}

func TestStore_WrongKey(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "secrets.enc")

	s1, _ := NewStore(path, "correct-key")
	s1.Set("secret", "value")

	_, err := NewStore(path, "wrong-key")
	if err == nil {
		t.Error("expected error with wrong key")
	}
}
