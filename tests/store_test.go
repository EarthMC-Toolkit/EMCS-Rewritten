package tests

import (
	"emcsrw/bot/store"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

const testDB = "testdb"
const testPersistDB = "testpersist"
const testStore = "teststore"

type TestData struct {
	Names []string `json:"names"`
	Name  string   `json:"name"`
}

func setupDB(dbName string) (*store.MapDB, string, error) {
	dir := "./db/" + dbName
	os.RemoveAll(dir)

	mdb, err := store.NewMapDB("./db/", dbName)
	if err != nil {
		return nil, dir, err
	}

	return mdb, dir, err
}

func setupTest(t *testing.T, dbName string) (*store.MapDB, string) {
	mdb, dir, err := setupDB(dbName)
	if err != nil {
		t.Fatal(err)
	}

	// Ensure cleanup after the whole test completes
	t.Cleanup(func() {
		os.RemoveAll(dir)
	})

	return mdb, dir
}

func setupBench(b *testing.B) (*store.MapDB, *store.Store[TestData], string) {
	mdb, dir, err := setupDB(testDB)
	if err != nil {
		b.Fatal(err)
	}

	s := store.AssignStoreToDB[TestData](mdb, testStore)

	// Ensure cleanup after the whole benchmark completes
	b.Cleanup(func() {
		os.RemoveAll(dir)
	})

	return mdb, s, dir
}

func TestStorePersistence(t *testing.T) {
	mdb, dbDir := setupTest(t, testPersistDB)
	s := store.AssignStoreToDB[TestData](mdb, testStore)

	s.SetKey("key1", TestData{Name: "PersistentKey", Names: []string{"Persist1", "Persist2"}})
	if err := s.WriteSnapshot(); err != nil {
		t.Fatal(err)
	}

	// Reload a new store from the same path
	rs, err := store.NewStore[TestData](fmt.Sprintf("%s.json", filepath.Join(dbDir, testStore)))
	if err != nil {
		t.Fatal(err)
	}

	v, _ := rs.GetKey("key1")
	if v == nil {
		t.Fatalf("expected key1 to exist after reload")
	}

	if v.Name != "PersistentKey" {
		t.Errorf("expected Name=PersistentKey, got %s", v.Name)
	}
}

func TestSetGet(t *testing.T) {
	mdb, _ := setupTest(t, testDB)
	s := store.AssignStoreToDB[TestData](mdb, testStore)

	s.SetKey("key1", TestData{Name: "Test", Names: []string{"Test1", "Test2"}})
	v, err := s.GetKey("key1")
	if err != nil {
		t.Fatal(err)
	}

	if v.Name != "Test" {
		t.Errorf("expected Name=Test, got '%s'", v.Name)
	}
}

func BenchmarkAllianceSet(b *testing.B) {
	_, s, _ := setupBench(b)
	for i := 0; b.Loop(); i++ {
		s.SetKey(fmt.Sprintf("key%d", i), TestData{Name: "Benchmark", Names: []string{"Bench1", "Bench2"}})
	}
}

func BenchmarkAllianceGet(b *testing.B) {
	_, s, _ := setupBench(b)

	for i := range 100000 {
		s.SetKey(fmt.Sprintf("key%d", i), TestData{Name: "Benchmark", Names: []string{"Bench1", "Bench2"}})
	}

	for i := 0; b.Loop(); i++ {
		key := fmt.Sprintf("key%d", i%100000) // cycle through keys
		if _, err := s.GetKey(key); err != nil {
			b.Fatal(err)
		}
	}
}
