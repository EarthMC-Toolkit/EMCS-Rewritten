package database

import (
	"emcsrw/bot/common"
	"os"
	"path/filepath"

	"github.com/dgraph-io/badger/v4"
)

var databases = make(map[string]*badger.DB)

func InitMapDB(mapName common.EMCMap) (*badger.DB, error) {
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

func GetMapDB(mapName common.EMCMap) *badger.DB {
	return databases[mapName]
}
