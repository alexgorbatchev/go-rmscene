package rmscene

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
)

var (
	// ErrBlockOverflow is returned when reading exceeds the declared block size.
	ErrBlockOverflow = errors.New("block overflow: read past declared block size")
	// ErrUnexpectedTag is returned when a tag does not match the expected index or type.
	ErrUnexpectedTag = errors.New("unexpected tag in block stream")
)

// DataStream wraps an io.ReadSeeker to provide primitives for reading reMarkable v6 binary data.
type DataStream struct {
	r io.ReadSeeker
}

// NewDataStream creates a new DataStream from an io.ReadSeeker.
func NewDataStream(r io.ReadSeeker) *DataStream {
	return &DataStream{r: r}
}

// Tell returns the current byte position in the stream.
func (s *DataStream) Tell() int64 {
	pos, _ := s.r.Seek(0, io.SeekCurrent)
	return pos
}

// Seek seeks to offset relative to whence.
func (s *DataStream) Seek(offset int64, whence int) (int64, error) {
	return s.r.Seek(offset, whence)
}

// ReadBytes reads exactly n bytes from the stream.
func (s *DataStream) ReadBytes(n int) ([]byte, error) {
	buf := make([]byte, n)
	if _, err := io.ReadFull(s.r, buf); err != nil {
		if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
			return nil, io.ErrUnexpectedEOF
		}
		return nil, err
	}
	return buf, nil
}

// ReadUint8 reads a single 8-bit unsigned integer.
func (s *DataStream) ReadUint8() (uint8, error) {
	b, err := s.ReadBytes(1)
	if err != nil {
		return 0, err
	}
	return b[0], nil
}

// ReadBool reads a single boolean byte.
func (s *DataStream) ReadBool() (bool, error) {
	b, err := s.ReadUint8()
	if err != nil {
		return false, err
	}
	return b != 0, nil
}

// ReadUint16 reads a 16-bit little-endian unsigned integer.
func (s *DataStream) ReadUint16() (uint16, error) {
	b, err := s.ReadBytes(2)
	if err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint16(b), nil
}

// ReadUint32 reads a 32-bit little-endian unsigned integer.
func (s *DataStream) ReadUint32() (uint32, error) {
	b, err := s.ReadBytes(4)
	if err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint32(b), nil
}

// ReadFloat32 reads a 32-bit little-endian IEEE-754 float.
func (s *DataStream) ReadFloat32() (float32, error) {
	u, err := s.ReadUint32()
	if err != nil {
		return 0, err
	}
	return math.Float32frombits(u), nil
}

// ReadFloat64 reads a 64-bit little-endian IEEE-754 double.
func (s *DataStream) ReadFloat64() (float64, error) {
	b, err := s.ReadBytes(8)
	if err != nil {
		return 0, err
	}
	u := binary.LittleEndian.Uint64(b)
	return math.Float64frombits(u), nil
}

// ReadVarUint reads an unsigned LEB128 variable-length integer.
func (s *DataStream) ReadVarUint() (uint64, error) {
	var result uint64
	var shift uint
	for {
		b, err := s.ReadUint8()
		if err != nil {
			return 0, err
		}
		result |= uint64(b&0x7F) << shift
		shift += 7
		if (b & 0x80) == 0 {
			break
		}
		if shift >= 64 {
			return 0, errors.New("varuint overflow: value exceeds 64 bits")
		}
	}
	return result, nil
}

// ReadCrdtId reads a CRDT identifier (uint8 part1 followed by varuint part2).
func (s *DataStream) ReadCrdtId() (CrdtId, error) {
	part1, err := s.ReadUint8()
	if err != nil {
		return CrdtId{}, err
	}
	part2, err := s.ReadVarUint()
	if err != nil {
		return CrdtId{}, err
	}
	return CrdtId{Part1: part1, Part2: part2}, nil
}

// ReadTagValues decodes the next varuint tag into (index, tagType).
func (s *DataStream) ReadTagValues() (int, TagType, error) {
	x, err := s.ReadVarUint()
	if err != nil {
		return 0, 0, err
	}
	index := int(x >> 4)
	tagType := TagType(x & 0x0F)
	return index, tagType, nil
}

