package encoding

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

// Temporary struct to hold the data for a row.
// This is used to encode and decode rows to and from bytes.
// Current Row size = 255 + 32 + 4 = 291 Bytes
const IDSize = 4
const UsernameSize = 32
const EmailSize = 255
const RowSize = IDSize + UsernameSize + EmailSize
type Row struct {
	ID uint32
	Username string
	Email string
}

// Converts a Row into a byte array, serializes ID by Little Endian
func (r *Row) SerializeRow() ([]byte, error) {
	// validate fields are within space contraints
	if len(r.Username) > UsernameSize {
		return nil, fmt.Errorf("%s greater than max field size (%d bytes) : size = %d bytes", r.Username, UsernameSize,len(r.Username))
	}
	if len(r.Email) > EmailSize {
		return nil, fmt.Errorf("%s greater than max field size (%d bytes) : size = %d bytes", r.Email, EmailSize,len(r.Email))
	}

	buffer := make([]byte, RowSize)

	binary.LittleEndian.PutUint32(buffer, r.ID)
	copy(buffer[IDSize:IDSize+UsernameSize], r.Username)
	copy(buffer[RowSize-EmailSize:RowSize], r.Email)

	return buffer, nil

}

func DeserializeRow(b []byte) (*Row, error){
	if (len(b) < RowSize) {
		return nil, fmt.Errorf("Row greater than max row size (%d bytes) : size = %d bytes", RowSize,len(b))
	}

	id := binary.LittleEndian.Uint32(b[0:IDSize])
	user := bytes.TrimRight( b[IDSize: IDSize + UsernameSize], "\x00")
	email := bytes.TrimRight(b[RowSize - EmailSize:RowSize], "\x00")
	
	return &Row{
		ID: id,
		Username: string(user),
		Email: string(email),
	}, nil

}

