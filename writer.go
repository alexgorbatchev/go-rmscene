package rmscene

import (
	"bytes"
	"encoding/binary"
	"io"
	"math"
)

// DataWriter wraps an io.Writer to serialize reMarkable v6 binary structures.
type DataWriter struct {
	w io.Writer
}

// NewDataWriter creates a new DataWriter.
func NewDataWriter(w io.Writer) *DataWriter {
	return &DataWriter{w: w}
}

// WriteHeader writes the 43-byte magic header.
func (w *DataWriter) WriteHeader() error {
	return w.WriteBytes([]byte(HeaderV6))
}

// WriteBytes writes raw bytes to the stream.
func (w *DataWriter) WriteBytes(b []byte) error {
	_, err := w.w.Write(b)
	return err
}

// WriteUint8 writes a single uint8.
func (w *DataWriter) WriteUint8(v uint8) error {
	return w.WriteBytes([]byte{v})
}

// WriteBool writes a single boolean byte.
func (w *DataWriter) WriteBool(v bool) error {
	var b uint8
	if v {
		b = 1
	}
	return w.WriteUint8(b)
}

// WriteUint16 writes a 16-bit little-endian integer.
func (w *DataWriter) WriteUint16(v uint16) error {
	buf := make([]byte, 2)
	binary.LittleEndian.PutUint16(buf, v)
	return w.WriteBytes(buf)
}

// WriteUint32 writes a 32-bit little-endian integer.
func (w *DataWriter) WriteUint32(v uint32) error {
	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, v)
	return w.WriteBytes(buf)
}

// WriteFloat32 writes a 32-bit little-endian float.
func (w *DataWriter) WriteFloat32(v float32) error {
	return w.WriteUint32(math.Float32bits(v))
}

// WriteFloat64 writes a 64-bit little-endian double.
func (w *DataWriter) WriteFloat64(v float64) error {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, math.Float64bits(v))
	return w.WriteBytes(buf)
}

// WriteVarUint writes an unsigned LEB128 variable-length integer.
func (w *DataWriter) WriteVarUint(v uint64) error {
	for {
		toWrite := uint8(v & 0x7F)
		v >>= 7
		if v != 0 {
			if err := w.WriteUint8(toWrite | 0x80); err != nil {
				return err
			}
		} else {
			return w.WriteUint8(toWrite)
		}
	}
}

// WriteTag writes a tag with the given index and TagType.
func (w *DataWriter) WriteTag(index int, tagType TagType) error {
	x := uint64((index << 4) | int(tagType))
	return w.WriteVarUint(x)
}

// WriteCrdtId writes a CrdtId (part1 uint8 + part2 varuint).
func (w *DataWriter) WriteCrdtId(id CrdtId) error {
	if err := w.WriteUint8(id.Part1); err != nil {
		return err
	}
	return w.WriteVarUint(id.Part2)
}

// WriteId writes a tagged CrdtId.
func (w *DataWriter) WriteId(index int, id CrdtId) error {
	if err := w.WriteTag(index, TagID); err != nil {
		return err
	}
	return w.WriteCrdtId(id)
}

// WriteTaggedBool writes a tagged bool (TagByte1).
func (w *DataWriter) WriteTaggedBool(index int, v bool) error {
	if err := w.WriteTag(index, TagByte1); err != nil {
		return err
	}
	return w.WriteBool(v)
}

// WriteTaggedByte writes a tagged uint8 (TagByte1).
func (w *DataWriter) WriteTaggedByte(index int, v uint8) error {
	if err := w.WriteTag(index, TagByte1); err != nil {
		return err
	}
	return w.WriteUint8(v)
}

// WriteTaggedInt writes a tagged uint32 (TagByte4).
func (w *DataWriter) WriteTaggedInt(index int, v uint32) error {
	if err := w.WriteTag(index, TagByte4); err != nil {
		return err
	}
	return w.WriteUint32(v)
}

