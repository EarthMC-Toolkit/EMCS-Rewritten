package store

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

var mapDbs = make(map[string]*MapDB)
var mut sync.RWMutex // guards access to mapDatabases

type MapDB struct {
	dir    string
	stores map[string]IStore // store file name → typed Store
	mut    sync.RWMutex      // guards access to stores
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

// Creates a new store an adds it to the given MapDB stores. Returns an error if the store already exists.
func DefineStore[T any](mdb *MapDB, name string) *Store[T] {
	mdb.mut.Lock()
	defer mdb.mut.Unlock()

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

func RegisterMapDB(name string, mdb *MapDB) {
	mut.Lock()
	defer mut.Unlock()
	mapDbs[name] = mdb
}

func GetMapDB(name string) (*MapDB, error) {
	mut.RLock()
	defer mut.RUnlock()

	mdb, ok := mapDbs[name]
	if !ok {
		return nil, fmt.Errorf("failed to get MapDB. no such db exists: %s", name)
	}

	return mdb, nil
}

// Calls Close() method on all stores in this DB, flushing data to their associated file.
func (mdb *MapDB) Close() error {
	for _, s := range mdb.stores {
		if err := s.Close(); err != nil {
			return err
		}
	}

	return nil
}
