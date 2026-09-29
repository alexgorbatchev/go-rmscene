package rmscene_test

import (
	"bytes"
	"math"
	"testing"

	"github.com/alexgorbatchev/go-rmscene"
)

func TestVarUintRoundtrip(t *testing.T) {
	testCases := []uint64{
		0,
		1,
		127,
		128,
		129,
		255,
		256,
		16383,
		16384,
		1<<32 - 1,
		1<<62 + 42,
	}

	for _, tc := range testCases {
		var buf bytes.Buffer
		w := rmscene.NewDataWriter(&buf)
		if err := w.WriteVarUint(tc); err != nil {
			t.Fatalf("failed to write varuint %d: %v", tc, err)
		}

		s := rmscene.NewDataStream(bytes.NewReader(buf.Bytes()))
		got, err := s.ReadVarUint()
		if err != nil {
			t.Fatalf("failed to read varuint for %d: %v", tc, err)
		}
		if got != tc {
			t.Fatalf("expected %d, got %d", tc, got)
		}
	}
}

func TestTags(t *testing.T) {
	var buf bytes.Buffer
	w := rmscene.NewDataWriter(&buf)

	// Write tag index 1, TagID (0xF)
	if err := w.WriteTag(1, rmscene.TagID); err != nil {
		t.Fatal(err)
	}
	// Write tag index 8, TagByte4 (0x4) -> this is Paper Pro 0x84 0x01!
	if err := w.WriteTag(8, rmscene.TagByte4); err != nil {
		t.Fatal(err)
	}

	data := buf.Bytes()
	// Tag 8 TagByte4: (8 << 4) | 4 = 132 (0x84). In LEB128: [0x84, 0x01]
	expectedPaperProTag := []byte{0x84, 0x01}
	if !bytes.Equal(data[1:3], expectedPaperProTag) {
		t.Fatalf("expected tag 8 Byte4 to be [0x84, 0x01], got %v", data[1:3])
	}

	s := rmscene.NewDataStream(bytes.NewReader(data))

	// CheckTag should verify tag without advancing
	if !s.CheckTag(1, rmscene.TagID) {
		t.Fatal("expected CheckTag(1, TagID) to return true")
	}
	if s.CheckTag(1, rmscene.TagByte1) {
		t.Fatal("expected CheckTag(1, TagByte1) to return false")
	}

	// ReadTag should succeed and advance
	idx, tagType, err := s.ReadTag(1, rmscene.TagID)
	if err != nil || idx != 1 || tagType != rmscene.TagID {
		t.Fatalf("expected tag (1, TagID), got (%d, %v, %v)", idx, tagType, err)
	}

	// ReadTag 8 Byte4
	idx, tagType, err = s.ReadTag(8, rmscene.TagByte4)
	if err != nil || idx != 8 || tagType != rmscene.TagByte4 {
		t.Fatalf("expected tag (8, TagByte4), got (%d, %v, %v)", idx, tagType, err)
	}
}

func TestTagMismatchRewind(t *testing.T) {
	var buf bytes.Buffer
	w := rmscene.NewDataWriter(&buf)
	if err := w.WriteTag(3, rmscene.TagByte4); err != nil {
		t.Fatal(err)
	}

	s := rmscene.NewDataStream(bytes.NewReader(buf.Bytes()))
	initialPos := s.Tell()

	// Try to read mismatched index
	_, _, err := s.ReadTag(2, rmscene.TagByte4)
	if err == nil {
		t.Fatal("expected error on mismatched index")
	}
	if s.Tell() != initialPos {
		t.Fatalf("expected stream to rewind to %d, got %d", initialPos, s.Tell())
	}

	// Try to read mismatched tag type
	_, _, err = s.ReadTag(3, rmscene.TagID)
	if err == nil {
		t.Fatal("expected error on mismatched tag type")
	}
	if s.Tell() != initialPos {
		t.Fatalf("expected stream to rewind to %d, got %d", initialPos, s.Tell())
	}

	// Correct read should now succeed
	_, _, err = s.ReadTag(3, rmscene.TagByte4)
	if err != nil {
		t.Fatalf("expected successful read after rewind, got: %v", err)
	}
}

