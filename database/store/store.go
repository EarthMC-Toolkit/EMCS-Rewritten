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

// The interface that a generic store must implement to retain basic functionality that is common across all stores.
// Once converted to a concrete store type, further type-specific operations may become available.
type IStore interface {
	CleanPath() string
	WriteSnapshot() error
	LoadFromFile() error
}

// The bridge between a generic store and concrete store. This way, we get both type-safe access
// to the underlying store, while retaining the ability to perform common operations that all stores implement.
//
// NOTE: This preserves the type so we can safely type-assert back to Store[T] since Go erases
// generic type parameters when storing a generic in an interface or map.
type WrappedStore[T any] struct {
	Store *Store[T]
}

func (w *WrappedStore[T]) CleanPath() string    { return w.Store.CleanPath() }
func (w *WrappedStore[T]) WriteSnapshot() error { return w.Store.WriteSnapshot() }
func (w *WrappedStore[T]) LoadFromFile() error  { return w.Store.LoadFromFile() }

type StoreKey = string
type StoreData[T any] map[StoreKey]T // Stores value not pointer. Use SetKey etc. to mutate data safely.

func (sd StoreData[T]) shallowCopy() StoreData[T] {
	return utils.CopyMap(sd)
}

// Essentially a persistent cache.
// Each 'store' is backed by a JSON file aka database which the cache will be populated from when it is initialized (if the file exists).
//
// A Store has the ability to Get/Set values or create snapshots of the cache which can be written to the database.
type Store[T any] struct {
	filePath string       // Path to the file/dataset for this store.
	data     StoreData[T] // The actual data within the file.
	mu       sync.RWMutex // Mutex lock to stop read & write collisions.
}

// Creates a new store backed by a JSON file at `path` for persistence.
// The path should be relative to the current working dir, i.e. "./db/map/alliances.json"
func New[T any](path string) (*Store[T], error) {
	s := &Store[T]{
		filePath: path,
		data:     make(map[StoreKey]T),
	}

	if err := s.LoadFromFile(); err != nil {
		return nil, fmt.Errorf("failed to load store from file: %w", err)
	}

	if !s.IsEmpty() {
		fmt.Printf("DEBUG | Loaded store from file at: %s\n", s.CleanPath())
	}

	//fmt.Printf("\nDEBUG | Loaded store from file at: %s\n", s.CleanPath())
	return s, nil
}

func (s *Store[T]) CleanPath() string {
	return filepath.Clean(s.filePath)
}

func (s *Store[T]) Entries() StoreData[T] {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.data.shallowCopy()
}

func (s *Store[T]) Values() (values []T) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, v := range s.data {
		values = append(values, v)
	}

	return
}

func (s *Store[T]) Overwrite(value StoreData[T]) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.data = value
}

func (s *Store[T]) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for k := range s.data {
		delete(s.data, k)
	}
}

func (s *Store[T]) IsEmpty() bool {
	return len(s.data) <= 0
}

// Deletes a value in this store that is associated with the key.
//
// This operation is case-sensitive. If you are storing keys insensitively,
// make sure the key is lowered before inputting to this func.
func (s *Store[T]) DeleteKey(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.data, key)
}

// Creates or overwrites the value in the store at the given key in a thread-safe manner.
func (s *Store[T]) SetKey(key string, value T) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.data[key] = value
}

// Retrieves a value from this store that is associated with the key.
//
// This operation is case-sensitive. If you are storing keys insensitively,
// make sure the key is lowered before inputting to this func.
func (s *Store[T]) GetKey(key string) (*T, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if v, ok := s.data[key]; ok {
		return &v, nil
	}

	return nil, fmt.Errorf("could not get value for key '%s' from store: %s. no such key exists", key, s.CleanPath())
}

func (s *Store[T]) HasKey(key string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	_, ok := s.data[key]
	return ok
}

func (s *Store[T]) GetMany(keys ...string) []T {
	s.mu.RLock()
	defer s.mu.RUnlock()

	results := make([]T, 0, len(keys))
	for _, key := range keys {
		if v, ok := s.data[key]; ok {
			results = append(results, v)
		}
	}

	return results
}

func (s *Store[T]) FindFirst(predicate func(value T) bool) (*T, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, v := range s.data {
		if predicate(v) {
			return &v, nil
		}
	}

	return nil, fmt.Errorf("no matching value found in store: %s", s.CleanPath())
}

func (s *Store[T]) FindMany(predicate func(value T) bool) []T {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var results []T
	for _, v := range s.data {
		if predicate(v) {
			results = append(results, v)
		}
	}

	return results
}

// Overwrite the current store cache state with data from the associated JSON file/database located at path.
// This should usually be called when the cache is empty and needs fresh data, for example when the bot starts up or when we are restoring from a backup.
// This function should never be called during normal operation as to not provide potentially stale data.
func (s *Store[T]) LoadFromFile() error {
	contents, err := os.ReadFile(s.CleanPath())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}

		return err
	}

	var data map[StoreKey]T
	if err := json.Unmarshal(contents, &data); err != nil {
		return err
	}

	s.Overwrite(data)
	return nil
}

// Creates a snapshot of the current cache state and writes it to the
// database (JSON file) at the path we provided when the store was initialized.
func (s *Store[T]) WriteSnapshot() error {
	s.mu.RLock()
	cpy := s.data.shallowCopy()
	s.mu.RUnlock()

	data, err := json.Marshal(cpy)
	if err != nil {
		return err
	}

	// yankee wit no brim
	tmp := s.filePath + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}

	// replace real file once temp file is fully written
	err = os.Rename(tmp, s.filePath)
	if err != nil {
		// TODO: if this occurs, the temp file could be left behind
		// we should check this file exists and either recover or delete it
		return fmt.Errorf("error writing store snapshot to %s: %w", s.filePath, err)
	}

	//fmt.Printf("Successfully closed store and wrote snapshot at: %s\n", s.filePath)
	return nil
}
