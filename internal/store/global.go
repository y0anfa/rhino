package store

import (
	"crypto/rand"
	"fmt"
	"sync"
	"time"
)

var (
	globalStore Store
	globalMu    sync.RWMutex
)

func Init(dbPath string) error {
	s, err := NewSQLiteStore(dbPath)
	if err != nil {
		return err
	}
	globalMu.Lock()
	globalStore = s
	globalMu.Unlock()
	return nil
}

func Global() Store {
	globalMu.RLock()
	defer globalMu.RUnlock()
	return globalStore
}

func CloseGlobal() {
	globalMu.Lock()
	defer globalMu.Unlock()
	if globalStore != nil {
		globalStore.Close()
		globalStore = nil
	}
}

// NewID generates a time-ordered unique ID (simplified ULID-like).
func NewID() string {
	ms := time.Now().UnixMilli()
	b := make([]byte, 5)
	rand.Read(b)
	return fmt.Sprintf("%011x%x", ms, b)
}
