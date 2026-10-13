package btree

import (
	"encoding/binary"
	"fmt"

	"github.com/vince/vinDB/encoding"
	"github.com/vince/vinDB/storage"
)

// B+ tree that has pages (4096 bytes) for its nodes.
// Each node will have a header with the following format:
// 	- IsLeaf = bool
// 	- IsRoot = bool
//  - ParentPageID = uint32
//  - NumKeys = uint32
// Total overhead = 10 bytes

// **For our FIXED row implementation:
// 	LeafCellSize = 4 + 291 = 295
// 	LeafMaxCells = (4096 -10) / 295 = 13 cells
//  Each cell is a (key,value) pair that stores (ID,Row)


const ParentPageIDSize = 4
const NumKeySize = 4
const NodeHeaderSize = 10
const KeySize = 4
const LeafCellSize  = KeySize + encoding.RowSize
const LeafMaxCells = (storage.PageSize - NodeHeaderSize) / LeafCellSize


type Node struct {
	Page *storage.Page
}

func (n *Node) IsLeaf() bool {
	return n.Page[0] == 1
}

func (n *Node) SetIsLeaf(isLeaf bool) {
	if isLeaf {
		n.Page[0] = 1
	} else {
		n.Page[0] = 0 
	}
}

func (n *Node) IsRoot() bool {
	return n.Page[1] == 1
}

func (n *Node) SetIsRoot(isRoot bool) {
	if isRoot {
		n.Page[1] = 1
	} else {
		n.Page[1] = 0 
	}
}

func (n *Node) ParentPageID() uint32 {
	return binary.LittleEndian.Uint32(n.Page[2:2+ParentPageIDSize])
}

func (n *Node) SetParentPageID(id uint32) {
	binary.LittleEndian.PutUint32(n.Page[2:2+ParentPageIDSize], id)
}

func (n *Node) NumKeys() uint32 {
	return binary.LittleEndian.Uint32(n.Page[2+ParentPageIDSize:2+ParentPageIDSize+NumKeySize])
}

func (n *Node) SetNumKeys(numKeys uint32) {
	binary.LittleEndian.PutUint32(n.Page[2+ParentPageIDSize:2+ParentPageIDSize+NumKeySize], numKeys)
}

func (n *Node) LeafKey(cellIndex uint32) uint32 {
	startByte := NodeHeaderSize + (cellIndex * LeafCellSize)
	key := binary.LittleEndian.Uint32(n.Page[startByte:startByte + KeySize])
	return key
}

func (n *Node) SetLeafKey(key uint32, cellIndex uint32) {
	startByte := NodeHeaderSize + (cellIndex * LeafCellSize)
	binary.LittleEndian.PutUint32(n.Page[startByte:startByte+KeySize], key)
}

func (n *Node) LeafValue(cellIndex uint32) []byte {
	startByte := NodeHeaderSize + KeySize + (cellIndex * LeafCellSize)
	return n.Page[startByte:startByte + encoding.RowSize]
}

func (n *Node) SetLeafValue(value []byte, cellIndex uint32){
	startByte := NodeHeaderSize + KeySize + (cellIndex * LeafCellSize)
	copy(n.Page[startByte:startByte + encoding.RowSize], value)
}

func (n *Node) SetLeafCell(key uint32, value []byte, cellIndex uint32) {
	n.SetLeafKey(key, cellIndex)
	n.SetLeafValue(value, cellIndex)
}

// FindKeyInLeaf searches for key in this leaf.
// It returns (cellIndex, found).
// If found is false, cellIndex is the position where the key SHOULD be inserted to keep keys sorted!
func (n *Node) FindKeyInLeaf(key uint32) (uint32, bool){
	if n.NumKeys() == 0 {
		return 0, false
	}
	low := 0
	high := int(n.NumKeys() - 1)
	
	for low <= high {
		mid := (high + low) / 2
		mid_val :=  n.LeafKey(uint32(mid))
		found := mid_val == key
		if found {
			return uint32(mid), found
		}
		if mid_val < key {
			low = mid+1
		} else {
			high = mid-1
		}
	}

	return uint32(low), false 
}

func (n* Node) InsertLeafCell (key uint32, val []byte) error {
	index, found := n.FindKeyInLeaf(key)
	if found {
		return fmt.Errorf("Key already exists: %v", key)
	}
	if n.NumKeys() == LeafMaxCells {
		return fmt.Errorf("Node at capacty: %v cells", LeafMaxCells)
	}
	// We need to shift cells from index -> n.NumKeys()-1 up by 1 to make space for our new cell
	byteStart:= NodeHeaderSize + (index * LeafCellSize)
	byteEnd := NodeHeaderSize + (n.NumKeys() * LeafCellSize)
	copy(n.Page[byteStart+LeafCellSize:byteEnd+LeafCellSize], n.Page[byteStart:byteEnd])
	n.SetLeafCell(key, val,index)
	n.SetNumKeys(n.NumKeys()+1)
	return nil
}

// A Btree is defined by a pager and its root page index.
// Btree methods return Nodes which allow manipulation of Pages
type BTree struct {
	pager *storage.Pager
	rootPageID uint32
}

func NewBTree(pager *storage.Pager, rootPageID uint32) (*BTree, error) {
	page, err := pager.GetPage(rootPageID)
	if err != nil {
		return nil, err
	}
	node := Node{Page: page}

	// Check if our page has a header
	if !node.IsRoot() && !node.IsLeaf() && node.NumKeys() == 0 {
		node.SetIsLeaf(true)
		node.SetIsRoot(true)
		node.SetNumKeys(0)
		err := pager.FlushPage(node.Page, rootPageID)
		if err != nil {
			return nil, err
		}
	}
	
	return &BTree{
		pager: pager ,
		rootPageID: rootPageID,
	}, nil
}

func (t *BTree) Find(nodeKey uint32) ([]byte, error){
	page, err := t.pager.GetPage(t.rootPageID)
	if err != nil {
		return nil, err
	}
	node := Node{Page: page}

	ind, found := node.FindKeyInLeaf(nodeKey)
	if found {
		return node.LeafValue(ind), nil

	} 
	return nil, fmt.Errorf("Key not found: %d", nodeKey)
}

func (t *BTree) Insert(nodeKey uint32, nodeVal []byte) error {
	page, err := t.pager.GetPage(t.rootPageID)
	if err != nil {
		return err
	}
	node := Node{Page: page}
	err = node.InsertLeafCell(nodeKey,nodeVal)
	if err != nil {
		return err
	}
	return t.pager.FlushPage(node.Page, t.rootPageID)
}


