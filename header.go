package rmscene

import (
	"bytes"
	"errors"
	"fmt"
	"io"
)

// HeaderV6 is the expected 43-byte magic header for reMarkable version 6 .rm files.
const HeaderV6 = "reMarkable .lines file, version=6          "

// HeaderLength is the required length of the header in bytes (43 bytes).
const HeaderLength = len(HeaderV6)

var (
	// ErrInvalidHeader is returned when the file header does not match HeaderV6.
	ErrInvalidHeader = errors.New("invalid reMarkable v6 header")
	// ErrUnexpectedEOF is returned when the stream ends unexpectedly.
	ErrUnexpectedEOF = errors.New("unexpected EOF")
)

// ValidateHeader reads HeaderLength bytes from r and returns nil if it matches HeaderV6.
func ValidateHeader(r io.Reader) error {
	buf := make([]byte, HeaderLength)
	if _, err := io.ReadFull(r, buf); err != nil {
		if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
			return fmt.Errorf("%w: expected %d bytes, got fewer", ErrInvalidHeader, HeaderLength)
		}
		return err
	}
	if string(buf) != HeaderV6 {
		return fmt.Errorf("%w: got %q, expected %q", ErrInvalidHeader, string(buf), HeaderV6)
	}
	return nil
}

// ValidateHeaderBytes verifies that the provided byte slice starts with HeaderV6.
func ValidateHeaderBytes(data []byte) error {
	if len(data) < HeaderLength {
		return fmt.Errorf("%w: data length %d is less than header length %d", ErrInvalidHeader, len(data), HeaderLength)
	}
	if !bytes.Equal(data[:HeaderLength], []byte(HeaderV6)) {
		return fmt.Errorf("%w: got %q, expected %q", ErrInvalidHeader, string(data[:HeaderLength]), HeaderV6)
	}
	return nil
}
