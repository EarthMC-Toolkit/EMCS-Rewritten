package store

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

var mapDatabases = make(map[string]*MapDB)
var mut sync.RWMutex // guards access to mapDatabases

type IStore interface {
	Close() error
}

type MapDB struct {
	dir    string
	stores map[string]IStore // store file name → typed Store
	mut    sync.RWMutex      // guards access to stores
}

// Initializes a MapDB for a given map directory and dataset names.
// Example datasets: []string{"towns", "nations", "alliances"}
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

func AddStore[T any](mdb *MapDB, name string) (*Store[T], error) {
	path := filepath.Join(mdb.dir, name+".json")
	store, err := NewStore[T](path)
	if err != nil {
		return nil, fmt.Errorf("failed to create store %q: %w", name, err)
	}

	mdb.mut.Lock()
	defer mdb.mut.Unlock()

	mdb.stores[name] = store
	return store, nil
}

func RegisterMapDB(name string, mdb *MapDB) {
	mut.Lock()
	defer mut.Unlock()
	mapDatabases[name] = mdb
}

func GetMapDB(name string) (*MapDB, error) {
	mut.RLock()
	defer mut.RUnlock()

	mdb, ok := mapDatabases[name]
	if !ok {
		return nil, fmt.Errorf("failed to get MapDB. no db exists with name: %s", name)
	}

	return mdb, nil
}

// Close flushes all stores to disk.
func (mdb *MapDB) Close() error {
	for _, s := range mdb.stores {
		if err := s.Close(); err != nil {
			return err
		}
	}

	return nil
}
