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

	keysToInsert := []uint32{50, 10, 30, 20, 5}
	for _, k := range keysToInsert {
		val := makeRow(k, fmt.Sprintf("user%d", k), fmt.Sprintf("user%d@test.com", k))
		if err := tree.Insert(k, val); err != nil {
			t.Fatalf("failed to insert key %d: %v", k, err)
		}
	}

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

	if err := tree.Insert(10, val); err == nil {
		t.Fatal("expected duplicate insert to return an error, got nil")
	}
}

func TestBTreePersistenceAcrossReopen(t *testing.T) {
	path := tempDB(t)

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

func TestBTreeRootSplitWith20Keys(t *testing.T) {
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

	// Insert 20 keys (capacity of single leaf is 13, so this forces a root split!)
	for i := uint32(1); i <= 20; i++ {
		val := makeRow(i, fmt.Sprintf("user%d", i), fmt.Sprintf("user%d@db.com", i))
		if err := tree.Insert(i, val); err != nil {
			t.Fatalf("insert %d failed: %v", i, err)
		}
	}

	// Verify all 20 keys can be found correctly across the split tree!
	for i := uint32(1); i <= 20; i++ {
		valBytes, err := tree.Find(i)
		if err != nil {
			t.Fatalf("failed to find key %d after split: %v", i, err)
		}

		row, err := encoding.DeserializeRow(valBytes)
		if err != nil {
			t.Fatalf("deserialize failed for key %d: %v", i, err)
		}

		if row.ID != i {
			t.Errorf("expected ID %d, got %d", i, row.ID)
		}
		expectedUser := fmt.Sprintf("user%d", i)
		if row.Username != expectedUser {
			t.Errorf("expected username %s, got %s", expectedUser, row.Username)
		}
	}
}

func TestBTreeChildLeafSplitWith50Keys(t *testing.T) {
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

	// Insert 50 sequential keys (forces root split and multiple child leaf splits!)
	for i := uint32(1); i <= 50; i++ {
		val := makeRow(i, fmt.Sprintf("user%d", i), fmt.Sprintf("user%d@db.com", i))
		if err := tree.Insert(i, val); err != nil {
			t.Fatalf("insert %d failed: %v", i, err)
		}
	}

	// Verify all 50 keys can be retrieved accurately
	for i := uint32(1); i <= 50; i++ {
		valBytes, err := tree.Find(i)
		if err != nil {
			t.Fatalf("failed to find key %d after multiple child splits: %v", i, err)
		}

		row, err := encoding.DeserializeRow(valBytes)
		if err != nil {
			t.Fatalf("deserialize failed for key %d: %v", i, err)
		}

		if row.ID != i {
			t.Errorf("expected ID %d, got %d", i, row.ID)
		}
		expectedUser := fmt.Sprintf("user%d", i)
		if row.Username != expectedUser {
			t.Errorf("expected username %s, got %s", expectedUser, row.Username)
		}
	}
}

func TestBTreeChildLeafSplitNonSequential(t *testing.T) {
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

	// Insert keys in non-sequential order to trigger splits in both left and middle leaves (Case B)
	keys := []uint32{
		50, 25, 75, 10, 30, 60, 90, 5, 15, 27, 35, 55, 65, 80, 95,
		1, 3, 7, 12, 18, 22, 28, 32, 38, 52, 58, 62, 68, 78, 85,
		88, 92, 98, 2, 4, 6, 8, 14, 16, 24, 26, 34, 36, 54, 56,
		64, 66, 76, 82, 86,
	}

	for _, k := range keys {
		val := makeRow(k, fmt.Sprintf("user%d", k), fmt.Sprintf("user%d@db.com", k))
		if err := tree.Insert(k, val); err != nil {
			t.Fatalf("insert %d failed: %v", k, err)
		}
	}

	// Verify all keys can be retrieved accurately
	for _, k := range keys {
		valBytes, err := tree.Find(k)
		if err != nil {
			t.Fatalf("failed to find key %d in non-sequential B-Tree: %v", k, err)
		}

		row, err := encoding.DeserializeRow(valBytes)
		if err != nil {
			t.Fatalf("deserialize failed for key %d: %v", k, err)
		}

		if row.ID != k {
			t.Errorf("expected ID %d, got %d", k, row.ID)
		}
		expectedUser := fmt.Sprintf("user%d", k)
		if row.Username != expectedUser {
			t.Errorf("expected username %s, got %s", expectedUser, row.Username)
		}
	}
}
