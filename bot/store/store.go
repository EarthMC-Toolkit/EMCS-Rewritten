package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
)

// Essentially a persistent cache.
// Each 'store' is backed by a JSON file aka database which the cache will be populated from when it is initialized (if the file exists).
//
// A Store has the ability to Get/Set values or create snapshots of the cache which can be written to the database.
type Store struct {
	dbPath string
	data   map[string][]byte
	mu     sync.RWMutex
}

// Creates a new store backed by a JSON file at `path` for persistence.
func NewStore(path string) (*Store, error) {
	s := &Store{
		dbPath: path,
		data:   make(map[string][]byte),
	}

	if err := s.LoadFromDB(); err != nil {
		return nil, fmt.Errorf("failed to load store from db: %w", err)
	}

	return s, nil
}

func (s *Store) LoadFromDB() error {
	data, err := os.ReadFile(s.dbPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil // no db file yet
		}

		return err
	}

	tmp := make(map[string][]byte)
	if err := json.Unmarshal(data, &tmp); err != nil {
		return fmt.Errorf("invalid JSON in %s: %w", s.dbPath, err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.data = tmp
	return nil
}

// Retrieves a value from the cache.
func (s *Store) Get(key string) []byte {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// No key, no need to copy. Exit early.
	val := s.data[key]
	if val == nil {
		return nil
	}

	// stop da caller from mutating store
	cpy := make([]byte, len(val))
	copy(cpy, val)

	return cpy
}

// Put or overwrite the value in the store at the given key.
//
// WARNING: The value is copied before storing to prevent external mutations. Keep this in mind if reusing slices elsewhere.
func (s *Store) Set(key string, value []byte) {
	cpy := make([]byte, len(value))
	copy(cpy, value)

	s.mu.Lock()
	s.data[key] = cpy
	s.mu.Unlock()
}

// Creates a snapshot of the current cache state and writes it to the
// database (JSON file) at the path we provided when the store was initialized.
func (s *Store) WriteSnapshot() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// yankee wit no brim
	data, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return err
	}

	tmp := s.dbPath + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}

	return os.Rename(tmp, s.dbPath)
}
