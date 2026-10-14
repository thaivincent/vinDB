# vinDB

A disk-backed relational database engine built from scratch in Go.

`vinDB` explores database internals from the ground up: block-level disk paging, binary record serialization, self-balancing B-Tree index structures, and an interactive command-line REPL.

---

## Architecture Overview

```
                      +-----------------------------+
                      |       CLI REPL (main)       |
                      |   insert / select / .exit   |
                      +--------------+--------------+
                                     |
                                     v
                      +-----------------------------+
                      |       B-Tree Indexing       |
                      |   (Leaf & Internal Nodes)   |
                      +--------------+--------------+
                                     |
                      +--------------v--------------+
                      |   Record Encoding (row.go)  |
                      |  Binary Serialization (LE)  |
                      +--------------+--------------+
                                     |
                      +--------------v--------------+
                      |     Heap File Manager       |
                      |    (Slot Math, Scans)       |
                      +--------------+--------------+
                                     |
                      +--------------v--------------+
                      |    Virtual Pager (pager.go) |
                      |    4KB Block-Level Disk I/O |
                      +--------------+--------------+
                                     |
                                     v
                           [ Physical .db File ]
```

---

## Features

- **4KB Page-Aligned Block Storage**: Data is organized into 4096-byte pages, matching standard OS virtual memory pages to minimize I/O overhead.
- **Positional Random-Access I/O**: Reads and writes operate through `os.File.ReadAt` and `WriteAt` without shared file-pointer mutations.
- **Binary Record Serialization**: Structured row serialization using little-endian byte ordering, strict bounds checking, and null-padded string compaction.
- **Slotted Heap File Organization**: Dynamic page allocation and slot offset translation for sequential row scans.
- **B-Tree Index Nodes (In Progress)**: Slotted page layouts for both internal and leaf nodes with sorted key arrays and binary search traversal.
- **Interactive CLI REPL**: Terminal shell for executing commands against persistent database files.

---

## Disk & Page Format

Every database file managed by `vinDB` is partitioned into contiguous 4096-byte pages:

```
+---------------------------------------------------------------+
|                       Page 0 (Header Page)                    |
|  [0..3] Row Count (uint32 LE) | [4..4095] Row Slots ...       |
+---------------------------------------------------------------+
|                       Page 1 (Data Page)                      |
|  [0..4095] Row Slots ...                                      |
+---------------------------------------------------------------+
|                       Page N ...                              |
+---------------------------------------------------------------+
```

### Row Schema (Fixed-Width, 100 Bytes)

| Column | Type | Size | Encoding |
|---|---|---|---|
| `ID` | `uint32` | 4 bytes | Little-Endian Integer |
| `Username` | `string` | 32 bytes | UTF-8, Null-Padded |
| `Email` | `string` | 64 bytes | UTF-8, Null-Padded |
| **Total** | | **100 bytes** | |

### B-Tree Leaf Node Layout (4096 Bytes)

```
+--------------------+---------------------+------------------------+
| Node Header (13B)  | Cells (Keys + Rows) | Unallocated Free Space |
+--------------------+---------------------+------------------------+
| isLeaf (1B)        | Cell 0 [Key, Row]   | ...                    |
| numCells (4B)      | Cell 1 [Key, Row]   |                        |
| parentPage (4B)    | ...                 |                        |
| nextPage (4B)      | Cell N [Key, Row]   |                        |
+--------------------+---------------------+------------------------+
```

---

## Project Structure

```
vinDB/
├── main.go               # CLI REPL entry point
├── btree/
│   ├── node.go           # B-Tree leaf/internal node memory layouts & cell math
│   └── btree_test.go     # Node layout, insertion, and split tests
├── encoding/
│   ├── row.go            # Row struct, binary serialization & deserialization
│   └── row_test.go       # Serialization round-trip & bounds tests
├── storage/
│   ├── page.go           # 4KB Page type definition
│   ├── pager.go          # File I/O manager (Open, GetPage, FlushPage)
│   ├── heap.go           # Heap file record scanner & slot allocator
│   └── storage_test.go   # Pager persistence & reopen test suites
├── go.mod
└── README.md
```

---

## Getting Started

### Prerequisites

- [Go](https://go.dev/dl/) 1.22 or later

### Installation & Build

```bash
git clone https://github.com/thaivincent/vinDB.git
cd vinDB
go build -o vinDB.exe
```

### Running the REPL

Launch the database shell by passing a database file:

```bash
./vinDB.exe mydb.db
```

Inside the REPL:

```text
vinDB > insert 1 alice alice@example.com
vinDB > insert 2 bob bob@example.com
vinDB > select
(1, alice, alice@example.com)
(2, bob, bob@example.com)
vinDB > .exit
```

Restarting the process with the same `.db` file preserves all inserted records.

---

## Running Tests

Run the full automated test suite across all packages:

```bash
go test -v ./...
```

Run tests for an individual package:

```bash
go test -v ./btree/...
go test -v ./storage/...
go test -v ./encoding/...
```

---

## Roadmap / Milestones

- [x] **Phase 1: Storage Engine & Heap File**
  - [x] 4KB block-level virtual pager (`ReadAt` / `WriteAt`)
  - [x] Fixed-width binary row serialization
  - [x] Heap file record scanner and slot location indexing
  - [x] Basic interactive REPL (`insert`, `select`, `.exit`)
- [ ] **Phase 2: B-Tree Indexing**
  - [x] Leaf and internal node byte packing inside 4KB pages
  - [x] Binary search for in-node key lookups
  - [ ] Node splitting on overflow and root node rebalancing
  - [ ] B-Tree cursor for ordered range scans
- [ ] **Phase 3: SQL Parser**
  - [ ] Hand-crafted tokenizer / lexer
  - [ ] Recursive-descent parser producing Abstract Syntax Trees (AST)
  - [ ] Support for `CREATE TABLE`, `INSERT`, `SELECT`, and `WHERE` clauses
- [ ] **Phase 4: Volcano Query Executor**
  - [ ] Iterator model execution (`Open` / `Next` / `Close`)
  - [ ] Sequential scan vs. B-Tree index scan selection
  - [ ] Predicate evaluation and filter operators
- [ ] **Phase 5: Write-Ahead Logging (WAL) & Recovery**
  - [ ] Append-only WAL with Log Sequence Numbers (LSNs)
  - [ ] Checkpoint records and startup replay recovery
- [ ] **Phase 6: ACID Transactions**
  - [ ] Two-Phase Locking (2PL)
  - [ ] `BEGIN`, `COMMIT`, and `ROLLBACK` commands

---

## License

MIT