// ReadTag reads and validates that the next tag matches expectedIndex and expectedType.
// If it does not match, the stream is rewound to the position before the tag.
func (s *DataStream) ReadTag(expectedIndex int, expectedType TagType) (int, TagType, error) {
	pos := s.Tell()
	index, tagType, err := s.ReadTagValues()
	if err != nil {
		_, _ = s.Seek(pos, io.SeekStart)
		return 0, 0, err
	}
	if index != expectedIndex {
		_, _ = s.Seek(pos, io.SeekStart)
		return index, tagType, fmt.Errorf("%w: expected index %d, got %d at position %d", ErrUnexpectedTag, expectedIndex, index, pos)
	}
	if tagType != expectedType {
		_, _ = s.Seek(pos, io.SeekStart)
		return index, tagType, fmt.Errorf("%w: expected tag type %s (0x%X), got 0x%X at position %d", ErrUnexpectedTag, expectedType, uint8(expectedType), uint8(tagType), pos)
	}
	return index, tagType, nil
}

// CheckTag checks whether the next tag matches expectedIndex and expectedType without advancing the stream.
func (s *DataStream) CheckTag(expectedIndex int, expectedType TagType) bool {
	pos := s.Tell()
	index, tagType, err := s.ReadTagValues()
	_, _ = s.Seek(pos, io.SeekStart)
	if err != nil {
		return false
	}
	return index == expectedIndex && tagType == expectedType
}

// ReadId reads a tagged CRDT ID with TagType ID.
func (s *DataStream) ReadId(index int) (CrdtId, error) {
	if _, _, err := s.ReadTag(index, TagID); err != nil {
		return CrdtId{}, err
	}
	return s.ReadCrdtId()
}

// ReadTaggedBool reads a tagged bool with TagType Byte1.
func (s *DataStream) ReadTaggedBool(index int) (bool, error) {
	if _, _, err := s.ReadTag(index, TagByte1); err != nil {
		return false, err
	}
	return s.ReadBool()
}

// ReadTaggedByte reads a tagged byte with TagType Byte1.
func (s *DataStream) ReadTaggedByte(index int) (uint8, error) {
	if _, _, err := s.ReadTag(index, TagByte1); err != nil {
		return 0, err
	}
	return s.ReadUint8()
}

// ReadTaggedInt reads a tagged 4-byte unsigned integer with TagType Byte4.
func (s *DataStream) ReadTaggedInt(index int) (uint32, error) {
	if _, _, err := s.ReadTag(index, TagByte4); err != nil {
		return 0, err
	}
	return s.ReadUint32()
}

// ReadTaggedFloat reads a tagged 4-byte float with TagType Byte4.
func (s *DataStream) ReadTaggedFloat(index int) (float32, error) {
	if _, _, err := s.ReadTag(index, TagByte4); err != nil {
		return 0, err
	}
	return s.ReadFloat32()
}

// ReadTaggedDouble reads a tagged 8-byte double with TagType Byte8.
func (s *DataStream) ReadTaggedDouble(index int) (float64, error) {
	if _, _, err := s.ReadTag(index, TagByte8); err != nil {
		return 0, err
	}
	return s.ReadFloat64()
}

// ReadString reads a string subblock at the given tag index.
func (s *DataStream) ReadString(index int) (string, error) {
	if _, _, err := s.ReadTag(index, TagLength4); err != nil {
		return "", err
	}
	subLen, err := s.ReadUint32()
	if err != nil {
		return "", err
	}
	startPos := s.Tell()

	strLen, err := s.ReadVarUint()
	if err != nil {
		return "", err
	}
	isAscii, err := s.ReadBool()
	if err != nil {
		return "", err
	}
	_ = isAscii

	b, err := s.ReadBytes(int(strLen))
	if err != nil {
		return "", err
	}

	// Consume any remaining bytes in the string subblock
	consumed := s.Tell() - startPos
	if consumed < int64(subLen) {
		remaining := int64(subLen) - consumed
		if _, err := s.ReadBytes(int(remaining)); err != nil {
			return "", err
		}
	} else if consumed > int64(subLen) {
		return "", fmt.Errorf("%w in string subblock", ErrBlockOverflow)
	}

	return string(b), nil
}