// WriteTaggedFloat writes a tagged float32 (TagByte4).
func (w *DataWriter) WriteTaggedFloat(index int, v float32) error {
	if err := w.WriteTag(index, TagByte4); err != nil {
		return err
	}
	return w.WriteFloat32(v)
}

// WriteTaggedDouble writes a tagged float64 (TagByte8).
func (w *DataWriter) WriteTaggedDouble(index int, v float64) error {
	if err := w.WriteTag(index, TagByte8); err != nil {
		return err
	}
	return w.WriteFloat64(v)
}

// WriteSubblock writes a subblock prefixed with tag index and length.
func (w *DataWriter) WriteSubblock(index int, writeFn func(sw *DataWriter) error) error {
	var buf bytes.Buffer
	subWriter := NewDataWriter(&buf)
	if err := writeFn(subWriter); err != nil {
		return err
	}
	if err := w.WriteTag(index, TagLength4); err != nil {
		return err
	}
	if err := w.WriteUint32(uint32(buf.Len())); err != nil {
		return err
	}
	return w.WriteBytes(buf.Bytes())
}

// WriteString writes a tagged string subblock.
func (w *DataWriter) WriteString(index int, s string) error {
	return w.WriteSubblock(index, func(sw *DataWriter) error {
		if err := sw.WriteVarUint(uint64(len(s))); err != nil {
			return err
		}
		if err := sw.WriteBool(true); err != nil {
			return err
		}
		return sw.WriteBytes([]byte(s))
	})
}

// WriteLwwString writes a tagged LwwValue[string].
func (w *DataWriter) WriteLwwString(index int, ts CrdtId, val string) error {
	return w.WriteSubblock(index, func(sw *DataWriter) error {
		if err := sw.WriteId(1, ts); err != nil {
			return err
		}
		return sw.WriteString(2, val)
	})
}

// WriteLwwBool writes a tagged LwwValue[bool].
func (w *DataWriter) WriteLwwBool(index int, ts CrdtId, val bool) error {
	return w.WriteSubblock(index, func(sw *DataWriter) error {
		if err := sw.WriteId(1, ts); err != nil {
			return err
		}
		return sw.WriteTaggedBool(2, val)
	})
}

// WriteLwwId writes a tagged LwwValue[CrdtId].
func (w *DataWriter) WriteLwwId(index int, ts CrdtId, val CrdtId) error {
	return w.WriteSubblock(index, func(sw *DataWriter) error {
		if err := sw.WriteId(1, ts); err != nil {
			return err
		}
		return sw.WriteId(2, val)
	})
}

// WriteLwwByte writes a tagged LwwValue[uint8].
func (w *DataWriter) WriteLwwByte(index int, ts CrdtId, val uint8) error {
	return w.WriteSubblock(index, func(sw *DataWriter) error {
		if err := sw.WriteId(1, ts); err != nil {
			return err
		}
		return sw.WriteTaggedByte(2, val)
	})
}

// WriteLwwFloat writes a tagged LwwValue[float32].
func (w *DataWriter) WriteLwwFloat(index int, ts CrdtId, val float32) error {
	return w.WriteSubblock(index, func(sw *DataWriter) error {
		if err := sw.WriteId(1, ts); err != nil {
			return err
		}
		return sw.WriteTaggedFloat(2, val)
	})
}

// WriteBlock writes a top-level block with 8-byte header and payload.
func (w *DataWriter) WriteBlock(blockType, minVersion, currentVersion uint8, writeFn func(bw *DataWriter) error) error {
	var buf bytes.Buffer
	blockWriter := NewDataWriter(&buf)
	if err := writeFn(blockWriter); err != nil {
		return err
	}
	// 8-byte header: Size (uint32), Unknown (0), MinVersion, CurrentVersion, BlockType
	if err := w.WriteUint32(uint32(buf.Len())); err != nil {
		return err
	}
	if err := w.WriteUint8(0); err != nil {
		return err
	}
	if err := w.WriteUint8(minVersion); err != nil {
		return err
	}
	if err := w.WriteUint8(currentVersion); err != nil {
		return err
	}
	if err := w.WriteUint8(blockType); err != nil {
		return err
	}
	return w.WriteBytes(buf.Bytes())
}

