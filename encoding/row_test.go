package encoding_test

import (
	"strings"
	"testing"

	"github.com/vince/vinDB/encoding"
)

func TestSerializeDeserializeRoundTrip(t *testing.T) {
	original := &encoding.Row{
		ID:       42,
		Username: "alice",
		Email:    "alice@example.com",
	}

	data, err := original.SerializeRow()
	if err != nil {
		t.Fatalf("SerializeRow failed: %v", err)
	}

	if len(data) != encoding.RowSize {
		t.Fatalf("expected data length %d, got %d", encoding.RowSize, len(data))
	}

	got, err := encoding.DeserializeRow(data)
	if err != nil {
		t.Fatalf("DeserializeRow failed: %v", err)
	}

	if got.ID != original.ID {
		t.Errorf("ID mismatch: got %d, want %d", got.ID, original.ID)
	}
	if got.Username != original.Username {
		t.Errorf("Username mismatch: got %q, want %q", got.Username, original.Username)
	}
	if got.Email != original.Email {
		t.Errorf("Email mismatch: got %q, want %q", got.Email, original.Email)
	}
}

func TestSerializeUsernameTooLong(t *testing.T) {
	row := &encoding.Row{
		ID:       1,
		Username: strings.Repeat("a", encoding.UsernameSize+1),
		Email:    "test@example.com",
	}

	_, err := row.SerializeRow()
	if err == nil {
		t.Fatal("expected error for oversized username, got nil")
	}
}

func TestSerializeEmailTooLong(t *testing.T) {
	row := &encoding.Row{
		ID:       1,
		Username: "bob",
		Email:    strings.Repeat("e", encoding.EmailSize+1),
	}

	_, err := row.SerializeRow()
	if err == nil {
		t.Fatal("expected error for oversized email, got nil")
	}
}

func TestDeserializeDataTooShort(t *testing.T) {
	_, err := encoding.DeserializeRow([]byte{1, 2, 3})
	if err == nil {
		t.Fatal("expected error when deserializing short slice, got nil")
	}
}

