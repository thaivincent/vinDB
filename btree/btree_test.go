package btree_test

import (
	"bytes"
	"fmt"
	"os"
	"testing"

	"github.com/vince/vinDB/btree"
	"github.com/vince/vinDB/encoding"
	"github.com/vince/vinDB/storage"
)

// tempDB creates a temporary database file for testing and cleans it up when done.
func tempDB(t *testing.T) string {
	t.Helper()
	f, err := os.CreateTemp("", "vindb-btree-test-*.db")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	name := f.Name()
	f.Close()
	t.Cleanup(func() { os.Remove(name) })
	return name
}

func makeRow(id uint32, username, email string) []byte {
	r := &encoding.Row{
		ID:       id,
		Username: username,
		Email:    email,
	}
	b, err := r.SerializeRow()
	if err != nil {
		panic(err)
	}
	return b
}

func TestBTreeInsertAndFindSorted(t *testing.T) {
	path := tempDB(t)

	pg, err := storage.Open(path)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer pg.Close()

	tree, err := btree.NewBTree(pg, 0)
	if err != nil {
		t.Fatalf("NewBTree failed: %v", err)
	}

	// Insert keys out of order
	keysToInsert := []uint32{50, 10, 30, 20, 5}
	for _, k := range keysToInsert {
		val := makeRow(k, fmt.Sprintf("user%d", k), fmt.Sprintf("user%d@test.com", k))
		if err := tree.Insert(k, val); err != nil {
			t.Fatalf("failed to insert key %d: %v", k, err)
		}
	}

	// Verify all keys can be found and return the correct row
	for _, k := range keysToInsert {
		valBytes, err := tree.Find(k)
		if err != nil {
			t.Fatalf("failed to find key %d: %v", k, err)
		}

		row, err := encoding.DeserializeRow(valBytes)
		if err != nil {
			t.Fatalf("failed to deserialize row %d: %v", k, err)
		}

		if row.ID != k {
			t.Errorf("expected ID %d, got %d", k, row.ID)
		}
		expectedUser := fmt.Sprintf("user%d", k)
		if row.Username != expectedUser {
			t.Errorf("expected username %s, got %s", expectedUser, row.Username)
		}
	}

	// Verify non-existent keys return an error
	nonExistentKeys := []uint32{1, 15, 25, 999}
	for _, k := range nonExistentKeys {
		_, err := tree.Find(k)
		if err == nil {
			t.Errorf("expected error for non-existent key %d, got nil", k)
		}
	}
}

func TestBTreeRejectDuplicate(t *testing.T) {
	path := tempDB(t)

	pg, err := storage.Open(path)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer pg.Close()

	tree, err := btree.NewBTree(pg, 0)
	if err != nil {
		t.Fatalf("NewBTree failed: %v", err)
	}

	val := makeRow(10, "bob", "bob@test.com")
	if err := tree.Insert(10, val); err != nil {
		t.Fatalf("initial insert failed: %v", err)
	}

	// Duplicate insert should fail
	if err := tree.Insert(10, val); err == nil {
		t.Fatal("expected duplicate insert to return an error, got nil")
	}
}

func TestBTreePersistenceAcrossReopen(t *testing.T) {
	path := tempDB(t)

	// Step 1: Open, insert keys, close
	pg, err := storage.Open(path)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}

	tree, err := btree.NewBTree(pg, 0)
	if err != nil {
		t.Fatalf("NewBTree failed: %v", err)
	}

	keys := []uint32{7, 3, 9, 1}
	for _, k := range keys {
		val := makeRow(k, fmt.Sprintf("persisted%d", k), "p@test.com")
		if err := tree.Insert(k, val); err != nil {
			t.Fatalf("insert %d failed: %v", k, err)
		}
	}

	if err := pg.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// Step 2: Reopen in a new process simulation
	pg2, err := storage.Open(path)
	if err != nil {
		t.Fatalf("reopen failed: %v", err)
	}
	defer pg2.Close()

	tree2, err := btree.NewBTree(pg2, 0)
	if err != nil {
		t.Fatalf("NewBTree on reopen failed: %v", err)
	}

	for _, k := range keys {
		val, err := tree2.Find(k)
		if err != nil {
			t.Fatalf("Find key %d after reopen failed: %v", k, err)
		}

		expectedPrefix := fmt.Sprintf("persisted%d", k)
		if !bytes.Contains(val, []byte(expectedPrefix)) {
			t.Errorf("expected value to contain %s", expectedPrefix)
		}
	}
}

