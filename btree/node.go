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
const KeySize = 4

const RightPageIDSize = 4
const InternalNodeHeaderSize = LeafNodeHeaderSize + RightPageIDSize
const InternalCellSize = ParentPageIDSize + KeySize
const InternalMaxCells = (storage.PageSize - InternalNodeHeaderSize) / InternalCellSize

const LeafNodeHeaderSize = 10
const LeafCellSize  = KeySize + encoding.RowSize
const LeafMaxCells = (storage.PageSize - LeafNodeHeaderSize) / LeafCellSize


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
	startByte := LeafNodeHeaderSize + (cellIndex * LeafCellSize)
	key := binary.LittleEndian.Uint32(n.Page[startByte:startByte + KeySize])
	return key
}

func (n *Node) SetLeafKey(key uint32, cellIndex uint32) {
	startByte := LeafNodeHeaderSize + (cellIndex * LeafCellSize)
	binary.LittleEndian.PutUint32(n.Page[startByte:startByte+KeySize], key)
}

func (n *Node) LeafValue(cellIndex uint32) []byte {
	startByte := LeafNodeHeaderSize + KeySize + (cellIndex * LeafCellSize)
	return n.Page[startByte:startByte + encoding.RowSize]
}

func (n *Node) SetLeafValue(value []byte, cellIndex uint32){
	startByte := LeafNodeHeaderSize + KeySize + (cellIndex * LeafCellSize)
	copy(n.Page[startByte:startByte + encoding.RowSize], value)
}

func (n *Node) SetLeafCell(key uint32, value []byte, cellIndex uint32) {
	n.SetLeafKey(key, cellIndex)
	n.SetLeafValue(value, cellIndex)
}

func (n *Node) InternalRightChild() uint32 {
	return binary.LittleEndian.Uint32(n.Page[LeafNodeHeaderSize:LeafNodeHeaderSize+RightPageIDSize])
}

func (n *Node) SetInternalRightChild(id uint32) {
	binary.LittleEndian.PutUint32(n.Page[LeafNodeHeaderSize:LeafNodeHeaderSize+RightPageIDSize],id)
}

func (n *Node) InternalKey(cellIndex uint32) uint32 {
	startByte := InternalNodeHeaderSize + (cellIndex * InternalCellSize)
	key := binary.LittleEndian.Uint32(n.Page[startByte:startByte + KeySize])
	return key
}

func (n *Node) SetInternalKey(key uint32, cellIndex uint32) {
	startByte := InternalNodeHeaderSize + (cellIndex * InternalCellSize)
	binary.LittleEndian.PutUint32(n.Page[startByte:startByte+KeySize], key)
}

func (n *Node) InternalValue(cellIndex uint32) uint32 {
	startByte := InternalNodeHeaderSize + KeySize + (cellIndex * InternalCellSize)
	return binary.LittleEndian.Uint32(n.Page[startByte:startByte + ParentPageIDSize])
}

func (n *Node) SetInternalValue(value uint32, cellIndex uint32){
	startByte := InternalNodeHeaderSize + KeySize + (cellIndex * InternalCellSize)
	binary.LittleEndian.PutUint32(n.Page[startByte:startByte + ParentPageIDSize], value)
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
	byteStart:= LeafNodeHeaderSize + (index * LeafCellSize)
	byteEnd := LeafNodeHeaderSize + (n.NumKeys() * LeafCellSize)
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
	{page, err := t.pager.GetPage(t.rootPageID)
	if err != nil {
		return nil, err
	}
	root := Node{Page: page}
	searchNode := root

	if !root.IsLeaf() {
		pageIndex := root.FindChildPage(nodeKey)
		p, err := t.pager.GetPage(pageIndex)
				if err != nil {
					return nil, err
				}
	searchNode = Node{Page: p}

	}
	ind, found := searchNode.FindKeyInLeaf(nodeKey)
	if found {
		return searchNode.LeafValue(ind), nil

	} 
	return nil, fmt.Errorf("Key not found: %d", nodeKey)}
}

func (t *BTree) Insert(nodeKey uint32, nodeVal []byte) error {
	page, err := t.pager.GetPage(t.rootPageID)
	if err != nil {
		return err
	}
	root := Node{Page: page}

	insertNode := root
	insertPageIndex := t.rootPageID

	if root.IsLeaf() && root.NumKeys() == LeafMaxCells {
		// if we are at capacity, we split, and refetch the state of the new pages
		if err = t.splitRootLeaf(); err != nil {return err}
		page, err = t.pager.GetPage(t.rootPageID)
		if err != nil {
			return err
		}
		root = Node{Page: page}
	}

	if !root.IsLeaf() {
		insertPageIndex = root.FindChildPage(nodeKey)
		p, err := t.pager.GetPage(insertPageIndex)
				if err != nil {
					return err
				}
				insertNode = Node{Page: p}
	}
	if insertNode.NumKeys() == LeafMaxCells{
		if err = t.splitChildLeaf(insertPageIndex, &insertNode); err != nil {return err}
		// Re-fetch most current root
		page, err = t.pager.GetPage(t.rootPageID)
		if err != nil {
			return err
		}
		root = Node{Page: page}
		
		insertPageIndex = root.FindChildPage(nodeKey)
		p, err := t.pager.GetPage(insertPageIndex)
		if err != nil {
			return err
		}
		insertNode = Node{Page: p}
	}
	err = insertNode.InsertLeafCell(nodeKey,nodeVal)
	if err != nil {
		return err
	}
	return t.pager.FlushPage(insertNode.Page, insertPageIndex)
}