func TestCrdtId(t *testing.T) {
	id := rmscene.CrdtId{Part1: 1, Part2: 123456}

	var buf bytes.Buffer
	w := rmscene.NewDataWriter(&buf)
	if err := w.WriteId(5, id); err != nil {
		t.Fatal(err)
	}

	s := rmscene.NewDataStream(bytes.NewReader(buf.Bytes()))
	got, err := s.ReadId(5)
	if err != nil {
		t.Fatalf("failed to read CRDT ID: %v", err)
	}
	if got != id {
		t.Fatalf("expected %v, got %v", id, got)
	}
}

func TestPrimitives(t *testing.T) {
	var buf bytes.Buffer
	w := rmscene.NewDataWriter(&buf)

	if err := w.WriteTaggedBool(1, true); err != nil {
		t.Fatal(err)
	}
	if err := w.WriteTaggedByte(2, 42); err != nil {
		t.Fatal(err)
	}
	if err := w.WriteTaggedInt(3, 100200); err != nil {
		t.Fatal(err)
	}
	if err := w.WriteTaggedFloat(4, 12.34); err != nil {
		t.Fatal(err)
	}
	if err := w.WriteTaggedDouble(5, 56.789); err != nil {
		t.Fatal(err)
	}

	s := rmscene.NewDataStream(bytes.NewReader(buf.Bytes()))

	bVal, err := s.ReadTaggedBool(1)
	if err != nil || !bVal {
		t.Fatalf("expected true, got %v (%v)", bVal, err)
	}

	byteVal, err := s.ReadTaggedByte(2)
	if err != nil || byteVal != 42 {
		t.Fatalf("expected 42, got %v (%v)", byteVal, err)
	}

	intVal, err := s.ReadTaggedInt(3)
	if err != nil || intVal != 100200 {
		t.Fatalf("expected 100200, got %v (%v)", intVal, err)
	}

	floatVal, err := s.ReadTaggedFloat(4)
	if err != nil || math.Abs(float64(floatVal)-12.34) > 1e-4 {
		t.Fatalf("expected ~12.34, got %v (%v)", floatVal, err)
	}

	doubleVal, err := s.ReadTaggedDouble(5)
	if err != nil || math.Abs(doubleVal-56.789) > 1e-6 {
		t.Fatalf("expected ~56.789, got %v (%v)", doubleVal, err)
	}
}

func TestLwwValues(t *testing.T) {
	var buf bytes.Buffer
	w := rmscene.NewDataWriter(&buf)

	ts := rmscene.CrdtId{Part1: 1, Part2: 100}
	lwwBool := rmscene.LwwValue[bool]{Timestamp: ts, Value: true}
	lwwStr := rmscene.LwwValue[string]{Timestamp: ts, Value: "Layer 1"}

	// Write LwwBool: subblock at index 1
	err := w.WriteSubblock(1, func(sw *rmscene.DataWriter) error {
		if err := sw.WriteId(1, lwwBool.Timestamp); err != nil {
			return err
		}
		return sw.WriteTaggedBool(2, lwwBool.Value)
	})
	if err != nil {
		t.Fatal(err)
	}

	// Write LwwString: subblock at index 2
	err = w.WriteSubblock(2, func(sw *rmscene.DataWriter) error {
		if err := sw.WriteId(1, lwwStr.Timestamp); err != nil {
			return err
		}
		// String in subblock 2
		return sw.WriteSubblock(2, func(ssw *rmscene.DataWriter) error {
			if err := ssw.WriteVarUint(uint64(len(lwwStr.Value))); err != nil {
				return err
			}
			if err := ssw.WriteBool(true); err != nil {
				return err
			}
			return ssw.WriteBytes([]byte(lwwStr.Value))
		})
	})
	if err != nil {
		t.Fatal(err)
	}

	s := rmscene.NewDataStream(bytes.NewReader(buf.Bytes()))

	gotBool, err := s.ReadLwwBool(1)
	if err != nil {
		t.Fatalf("failed to read LwwBool: %v", err)
	}
	if gotBool != lwwBool {
		t.Fatalf("expected %v, got %v", lwwBool, gotBool)
	}

	gotStr, err := s.ReadLwwString(2)
	if err != nil {
		t.Fatalf("failed to read LwwString: %v", err)
	}
	if gotStr != lwwStr {
		t.Fatalf("expected %v, got %v", lwwStr, gotStr)
	}
}