// ReadLwwBool reads a LWW bool subblock at the given index.
func (s *DataStream) ReadLwwBool(index int) (LwwValue[bool], error) {
	if _, _, err := s.ReadTag(index, TagLength4); err != nil {
		return LwwValue[bool]{}, err
	}
	subLen, err := s.ReadUint32()
	if err != nil {
		return LwwValue[bool]{}, err
	}
	startPos := s.Tell()

	ts, err := s.ReadId(1)
	if err != nil {
		return LwwValue[bool]{}, err
	}
	val, err := s.ReadTaggedBool(2)
	if err != nil {
		return LwwValue[bool]{}, err
	}

	if err := s.finishSubblock(startPos, subLen); err != nil {
		return LwwValue[bool]{}, err
	}
	return LwwValue[bool]{Timestamp: ts, Value: val}, nil
}

// ReadLwwByte reads a LWW byte subblock at the given index.
func (s *DataStream) ReadLwwByte(index int) (LwwValue[uint8], error) {
	if _, _, err := s.ReadTag(index, TagLength4); err != nil {
		return LwwValue[uint8]{}, err
	}
	subLen, err := s.ReadUint32()
	if err != nil {
		return LwwValue[uint8]{}, err
	}
	startPos := s.Tell()

	ts, err := s.ReadId(1)
	if err != nil {
		return LwwValue[uint8]{}, err
	}
	val, err := s.ReadTaggedByte(2)
	if err != nil {
		return LwwValue[uint8]{}, err
	}

	if err := s.finishSubblock(startPos, subLen); err != nil {
		return LwwValue[uint8]{}, err
	}
	return LwwValue[uint8]{Timestamp: ts, Value: val}, nil
}

// ReadLwwFloat reads a LWW float subblock at the given index.
func (s *DataStream) ReadLwwFloat(index int) (LwwValue[float32], error) {
	if _, _, err := s.ReadTag(index, TagLength4); err != nil {
		return LwwValue[float32]{}, err
	}
	subLen, err := s.ReadUint32()
	if err != nil {
		return LwwValue[float32]{}, err
	}
	startPos := s.Tell()

	ts, err := s.ReadId(1)
	if err != nil {
		return LwwValue[float32]{}, err
	}
	val, err := s.ReadTaggedFloat(2)
	if err != nil {
		return LwwValue[float32]{}, err
	}

	if err := s.finishSubblock(startPos, subLen); err != nil {
		return LwwValue[float32]{}, err
	}
	return LwwValue[float32]{Timestamp: ts, Value: val}, nil
}

// ReadLwwId reads a LWW CRDT ID subblock at the given index.
func (s *DataStream) ReadLwwId(index int) (LwwValue[CrdtId], error) {
	if _, _, err := s.ReadTag(index, TagLength4); err != nil {
		return LwwValue[CrdtId]{}, err
	}
	subLen, err := s.ReadUint32()
	if err != nil {
		return LwwValue[CrdtId]{}, err
	}
	startPos := s.Tell()

	ts, err := s.ReadId(1)
	if err != nil {
		return LwwValue[CrdtId]{}, err
	}
	val, err := s.ReadId(2)
	if err != nil {
		return LwwValue[CrdtId]{}, err
	}

	if err := s.finishSubblock(startPos, subLen); err != nil {
		return LwwValue[CrdtId]{}, err
	}
	return LwwValue[CrdtId]{Timestamp: ts, Value: val}, nil
}

// ReadLwwString reads a LWW string subblock at the given index.
func (s *DataStream) ReadLwwString(index int) (LwwValue[string], error) {
	if _, _, err := s.ReadTag(index, TagLength4); err != nil {
		return LwwValue[string]{}, err
	}
	subLen, err := s.ReadUint32()
	if err != nil {
		return LwwValue[string]{}, err
	}
	startPos := s.Tell()

	ts, err := s.ReadId(1)
	if err != nil {
		return LwwValue[string]{}, err
	}
	val, err := s.ReadString(2)
	if err != nil {
		return LwwValue[string]{}, err
	}

	if err := s.finishSubblock(startPos, subLen); err != nil {
		return LwwValue[string]{}, err
	}
	return LwwValue[string]{Timestamp: ts, Value: val}, nil
}

func (s *DataStream) finishSubblock(startPos int64, subLen uint32) error {
	consumed := s.Tell() - startPos
	if consumed > int64(subLen) {
		return fmt.Errorf("%w: subblock size %d exceeded by %d bytes", ErrBlockOverflow, subLen, consumed-int64(subLen))
	}
	if consumed < int64(subLen) {
		remaining := int64(subLen) - consumed
		if _, err := s.ReadBytes(int(remaining)); err != nil {
			return err
		}
	}
	return nil
}
