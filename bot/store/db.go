package store

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

var mapDbs = make(map[string]*MapDB)
var mu sync.RWMutex // Guards access to mapDatabases

type MapDB struct {
	dir     string            // Path (relative to cwd) to the dir where this db lives.
	stores  map[string]IStore // Store file name → typed Store.
	storeMu sync.RWMutex      // Guards access to `stores`.
	flushMu sync.Mutex        // Ensures multiple flushes cannot happen simultaneously.
}

// Initializes a MapDB by reading the dir at baseDir+mapName (created if it does not exist) and registers it into global map.
//
// Once complete, we can add a store to the DB by calling DefineStore with the appropriate type which the store file can be unmarshaled into.
func NewMapDB(baseDir string, mapName string) (*MapDB, error) {
	dir := filepath.Join(baseDir, mapName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}

	db := &MapDB{
		dir:    dir,
		stores: make(map[string]IStore),
	}

	// put into mapDatabases
	RegisterMapDB(mapName, db)

	return db, nil
}

// The clean path to the dir of this MapDB which all store files live under.
func (mdb *MapDB) Dir() string {
	return filepath.Clean(mdb.dir)
}

// Calls WriteSnapshot on every store in this DB, flushing its current state to its associated file.
// A mutex lock is acquired before the loop, ensuring no two flushes can run simultaneously.
func (mdb *MapDB) Flush() error {
	errs := []error{}

	mdb.flushMu.Lock()
	defer mdb.flushMu.Unlock()

	for name, s := range mdb.stores {
		if err := s.WriteSnapshot(); err != nil {
			errs = append(errs, fmt.Errorf("store %s: %w", name, err))
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	fmt.Printf("\nDEBUG | Successfully flushed all stores to disk.\n")
	return nil
}

func RegisterMapDB(name string, mdb *MapDB) {
	mu.Lock()
	defer mu.Unlock()
	mapDbs[name] = mdb
}

func GetMapDB(name string) (*MapDB, error) {
	mu.RLock()
	defer mu.RUnlock()

	mdb, ok := mapDbs[name]
	if !ok {
		return nil, fmt.Errorf("failed to get MapDB. no such db exists: %s", name)
	}

	return mdb, nil
}

// Creates a new store an adds it to the given MapDB stores. Returns an error if the store already exists.
func DefineStore[T any](mdb *MapDB, name string) *Store[T] {
	mdb.storeMu.Lock()
	defer mdb.storeMu.Unlock()

	if s, ok := mdb.stores[name]; ok {
		fmt.Printf("\nstore '%s' already defined", name)
		return s.(*Store[T])
	}

	path := filepath.Join(mdb.dir, name+".json")
	store, err := NewStore[T](path)
	if err != nil {
		fmt.Printf("\nfailed to create store '%s': %v", name, err)
		return nil
	}

	mdb.stores[name] = store
	return store
}

// Retrieves the Store for a specific file/db.
func GetStore[T any](db *MapDB, name string) (*Store[T], error) {
	db.storeMu.RLock()
	defer db.storeMu.RUnlock()

	s, ok := db.stores[name]
	if !ok {
		return nil, fmt.Errorf("could not find store '%s' in db: %s", name, db.dir)
	}

	return s.(*Store[T]), nil
}

func GetStoreForMap[T any](mapName, storeName string) (*Store[T], error) {
	mdb, err := GetMapDB(mapName)
	if err != nil {
		return nil, err
	}

	return GetStore[T](mdb, storeName)
}
