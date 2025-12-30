package store

import (
	"emcsrw/utils"
	"emcsrw/utils/sets"
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

type StoreKey = string
type StoreData[T any] map[StoreKey]T // Stores value not pointer. Use SetKey etc. to mutate data safely.

func (sd StoreData[T]) shallowCopy() StoreData[T] {
	return utils.CopyMap(sd)
}

// Essentially a persistent cache that can be interfaced with like a KV store.
//
// Each 'store' is backed by a JSON file aka database which the cache will be populated from when it is initialized (if the file exists).
// From there on, all operations are done in-memory and the current state can be saved to the file on demand.
//
// The store is thread-safe and can be used concurrently across multiple goroutines.
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

func (s *Store[T]) Keys() (keys []StoreKey) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for k := range s.data {
		keys = append(keys, k)
	}

	return
}

func (s *Store[T]) Values() (values []T) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, v := range s.data {
		values = append(values, v)
	}

	return
}

func (s *Store[T]) Entries() StoreData[T] {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.data.shallowCopy()
}

// Similar to Entries(), this func will return a map, with the key being customizable based on keyFunc.
//
// This is useful for creating maps, where the key is based on a specific field of the stored value type.
func (s *Store[T]) EntriesFunc(f func(value T) string) map[string]T {
	vals := s.Values()
	out := make(map[string]T, len(vals))
	for _, v := range vals {
		out[f(v)] = v
	}

	return out
}

// Gets both entries and values in a single pass.
func (s *Store[T]) EntriesAndValues() (entries StoreData[T], values []T) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entries = make(StoreData[T], s.Count())
	values = make([]T, 0, s.Count())
	for k, v := range s.data {
		entries[k] = v
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
	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.data) == 0
}

func (s *Store[T]) Count() int {
	return len(s.data)
}

// Deletes a value in this store that is associated with the key.
//
// This operation is case-sensitive. If you are storing keys insensitively,
// make sure the key is lowered before inputting to this func.
func (s *Store[T]) Delete(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.data, key)
}

// Checks whether the store has a value associated with the given key.
func (s *Store[T]) HasKey(key string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	_, ok := s.data[key]
	return ok
}

// Creates or overwrites the value in the store at the given key in a thread-safe manner.
func (s *Store[T]) Set(key string, value T) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.data[key] = value
}

// Retrieves a value from this store that is associated with the key.
//
// This operation is case-sensitive. If you are storing keys insensitively,
// make sure the key is lowered before inputting to this func.
func (s *Store[T]) Get(key string) (*T, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if v, ok := s.data[key]; ok {
		return &v, nil
	}

	return nil, fmt.Errorf("could not get value for key '%s' from store: %s. no such key exists", key, s.CleanPath())
}

func (s *Store[T]) GetFunc(predicate func(key StoreKey) bool) (*T, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for k, v := range s.data {
		if predicate(k) {
			return &v, nil
		}
	}

	return nil, fmt.Errorf("no matching key found in store: %s", s.CleanPath())
}

func (s *Store[T]) GetFromSet(set sets.StringSet) (results []T) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for k := range set {
		if v, ok := s.data[k]; ok {
			results = append(results, v)
		}
	}

	return results
}

// func (s *Store[T]) GetMulti(keys ...string) (results []T) {
// 	s.mu.RLock()
// 	defer s.mu.RUnlock()

// 	for _, key := range keys {
// 		if v, ok := s.data[key]; ok {
// 			results = append(results, v)
// 		}
// 	}

// 	return results
// }

func (s *Store[T]) Find(predicate func(value T) bool) (*T, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, v := range s.data {
		if predicate(v) {
			return &v, nil
		}
	}

	return nil, fmt.Errorf("no matching value found in store: %s", s.CleanPath())
}

func (s *Store[T]) FindAll(predicate func(value T) bool) (results []T) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, v := range s.data {
		if predicate(v) {
			results = append(results, v)
		}
	}

	return
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

	// using a copy prevents a panic if map is modified when marshal iterates it
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
