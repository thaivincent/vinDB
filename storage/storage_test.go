package storage_test

import (
	"os"
	"testing"

	"github.com/vince/vinDB/encoding"
	"github.com/vince/vinDB/storage"
)

// tempDB creates a temporary test database file and schedules its deletion when the test finishes.
func tempDB(t *testing.T) string {
	t.Helper()
	f, err := os.CreateTemp("", "vindb-test-*.db")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	name := f.Name()
	f.Close()
	t.Cleanup(func() { os.Remove(name) })
	return name
}

func TestPagerReadWriteRoundTrip(t *testing.T) {
	path := tempDB(t)

	pg, err := storage.Open(path)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}

	page, err := pg.GetPage(0)
	if err != nil {
		t.Fatalf("GetPage(0) failed: %v", err)
	}

	// Write signature bytes into page 0
	page[0] = 0xAA
	page[1] = 0xBB
	page[storage.PageSize-1] = 0xCC

	if err := pg.FlushPage(page, 0); err != nil {
		t.Fatalf("FlushPage failed: %v", err)
	}
	if err := pg.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// Reopen file in a new Pager instance
	pg2, err := storage.Open(path)
	if err != nil {
		t.Fatalf("reopen failed: %v", err)
	}
	defer pg2.Close()

	page2, err := pg2.GetPage(0)
	if err != nil {
		t.Fatalf("GetPage after reopen failed: %v", err)
	}

	if page2[0] != 0xAA || page2[1] != 0xBB || page2[storage.PageSize-1] != 0xCC {
		t.Errorf("bytes did not survive reopen: [0]=%x, [1]=%x, [last]=%x",
			page2[0], page2[1], page2[storage.PageSize-1])
	}
}

func TestHeapInsertAndScanAcrossReopen(t *testing.T) {
	path := tempDB(t)

	pg, err := storage.Open(path)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}

	heap, err := pg.OpenHeap()
	if err != nil {
		t.Fatalf("OpenHeap failed: %v", err)
	}

	testRows := []*encoding.Row{
		{ID: 1, Username: "alice", Email: "alice@example.com"},
		{ID: 2, Username: "bob", Email: "bob@example.com"},
		{ID: 3, Username: "carol", Email: "carol@example.com"},
	}

	for _, r := range testRows {
		if err := heap.Insert(r); err != nil {
			t.Fatalf("Insert failed for row %d: %v", r.ID, err)
		}
	}

	if heap.NumRows != 3 {
		t.Fatalf("expected NumRows = 3, got %d", heap.NumRows)
	}

	if err := pg.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// Reopen in a new process simulation
	pg2, err := storage.Open(path)
	if err != nil {
		t.Fatalf("reopen Open failed: %v", err)
	}
	defer pg2.Close()

	heap2, err := pg2.OpenHeap()
	if err != nil {
		t.Fatalf("reopen OpenHeap failed: %v", err)
	}

	if heap2.NumRows != 3 {
		t.Fatalf("expected NumRows = 3 after reopen, got %d", heap2.NumRows)
	}

	var scanned []*encoding.Row
	err = heap2.Scan(func(r *encoding.Row) bool {
		scanned = append(scanned, r)
		return true
	})
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	if len(scanned) != len(testRows) {
		t.Fatalf("scanned count mismatch: got %d, want %d", len(scanned), len(testRows))
	}

	for i, want := range testRows {
		got := scanned[i]
		if got.ID != want.ID || got.Username != want.Username || got.Email != want.Email {
			t.Errorf("row %d mismatch: got %+v, want %+v", i, got, want)
		}
	}
}

func TestHeapEarlyScanStop(t *testing.T) {
	path := tempDB(t)

	pg, err := storage.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer pg.Close()

	heap, err := pg.OpenHeap()
	if err != nil {
		t.Fatal(err)
	}

	for i := uint32(1); i <= 5; i++ {
		heap.Insert(&encoding.Row{ID: i, Username: "user", Email: "user@example.com"})
	}

	count := 0
	heap.Scan(func(r *encoding.Row) bool {
		count++
		return count < 2 // stop scanning after 2 rows
	})

	if count != 2 {
		t.Errorf("expected scan to stop at 2, got %d", count)
	}
}

