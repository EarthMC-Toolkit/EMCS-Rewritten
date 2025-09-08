package tests

import (
	"emcsrw/bot/database"
	"testing"

	"github.com/dgraph-io/badger/v4"
)

func TestDumpAlliances(t *testing.T) {
	db, err := badger.Open(badger.DefaultOptions("../db/aurora"))
	if err != nil {
		t.Fatal(err)
	}

	err = database.DumpAlliances(db)
	if err != nil {
		t.Fatal(err)
	}
}
