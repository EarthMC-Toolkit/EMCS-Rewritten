package database

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

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

		val, _ := item.ValueCopy(nil)
		return json.Unmarshal(val, &a)
	})

	if err != nil {
		return nil, err
	}

	return &a, nil
}
