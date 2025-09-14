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

	dbDir := filepath.Join(cwd, "db", mapName)
	db, err := badger.Open(badger.DefaultOptions(dbDir))
	if err != nil {
		return nil, err
	}

	databases[mapName] = db
	return db, nil
}

func GetMapDB(mapName string) *badger.DB {
	return databases[mapName]
}

func PutInsensitive(db *badger.DB, key string, data []byte) error {
	return db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte(strings.ToLower(key)), data)
	})
}

func GetInsensitive[T any](db *badger.DB, key string) (*T, error) {
	var a T

	err := db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(strings.ToLower(key)))
		if err != nil {
			return err
		}

		start := time.Now()

		val, _ := item.ValueCopy(nil)
		err = json.Unmarshal(val, &a)

		elapsed := time.Since(start)
		fmt.Printf("DEBUG: time to copy and unmarshal '%s' value: %s\n", key, elapsed)

		return err
	})

	if err != nil {
		return nil, err
	}

	return &a, nil
}