// WritePoint writes a Point in version 2 format (14 bytes) or version 1 format (24 bytes).
func (w *DataWriter) WritePoint(p Point, version uint8) error {
	if err := w.WriteFloat32(p.X); err != nil {
		return err
	}
	if err := w.WriteFloat32(p.Y); err != nil {
		return err
	}
	if version == 1 {
		if err := w.WriteFloat32(float32(p.Speed) / 4.0); err != nil {
			return err
		}
		if err := w.WriteFloat32(float32(p.Direction) * (2.0 * math.Pi) / 255.0); err != nil {
			return err
		}
		if err := w.WriteFloat32(float32(p.Width) / 4.0); err != nil {
			return err
		}
		return w.WriteFloat32(float32(p.Pressure) / 255.0)
	}
	// Version 2
	if err := w.WriteUint16(p.Speed); err != nil {
		return err
	}
	if err := w.WriteUint16(p.Width); err != nil {
		return err
	}
	if err := w.WriteUint8(p.Direction); err != nil {
		return err
	}
	return w.WriteUint8(p.Pressure)
}

// WriteLine writes a Line structure inside a subblock.
func (w *DataWriter) WriteLine(line *Line, version uint8) error {
	if err := w.WriteTaggedInt(1, uint32(line.Tool)); err != nil {
		return err
	}
	if err := w.WriteTaggedInt(2, uint32(line.Color)); err != nil {
		return err
	}
	if err := w.WriteTaggedDouble(3, line.ThicknessScale); err != nil {
		return err
	}
	if err := w.WriteTaggedFloat(4, line.StartingLength); err != nil {
		return err
	}
	// Points in subblock 5
	err := w.WriteSubblock(5, func(pw *DataWriter) error {
		for _, pt := range line.Points {
			if err := pw.WritePoint(pt, version); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	// Timestamp in Tag 6
	if err := w.WriteId(6, CrdtId{Part1: 0, Part2: 1}); err != nil {
		return err
	}
	if line.MoveID != nil {
		if err := w.WriteId(7, *line.MoveID); err != nil {
			return err
		}
	}
	return nil
}

// WriteSceneLineItemBlock writes a complete SceneLineItemBlock.
func (w *DataWriter) WriteSceneLineItemBlock(block *SceneLineItemBlock, version uint8) error {
	if version == 0 {
		version = 2
	}
	return w.WriteBlock(BlockTypeSceneLine, 1, version, func(bw *DataWriter) error {
		if err := bw.WriteId(1, block.ParentID); err != nil {
			return err
		}
		if err := bw.WriteId(2, block.Item.ItemID); err != nil {
			return err
		}
		if err := bw.WriteId(3, block.Item.LeftID); err != nil {
			return err
		}
		if err := bw.WriteId(4, block.Item.RightID); err != nil {
			return err
		}
		if err := bw.WriteTaggedInt(5, block.Item.DeletedLength); err != nil {
			return err
		}
		if block.Item.Value != nil {
			err := bw.WriteSubblock(6, func(sw *DataWriter) error {
				if err := sw.WriteUint8(0x03); err != nil { // item_type == 3 for Line
					return err
				}
				if err := sw.WriteLine(block.Item.Value, version); err != nil {
					return err
				}
				if len(block.ExtraValueData) > 0 {
					return sw.WriteBytes(block.ExtraValueData)
				}
				return nil
			})
			if err != nil {
				return err
			}
		}
		if len(block.extraData) > 0 {
			return bw.WriteBytes(block.extraData)
		}
		return nil
	})
}
