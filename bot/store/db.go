package store

import (
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
	//flushing atomic.Bool
}

// Initializes a MapDB for a given map directory.
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

func (mdb *MapDB) Dir() string {
	return filepath.Clean(mdb.dir)
}

// Calls appropriate method on all stores in this DB that flushes data to their associated file.
func (mdb *MapDB) Flush() error {
	// TODO: WE NEED TO MAKE SURE FLUSHES DONT OVERLAP.

	for _, s := range mdb.stores {
		if err := s.WriteSnapshot(); err != nil {
			return err
		}
	}

	return nil
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
