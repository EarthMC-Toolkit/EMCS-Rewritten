package tests

import (
	"emcsrw/bot/store"
	"fmt"
	"os"
	"testing"
	"time"
)

type TestAlliance struct {
	Name string `json:"name"`
}

func setupMapDB(t *testing.T) *store.MapDB {
	os.RemoveAll("../db/testmap")

	mdb, err := store.NewMapDB("./db/", "testmap")
	if err != nil {
		t.Fatal(err)
	}

	return mdb
}

func TestStorePersistence(t *testing.T) {
	// Setup MapDB folder for test
	dbDir := "./db/testpersist"
	os.RemoveAll(dbDir) // clean up old data

	mdb, err := store.NewMapDB("./db/", "testpersist")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dbDir)

	// Get typed store
	alliances, err := store.GetStore[TestAlliance](mdb, "alliances")
	if err != nil {
		t.Fatal(err)
	}

	// Set a value and write snapshot
	alliances.SetKey("alliance1", TestAlliance{Name: "PersistentAlliance"})
	if err := alliances.WriteSnapshot(); err != nil {
		t.Fatal(err)
	}

	// Reload a new store from the same path
	reloadedStore, err := store.NewStore[TestAlliance](fmt.Sprintf("%s/alliances.json", dbDir))
	if err != nil {
		t.Fatal(err)
	}

	a, _ := reloadedStore.GetKey("alliance1")
	if a == nil {
		t.Fatalf("expected alliance1 to exist after reload")
	}

	if a.Name != "PersistentAlliance" {
		t.Errorf("expected Name=PersistentAlliance, got %s", a.Name)
	}
}

func TestSetGetAlliance(t *testing.T) {
	start := time.Now()

	mdb := setupMapDB(t)
	defer mdb.Close()
	fmt.Printf("setupMapDB took %s\n", time.Since(start))

	start = time.Now()
	alliances, _ := store.GetStore[TestAlliance](mdb, "alliances")
	fmt.Printf("GetStore took %s\n", time.Since(start))

	start = time.Now()
	alliances.SetKey("alliance1", TestAlliance{Name: "TestAlliance"})
	fmt.Printf("Set took %s\n", time.Since(start))

	start = time.Now()
	a, _ := alliances.GetKey("alliance1")
	fmt.Printf("Get took %s\n", time.Since(start))

	if a.Name != "TestAlliance" {
		t.Errorf("expected TestAlliance, got %s", a.Name)
	}
}

func setupStoreForBenchmark[T any](key string) *store.Store[T] {
	mdb, _ := store.NewMapDB("./db/", "testmap")
	return store.AssignStoreToDB[T](mdb, key)
}

func BenchmarkAllianceSet(b *testing.B) {
	allianceStore := setupStoreForBenchmark[TestAlliance]("alliances")
	for i := 0; b.Loop(); i++ {
		allianceStore.SetKey(fmt.Sprintf("alliance%d", i), TestAlliance{Name: "Benchmark"})
	}
}
