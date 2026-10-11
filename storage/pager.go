package storage
import (
	"os"
)



type Pager struct {
	FileRef *os.File
	PageCount int64
}

func Open(filename string) (*Pager, error) {
	// bitwise OR onto the flags for read/write and create so we can get all 3 functionality
	// 0600 is the read/write permission
	file, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		return nil, err
	}

	filestat, err := file.Stat()
	if err != nil {
		return nil, err
	}

	filesize := filestat.Size()
	p := Pager{
		FileRef: file,
		PageCount: filesize / PageSize,
	}

	return &p, err
}

func (p *Pager) GetPage(pageIndex uint32)(*Page, error) {
	offset := pageIndex * PageSize
	pg := Page{}

	if pageIndex >= uint32(p.PageCount) {
		p.PageCount = int64(pageIndex) + 1
	} else {
		_, err := p.FileRef.ReadAt(pg[:],int64(offset))
		if err != nil {
			return nil, err
		}
	}

	return &pg, nil
	
}
