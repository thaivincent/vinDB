package storage

import (
	"encoding/binary"
	"github.com/vince/vinDB/encoding"
)

// The struct that manages the rows inside of the file
// the only persistant data it needs is the number of rows the file has.
// Metadata on number of rows is stored in the first 4 bytes of the first page in the file
type HeapFile struct {
	Pager *Pager
	NumRows uint32
}

const HeaderSize = 4

func (p *Pager) OpenHeap() (*HeapFile, error) {
	fp, err := p.GetPage(0)

	if err != nil {
		return nil, err
	}

	rowCount := binary.LittleEndian.Uint32(fp[0:HeaderSize])

	return &HeapFile{
		Pager: p,
		NumRows: rowCount,
	}, nil
}


// A helper that returns the the page and offset of the specified row number
func (h *HeapFile) slotLocation(rowNum uint32) (pageId uint32, offset int) {
	bytePos := HeaderSize + rowNum * encoding.RowSize
	pageNum := bytePos / PageSize
	offset = int(bytePos % PageSize)
	return pageNum, offset
}

func (h *HeapFile) Insert(row *encoding.Row) error {
	pageNum, offset := h.slotLocation(h.NumRows)


	bytes, err := row.SerializeRow()
	page, err := h.Pager.GetPage(pageNum)

	if err != nil {
		return err
	}

	copy(page[offset:offset + encoding.RowSize], bytes)
	err = h.Pager.FlushPage(page, pageNum)
	if err != nil {
		return err
	}

	h.NumRows++
	zeroPage, err := h.Pager.GetPage(0)
	if err != nil {
		return err
	}
	binary.LittleEndian.PutUint32(zeroPage[0:HeaderSize], h.NumRows)

	return h.Pager.FlushPage(zeroPage, 0)

}

func (h *HeapFile) Scan(fn func(*encoding.Row) bool) error {
	for i:= 0; i < int(h.NumRows); i++ {
		pageNum, buffer := h.slotLocation(uint32(i))

		page, err := h.Pager.GetPage(pageNum)
		if err != nil {
			return err
		}

		r, err := encoding.DeserializeRow(page[buffer:buffer + encoding.RowSize])
		if err != nil {
			return err
		}

		if !fn(r) {
			break
		}
	}
	return nil
}


