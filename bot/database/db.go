package database

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dgraph-io/badger/v4"
)

var databases = make(map[string]*badger.DB)

func InitMapDB(mapName string) (*badger.DB, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	// Absolute path to the db for the input map.
	dbDir := filepath.Join(cwd, "db", mapName)

	opts := badger.DefaultOptions(dbDir)
	opts.ZSTDCompressionLevel = 2
	opts.NumLevelZeroTables = 1
	opts.NumVersionsToKeep = 1
	opts.CompactL0OnClose = true

	db, err := badger.Open(opts)
	if err != nil {
		return nil, err
	}

	databases[mapName] = db
	return db, nil
}

func GetMapDB(mapName string) *badger.DB {
	return databases[mapName]
}

func GetInsensitiveTxn[T any](txn *badger.Txn, key string) (out *T, err error) {
	out = new(T)

	item, err := txn.Get([]byte(strings.ToLower(key)))
	if err != nil {
		return
	}

	val, _ := item.ValueCopy(nil)
	err = json.Unmarshal(val, out)

	return
}

func GetInsensitive[T any](db *badger.DB, key string) (out *T, err error) {
	err = db.View(func(txn *badger.Txn) error {
		start := time.Now()
		out, err = GetInsensitiveTxn[T](txn, key)
		fmt.Printf("DEBUG | time to get key '%s' from db: %s\n", key, time.Since(start))
		return err
	})

	return
}

// Puts data into the DB at the specified key which is automatically lowercased.
//
// This func is a very simple wrapper around db.Update() and txn.Set().
// If any get call or data manipulation is required prior to txn.Set(), prefer a single transaction via db.Update().
func PutInsensitive(db *badger.DB, key string, data []byte) error {
	return db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte(strings.ToLower(key)), data)
	})
}
