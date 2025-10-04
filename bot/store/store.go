package store

import (
	"emcsrw/utils"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// Specifies the basic functionality a Store should have, regardless
// of how it is interacted with via its other methods.
type IStore interface {
	CleanPath() string
	WriteSnapshot() error
	LoadFromFile() error
}

type StoreKey = string

// Essentially a persistent cache.
// Each 'store' is backed by a JSON file aka database which the cache will be populated from when it is initialized (if the file exists).
//
// A Store has the ability to Get/Set values or create snapshots of the cache which can be written to the database.
type Store[T any] struct {
	filePath string         // Path to the file/dataset for this store.
	data     map[StoreKey]T // The actual data within the file.
	mu       sync.RWMutex   // Mutex lock to stop read & write collisions.
}

// Creates a new store backed by a JSON file at `path` for persistence.
// The path should be relative to the current working dir, i.e. "./db/map/alliances.json"
func NewStore[T any](path string) (*Store[T], error) {
	s := &Store[T]{
		filePath: path,
		data:     make(map[StoreKey]T),
	}

	if err := s.LoadFromFile(); err != nil {
		return nil, fmt.Errorf("failed to load store from file: %w", err)
	}

	if len(s.data) > 0 {
		fmt.Printf("DEBUG | Loaded store from file at: %s\n", s.CleanPath())
	}

	//fmt.Printf("\nDEBUG | Loaded store from file at: %s\n", s.CleanPath())
	return s, nil
}

func (s *Store[T]) CleanPath() string {
	return filepath.Clean(s.filePath)
}

func (s *Store[T]) All() map[StoreKey]T {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return utils.CopyMap(s.data)
}

func (s *Store[T]) Overwrite(value map[StoreKey]T) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.data = value
}

func (s *Store[T]) GetKey(key string) (*T, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if v, ok := s.data[key]; ok {
		return &v, nil
	}

	return nil, fmt.Errorf("could not get value for key '%s' from store: %s. no such key exists", key, s.CleanPath())
}

// Create or overwrite the value in the store at the given key.
func (s *Store[T]) SetKey(key string, value T) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.data[key] = value
}

func (s *Store[T]) DeleteKey(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.data, key)
}

// Overwrite the current store cache state with data from the associated JSON file/database located at path.
// This should usually be called when the cache is empty and needs fresh data, for example when the bot starts up or when we are restoring from a backup.
// This function should never be called during normal operation as to not provide potentially stale data.
func (s *Store[T]) LoadFromFile() error {
	data, err := os.ReadFile(s.CleanPath())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}

		return err
	}

	var v map[StoreKey]T
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	s.Overwrite(v)
	return nil
}

// Creates a snapshot of the current cache state and writes it to the
// database (JSON file) at the path we provided when the store was initialized.
func (s *Store[T]) WriteSnapshot() error {
	s.mu.RLock()
	cpy := utils.CopyMap(s.data)
	s.mu.RUnlock()

	data, err := json.Marshal(cpy)
	if err != nil {
		return err
	}

	tmp := s.filePath + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}

	// yankee wit no brim
	err = os.Rename(tmp, s.filePath)
	if err != nil {
		return fmt.Errorf("error writing store snapshot to %s: %w", s.filePath, err)
	}

	//fmt.Printf("Successfully closed store and wrote snapshot at: %s\n", s.filePath)
	return nil
}
