package database

import (
	"emcsrw/database/store"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

var mapDbs = make(map[string]*Database)
var mu sync.RWMutex // Guards access to mapDatabases

// A database that is responsible for multiple persistent caches aka "stores"
// which can be assigned to this database and then retrieved for use again later.
// In addition, it handles read/write conflicts via mutexes to prevent data being lost or scrambled
// between stores and ensures old entries cannot overwrite new ones as they enter their JSON file
type Database struct {
	dirPath string                  // Path (relative to cwd) to the dir where this db lives.
	stores  map[string]store.IStore // Store file name → typed Store.
	storeMu sync.RWMutex            // Guards access to `stores`.
	flushMu sync.Mutex              // Ensures multiple flushes cannot happen simultaneously.
}

// Creates an instance of [Database], then reads the dir at baseDir+mapName (created if it does not exist) and registers it into global map.
//
// NOTE: To add a store to this DB, call [AssignStore] with the appropriate type which the store file can be unmarshaled into.
func New(baseDir string, mapName string) (*Database, error) {
	dir := filepath.Join(baseDir, mapName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}

	db := &Database{
		dirPath: dir,
		stores:  make(map[string]store.IStore),
	}

	// put into mapDatabases
	Register(mapName, db)

	return db, nil
}

// The clean path to the dir of this MapDB which all store files live under.
func (db *Database) Dir() string {
	return filepath.Clean(db.dirPath)
}

// Calls WriteSnapshot on every store in this DB, flushing its current state to its associated file.
// A mutex lock is acquired before the loop, ensuring no two flushes can run simultaneously.
func (db *Database) Flush() error {
	errs := []error{}

	db.flushMu.Lock()
	defer db.flushMu.Unlock()

	for name, s := range db.stores {
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

func Register(name string, mdb *Database) {
	mu.Lock()
	defer mu.Unlock()
	mapDbs[name] = mdb
}

func Get(name string) (*Database, error) {
	mu.RLock()
	defer mu.RUnlock()

	mdb, ok := mapDbs[name]
	if !ok {
		return nil, fmt.Errorf("failed to get MapDB. no such db exists: %s", name)
	}

	return mdb, nil
}

// Creates a new store an adds it to the given MapDB stores. Returns an error if the store already exists.
func AssignStore[T any](mdb *Database, name string) *store.Store[T] {
	mdb.storeMu.Lock()
	defer mdb.storeMu.Unlock()

	if s, ok := mdb.stores[name]; ok {
		fmt.Printf("\nstore '%s' already defined", name)
		return s.(*store.Store[T])
	}

	fpath := filepath.Join(mdb.dirPath, name+".json")
	store, err := store.New[T](fpath)
	if err != nil {
		fmt.Printf("\nfailed to create store '%s': %v", name, err)
		return nil
	}

	mdb.stores[name] = store
	return store
}

// Retrieves the Store for a specific file/db.
func GetStore[T any](db *Database, name string) (*store.Store[T], error) {
	db.storeMu.RLock()
	defer db.storeMu.RUnlock()

	s, ok := db.stores[name]
	if !ok {
		return nil, fmt.Errorf("could not find store '%s' in db: %s", name, db.dirPath)
	}

	return s.(*store.Store[T]), nil
}

func GetStoreForMap[T any](mapName, storeName string) (*store.Store[T], error) {
	mdb, err := Get(mapName)
	if err != nil {
		return nil, err
	}

	return GetStore[T](mdb, storeName)
}
