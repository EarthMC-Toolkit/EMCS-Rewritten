package database

import (
	"emcsrw/api/oapi"
	"emcsrw/database/store"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

type StoreDefinition[T any] struct {
	Name string
}

var SERVER_STORE = StoreDefinition[oapi.ServerInfo]{Name: "server"}
var TOWNS_STORE = StoreDefinition[oapi.TownInfo]{Name: "towns"}
var NATIONS_STORE = StoreDefinition[oapi.NationInfo]{Name: "nations"}
var ENTITIES_STORE = StoreDefinition[oapi.EntityList]{Name: "entities"}
var ALLIANCES_STORE = StoreDefinition[Alliance]{Name: "alliances"}
var USAGE_USERS_STORE = StoreDefinition[UserUsage]{Name: "usage-users"}

var databases = make(map[string]*Database)
var mu sync.RWMutex // Guards access to databases

// A database that is responsible for multiple persistent caches aka "stores"
// which can be assigned to this database and then retrieved for use again later.
// In addition, it handles read/write conflicts via mutexes to prevent data being lost or scrambled
// between stores and ensures old entries cannot overwrite new ones as they enter their JSON file.
type Database struct {
	dirPath string                  // Path (relative to cwd) to the dir where this db lives.
	stores  map[string]store.IStore // Mapping from file name → generic Store instance.
	storeMu sync.RWMutex            // Guards access to `stores`.
	flushMu sync.Mutex              // Ensures multiple flushes cannot happen simultaneously.
}

// Creates an instance of [Database] with the dir at baseDir+mapName (created if it does not exist) and registers it into global map.
//
// NOTE: To add a store to this DB, call [AssignStore] with the appropriate type which the store file can be unmarshalled into.
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
	databases[name] = mdb
}

func Get(name string) (*Database, error) {
	mu.RLock()
	defer mu.RUnlock()

	mdb, ok := databases[name]
	if !ok {
		return nil, fmt.Errorf("failed to get MapDB. no such db exists: %s", name)
	}

	return mdb, nil
}

// Creates a new store an adds it to the given MapDB stores. Returns an error if the store already exists.
func AssignStore[T any](db *Database, storeDef StoreDefinition[T]) *store.Store[T] {
	db.storeMu.Lock()
	defer db.storeMu.Unlock()

	if s, ok := db.stores[storeDef.Name]; ok {
		fmt.Printf("\nstore '%s' already defined", storeDef.Name)
		return s.(*store.Store[T])
	}

	fpath := filepath.Join(db.dirPath, storeDef.Name+".json")
	store, err := store.New[T](fpath)
	if err != nil {
		fmt.Printf("\nfailed to create store '%s': %v", storeDef.Name, err)
		return nil
	}

	db.stores[storeDef.Name] = store
	return store
}

// Retrieves the Store for a specific file/db.
func GetStore[T any](db *Database, storeDef StoreDefinition[T]) (*store.Store[T], error) {
	db.storeMu.RLock()
	defer db.storeMu.RUnlock()

	si, ok := db.stores[storeDef.Name]
	if !ok {
		return nil, fmt.Errorf("could not find store '%s' in db: %s", storeDef.Name, db.dirPath)
	}

	s, ok := si.(*store.Store[T])
	if !ok {
		return nil, fmt.Errorf(
			"store '%s' exists but with a different type: expected *Store[%T], got %T",
			storeDef.Name, (*store.Store[T])(nil), si,
		)
	}

	return s, nil
}

func GetStoreForMap[T any](mapName string, store StoreDefinition[T]) (*store.Store[T], error) {
	mdb, err := Get(mapName)
	if err != nil {
		return nil, err
	}

	return GetStore(mdb, store)
}
