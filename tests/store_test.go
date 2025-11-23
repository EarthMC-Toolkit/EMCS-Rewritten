package tests

import (
	"emcsrw/database"
	"emcsrw/database/store"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

const testDB = "testdb"
const testPersistDB = "testpersist"

var testStore = database.StoreDefinition[TestData]{Name: "teststore"}

type TestData struct {
	Names []string `json:"names"`
	Name  string   `json:"name"`
}

func setupDB(dbName string) (*database.Database, string, error) {
	dir := "./db/" + dbName
	os.RemoveAll(dir)

	mdb, err := database.New("./db/", dbName)
	if err != nil {
		return nil, dir, err
	}

	return mdb, dir, err
}

func setupTest(t *testing.T, dbName string) (*database.Database, string) {
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

func setupBench(b *testing.B) (*database.Database, *store.Store[TestData], string) {
	mdb, dir, err := setupDB(testDB)
	if err != nil {
		b.Fatal(err)
	}

	s := database.AssignStore(mdb, testStore)

	// Ensure cleanup after the whole benchmark completes
	b.Cleanup(func() {
		perOp := float64(b.Elapsed().Milliseconds()) / float64(b.N)
		if perOp > 0.1 {
			fmt.Printf("\nms/op: %.2f\n", perOp)
		}

		os.RemoveAll(dir)
	})

	return mdb, s, dir
}

func TestStorePersistence(t *testing.T) {
	mdb, dbDir := setupTest(t, testPersistDB)
	s := database.AssignStore(mdb, testStore)

	s.SetKey("key1", TestData{Name: "PersistentKey", Names: []string{"Persist1", "Persist2"}})
	if err := s.WriteSnapshot(); err != nil {
		t.Fatal(err)
	}

	// Reload a new store from the same path
	rs, err := store.New[TestData](fmt.Sprintf("%s.json", filepath.Join(dbDir, testStore.Name)))
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
	s := database.AssignStore(mdb, testStore)

	s.SetKey("key1", TestData{Name: "Test", Names: []string{"Test1", "Test2"}})
	v, err := s.GetKey("key1")
	if err != nil {
		t.Fatal(err)
	}

	if v.Name != "Test" {
		t.Errorf("expected Name=Test, got '%s'", v.Name)
	}
}

func BenchmarkSet(b *testing.B) {
	_, s, _ := setupBench(b)
	for i := 0; b.Loop(); i++ {
		s.SetKey(fmt.Sprintf("key%d", i), TestData{Name: "Benchmark", Names: []string{"Bench1", "Bench2"}})
	}
}

func BenchmarkGet(b *testing.B) {
	_, s, _ := setupBench(b)

	total := 100_000
	for i := range total {
		s.SetKey(fmt.Sprintf("key%d", i), TestData{Name: "Benchmark", Names: []string{"Bench1", "Bench2"}})
	}

	for i := 0; b.Loop(); i++ {
		if _, err := s.GetKey(fmt.Sprintf("key%d", i%total)); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSetSingle(b *testing.B) {
	_, s, _ := setupBench(b)

	key := "key1"
	for b.Loop() {
		s.SetKey(key, TestData{Name: "Benchmark", Names: []string{"Bench1", "Bench2"}})
	}
}

func BenchmarkGetSingle(b *testing.B) {
	_, s, _ := setupBench(b)

	key := "key1"
	s.SetKey(key, TestData{Name: "Benchmark", Names: []string{"Bench1", "Bench2"}})

	for b.Loop() {
		if _, err := s.GetKey(key); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkWriteSnapshot(b *testing.B) {
	_, s, _ := setupBench(b)

	// Fill the store with some data
	total := 20_000
	for i := range total {
		s.SetKey(fmt.Sprintf("key%d", i), TestData{Name: "Benchmark", Names: []string{"Bench1", "Bench2"}})
	}

	for b.Loop() {
		if err := s.WriteSnapshot(); err != nil {
			b.Fatal(err)
		}
	}
}