func (n *Node) FindChildPage(key uint32) uint32 {
	pageIndex := n.InternalRightChild()
	for i := uint32(0); i < n.NumKeys(); i++ {
		if key <= n.InternalKey(i) {
			return n.InternalValue(i)
		}
	}
	return pageIndex
}

func (t *BTree) splitRootLeaf() error {
	p := t.pager

	page, err := p.GetPage(t.rootPageID)

	leftPageIndex := uint32(p.PageCount)
	rightPageIndex := uint32(p.PageCount + 1)

	leftPage, err := p.GetPage(leftPageIndex)
	rightPage, err := p.GetPage(rightPageIndex)
	if err != nil {
		return err
	}

	root := Node{Page: page}
	left := Node{Page: leftPage}
	right := Node{Page: rightPage}
	leftCells := (LeafMaxCells + 1) / 2
	rightCells := LeafMaxCells - leftCells
	startFirstHalf := LeafNodeHeaderSize
	endFirstHalf := LeafNodeHeaderSize + (leftCells * LeafCellSize)
	
	startSecondHalf := endFirstHalf
	endSecondHalf := endFirstHalf + (rightCells * LeafCellSize)

	copy(left.Page[LeafNodeHeaderSize:endFirstHalf], root.Page[startFirstHalf:endFirstHalf])
	copy(right.Page[LeafNodeHeaderSize:], root.Page[startSecondHalf:endSecondHalf])

	left.SetNumKeys(uint32(leftCells))
	left.SetIsLeaf(true)
	left.SetIsRoot(false)
	left.SetParentPageID(t.rootPageID)

	right.SetNumKeys(uint32(rightCells))
	right.SetIsLeaf(true)
	right.SetIsRoot(false)
	right.SetParentPageID(t.rootPageID)

	rightChildID := rightPageIndex
	childKey := root.LeafKey(uint32(leftCells-1))
	childPageID := leftPageIndex

	clear(root.Page[LeafNodeHeaderSize:])
	root.SetInternalRightChild(rightChildID)
	root.SetInternalKey(childKey,0)
	root.SetInternalValue(childPageID,0)
	root.SetIsLeaf(false)
	root.SetNumKeys(1)


	if err := p.FlushPage(root.Page,0); err != nil {return err}
	if err := p.FlushPage(left.Page,leftPageIndex); err != nil {return err}
	if err := p.FlushPage(right.Page,rightPageIndex); err != nil {return err}

	return nil

}

func (n *Node) InsertInternalCell(key uint32, childPageId uint32) error {
	if n.NumKeys() == InternalMaxCells {
		return fmt.Errorf("Page at capacity: Max Internal cells %d", InternalMaxCells)
	}
	low := 0
	high := int(n.NumKeys())-1
	insertIndex := int(n.NumKeys())

	for low <= high {
		mid := (high + low) / 2
		k :=  n.InternalKey(uint32(mid))
		if key == k{
			return fmt.Errorf("Key already exists: %d", key)
		} else if key < k {
			insertIndex = mid
			high = mid - 1
		} else {
			low  = mid + 1
		}
	}
	startByte := InternalNodeHeaderSize + (insertIndex * InternalCellSize)
	endByte := InternalNodeHeaderSize + (n.NumKeys() * InternalCellSize)

	copy(n.Page[startByte + InternalCellSize:], n.Page[startByte:endByte])
	n.SetInternalKey(key, uint32(insertIndex))
	n.SetInternalValue(childPageId, uint32(insertIndex))
	n.SetNumKeys(n.NumKeys() + 1)

	return nil
}

func (t *BTree) splitChildLeaf(pageIndex uint32, insertNode *Node) error {
	p := t.pager
	rightPageIndex := p.PageCount

	right, err := p.GetPage(uint32(p.PageCount))
	if err != nil {
    	return err
	}	
	r, err := p.GetPage(t.rootPageID)
	if err != nil {
		return err
	}

	root := Node{Page: r}

	leftCells := (LeafMaxCells + 1) / 2
	rightCells := LeafMaxCells - leftCells

	secondHalfbytes := insertNode.Page[LeafNodeHeaderSize+(leftCells * LeafCellSize):]
	
	copy(right[LeafNodeHeaderSize:LeafNodeHeaderSize+(rightCells * LeafCellSize)], secondHalfbytes)
	clear(insertNode.Page[LeafNodeHeaderSize+(leftCells * LeafCellSize):])
	
	rightNode := Node{Page: right}
	
	rightNode.SetIsLeaf(true)
	rightNode.SetIsRoot(false)
	rightNode.SetNumKeys(uint32(rightCells))
	rightNode.SetParentPageID(t.rootPageID)

	insertNode.SetNumKeys(uint32(leftCells))

	if (pageIndex == root.InternalRightChild()) {
		root.SetInternalRightChild(uint32(rightPageIndex))
	} else {
		for i := uint32(0); i < root.NumKeys(); i++ {
			if root.InternalValue(i) == pageIndex {
				root.SetInternalValue(uint32(rightPageIndex), i)
				break
			}
		}
	}
	dividingKey := insertNode.LeafKey(uint32(leftCells)-1)
	root.InsertInternalCell(dividingKey,pageIndex)

	if err = p.FlushPage(insertNode.Page, pageIndex); err != nil {return err}
	if err = p.FlushPage(root.Page, t.rootPageID); err != nil {return err}
	return p.FlushPage(rightNode.Page,uint32(rightPageIndex))

}