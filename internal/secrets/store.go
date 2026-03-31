package secrets

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
)

const (
	EnvSecretKey = "RHINO_SECRET_KEY"
)

type Store struct {
	path string
	key  [32]byte
	data map[string]string
	mu   sync.RWMutex
}

func DefaultStorePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".rhino/secrets.enc"
	}
	return filepath.Join(home, ".rhino", "secrets.enc")
}

func NewStore(path string, masterKey string) (*Store, error) {
	if masterKey == "" {
		masterKey = os.Getenv(EnvSecretKey)
	}
	if masterKey == "" {
		return nil, fmt.Errorf("master key required: set %s or provide key", EnvSecretKey)
	}

	key := sha256.Sum256([]byte(masterKey))

	s := &Store{
		path: path,
		key:  key,
		data: make(map[string]string),
	}

	if err := s.load(); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to load secret store: %w", err)
	}

	return s, nil
}

func (s *Store) Set(name, value string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[name] = value
	return s.save()
}

func (s *Store) Get(name string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	val, ok := s.data[name]
	return val, ok
}

func (s *Store) Delete(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, name)
	return s.save()
}

func (s *Store) List() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	names := make([]string, 0, len(s.data))
	for name := range s.data {
		names = append(names, name)
	}
	return names
}

func (s *Store) load() error {
	ciphertext, err := os.ReadFile(s.path)
	if err != nil {
		return err
	}

	plaintext, err := s.decrypt(ciphertext)
	if err != nil {
		return fmt.Errorf("failed to decrypt secret store: %w", err)
	}

	return json.Unmarshal(plaintext, &s.data)
}

func (s *Store) save() error {
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create secret store directory: %w", err)
	}

	plaintext, err := json.Marshal(s.data)
	if err != nil {
		return err
	}

	ciphertext, err := s.encrypt(plaintext)
	if err != nil {
		return err
	}

	return os.WriteFile(s.path, ciphertext, 0600)
}

func (s *Store) encrypt(plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(s.key[:])
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

func (s *Store) decrypt(ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(s.key[:])
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	return gcm.Open(nil, nonce, ciphertext, nil)
}
