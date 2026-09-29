package rmscene

import (
	"errors"
	"fmt"
	"io"
	"math"
)

const (
	BlockTypeMigrationInfo = 0x00
	BlockTypeSceneTree     = 0x01
	BlockTypeTreeNode      = 0x02
	BlockTypeSceneGlyph    = 0x03
	BlockTypeSceneGroup    = 0x04
	BlockTypeSceneLine     = 0x05
	BlockTypeSceneText     = 0x06
	BlockTypeRootText      = 0x07
	BlockTypeSceneTombstone = 0x08
	BlockTypeAuthorIds     = 0x09
	BlockTypePageInfo      = 0x0A
	BlockTypeSceneInfo     = 0x0D
)

// SceneLineItemBlock represents a stroke line item block (BlockType 0x05).
type SceneLineItemBlock struct {
	ParentID       CrdtId      `json:"parent_id"`
	Item           Item[*Line] `json:"item"`
	ExtraValueData []byte      `json:"extra_value_data,omitempty"`
	extraData      []byte
}

func (b *SceneLineItemBlock) BlockType() uint8 { return BlockTypeSceneLine }
func (b *SceneLineItemBlock) ExtraData() []byte { return b.extraData }

// SceneGroupItemBlock represents a scene group item block (BlockType 0x04).
type SceneGroupItemBlock struct {
	ParentID  CrdtId       `json:"parent_id"`
	Item      Item[CrdtId] `json:"item"`
	extraData []byte
}

func (b *SceneGroupItemBlock) BlockType() uint8 { return BlockTypeSceneGroup }
func (b *SceneGroupItemBlock) ExtraData() []byte { return b.extraData }

// TreeNodeBlock represents a layer or tree node block (BlockType 0x02).
type TreeNodeBlock struct {
	Group     Group  `json:"group"`
	extraData []byte
}

func (b *TreeNodeBlock) BlockType() uint8 { return BlockTypeTreeNode }
func (b *TreeNodeBlock) ExtraData() []byte { return b.extraData }

// MigrationInfoBlock represents migration metadata (BlockType 0x00).
type MigrationInfoBlock struct {
	MigrationID CrdtId `json:"migration_id"`
	IsDevice    bool   `json:"is_device"`
	Unknown     bool   `json:"unknown,omitempty"`
	extraData   []byte
}

func (b *MigrationInfoBlock) BlockType() uint8 { return BlockTypeMigrationInfo }
func (b *MigrationInfoBlock) ExtraData() []byte { return b.extraData }

// SceneTreeBlock represents a hierarchy link in the scene tree (BlockType 0x01).
type SceneTreeBlock struct {
	TreeID    CrdtId `json:"tree_id"`
	NodeID    CrdtId `json:"node_id"`
	IsUpdate  bool   `json:"is_update"`
	ParentID  CrdtId `json:"parent_id"`
	extraData []byte
}

func (b *SceneTreeBlock) BlockType() uint8 { return BlockTypeSceneTree }
func (b *SceneTreeBlock) ExtraData() []byte { return b.extraData }

// AuthorIdsBlock maps author UUIDs to numerical IDs (BlockType 0x09).
type AuthorIdsBlock struct {
	Authors   map[uint16][]byte `json:"authors"`
	extraData []byte
}

func (b *AuthorIdsBlock) BlockType() uint8 { return BlockTypeAuthorIds }
func (b *AuthorIdsBlock) ExtraData() []byte { return b.extraData }

// PageInfoBlock holds document page statistics (BlockType 0x0A).
type PageInfoBlock struct {
	LoadsCount        uint32 `json:"loads_count"`
	MergesCount       uint32 `json:"merges_count"`
	TextCharsCount    uint32 `json:"text_chars_count"`
	TextLinesCount    uint32 `json:"text_lines_count"`
	TypeFolioUseCount uint32 `json:"type_folio_use_count,omitempty"`
	extraData         []byte
}

func (b *PageInfoBlock) BlockType() uint8 { return BlockTypePageInfo }
func (b *PageInfoBlock) ExtraData() []byte { return b.extraData }

// SceneInfoBlock holds root scene visibility and active layer info (BlockType 0x0D).
type SceneInfoBlock struct {
	CurrentLayer        LwwValue[CrdtId] `json:"current_layer"`
	BackgroundVisible   LwwValue[bool]   `json:"background_visible"`
	RootDocumentVisible LwwValue[bool]   `json:"root_document_visible"`
	extraData           []byte
}

func (b *SceneInfoBlock) BlockType() uint8 { return BlockTypeSceneInfo }
func (b *SceneInfoBlock) ExtraData() []byte { return b.extraData }

// UnknownBlock captures unhandled or forward-compatible block types safely.
type UnknownBlock struct {
	Type      uint8         `json:"block_type"`
	Info      MainBlockInfo `json:"info"`
	Data      []byte        `json:"data"`
	extraData []byte
}

func (b *UnknownBlock) BlockType() uint8 { return b.Type }
func (b *UnknownBlock) ExtraData() []byte { return b.extraData }

// ReadBlock parses the next top-level block from stream.
// Returns io.EOF when no more blocks remain.
func ReadBlock(s *DataStream) (Block, error) {
	size, err := s.ReadUint32()
	if err != nil {
		if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
			return nil, io.EOF
		}
		return nil, err
	}

	unknown, err := s.ReadUint8()
	if err != nil {
		return nil, err
	}
	minVersion, err := s.ReadUint8()
	if err != nil {
		return nil, err
	}
	currentVersion, err := s.ReadUint8()
	if err != nil {
		return nil, err
	}
	blockType, err := s.ReadUint8()
	if err != nil {
		return nil, err
	}

	startOffset := s.Tell()
	info := MainBlockInfo{
		Offset:         startOffset,
		Size:           size,
		Unknown:        unknown,
		MinVersion:     minVersion,
		CurrentVersion: currentVersion,
		BlockType:      blockType,
	}

	var block Block
	switch blockType {
	case BlockTypeSceneLine:
		block, err = readSceneLineItemBlock(s, info)
	case BlockTypeSceneGroup:
		block, err = readSceneGroupItemBlock(s, info)
	case BlockTypeTreeNode:
		block, err = readTreeNodeBlock(s, info)
	case BlockTypeMigrationInfo:
		block, err = readMigrationInfoBlock(s, info)
	case BlockTypeSceneTree:
		block, err = readSceneTreeBlock(s, info)
	case BlockTypeAuthorIds:
		block, err = readAuthorIdsBlock(s, info)
	case BlockTypePageInfo:
		block, err = readPageInfoBlock(s, info)
	case BlockTypeSceneInfo:
		block, err = readSceneInfoBlock(s, info)
	default:
		block, err = readUnknownBlock(s, info)
	}

	if err != nil {
		// Seek to end of block on error so caller can continue parsing if desired
		_, _ = s.Seek(startOffset+int64(size), io.SeekStart)
		return nil, err
	}

	// Consume any remaining bytes in block
	consumed := s.Tell() - startOffset
	if consumed > int64(size) {
		return nil, fmt.Errorf("%w: block type 0x%X at offset %d (overflow %d bytes)",
			ErrBlockOverflow, blockType, startOffset, consumed-int64(size))
	}
	if consumed < int64(size) {
		remaining := int64(size) - consumed
		extra, err := s.ReadBytes(int(remaining))
		if err != nil {
			return nil, err
		}
		if ub, ok := block.(*UnknownBlock); ok {
			ub.extraData = extra
		}
	}

	return block, nil
}

func readSceneLineItemBlock(s *DataStream, info MainBlockInfo) (*SceneLineItemBlock, error) {
	parentID, err := s.ReadId(1)
	if err != nil {
		return nil, err
	}
	itemID, err := s.ReadId(2)
	if err != nil {
		return nil, err
	}
	leftID, err := s.ReadId(3)
	if err != nil {
		return nil, err
	}
	rightID, err := s.ReadId(4)
	if err != nil {
		return nil, err
	}
	deletedLen, err := s.ReadTaggedInt(5)
	if err != nil {
		return nil, err
	}

	var line *Line
	var extraVal []byte

	if s.CheckTag(6, TagLength4) {
		if _, _, err := s.ReadTag(6, TagLength4); err != nil {
			return nil, err
		}
		subLen, err := s.ReadUint32()
		if err != nil {
			return nil, err
		}
		subStart := s.Tell()

		itemType, err := s.ReadUint8()
		if err != nil {
			return nil, err
		}
		_ = itemType // 0x03 for line item

		line, err = readLine(s, info.CurrentVersion)
		if err != nil {
			return nil, err
		}

		consumed := s.Tell() - subStart
		if consumed < int64(subLen) {
			remaining := int64(subLen) - consumed
			extraVal, err = s.ReadBytes(int(remaining))
			if err != nil {
				return nil, err
			}
		} else if consumed > int64(subLen) {
			return nil, fmt.Errorf("%w: line subblock overflow by %d bytes", ErrBlockOverflow, consumed-int64(subLen))
		}
	}

	return &SceneLineItemBlock{
		ParentID: parentID,
		Item: Item[*Line]{
			ItemID:        itemID,
			LeftID:        leftID,
			RightID:       rightID,
			DeletedLength: deletedLen,
			Value:         line,
		},
		ExtraValueData: extraVal,
	}, nil
}

func readLine(s *DataStream, version uint8) (*Line, error) {
	toolID, err := s.ReadTaggedInt(1)
	if err != nil {
		return nil, err
	}
	colorID, err := s.ReadTaggedInt(2)
	if err != nil {
		return nil, err
	}
	thicknessScale, err := s.ReadTaggedDouble(3)
	if err != nil {
		return nil, err
	}
	startingLength, err := s.ReadTaggedFloat(4)
	if err != nil {
		return nil, err
	}

	// Subblock 5: packed points
	if _, _, err := s.ReadTag(5, TagLength4); err != nil {
		return nil, err
	}
	pointsLen, err := s.ReadUint32()
	if err != nil {
		return nil, err
	}

	pointSize := uint32(14)
	if version == 1 {
		pointSize = 24
	}
	if pointsLen%pointSize != 0 {
		return nil, fmt.Errorf("point data size %d is not a multiple of point size %d", pointsLen, pointSize)
	}

	numPoints := pointsLen / pointSize
	points := make([]Point, numPoints)
	for i := uint32(0); i < numPoints; i++ {
		p, err := readPoint(s, version)
		if err != nil {
			return nil, err
		}
		points[i] = p
	}

	// Tag 6: timestamp
	_, err = s.ReadId(6)
	if err != nil {
		return nil, err
	}

	// Optional Tag 7: move_id
	var moveID *CrdtId
	if s.CheckTag(7, TagID) {
		mid, err := s.ReadId(7)
		if err == nil {
			moveID = &mid
		}
	}

	return &Line{
		Color:          PenColor(colorID),
		Tool:           PenTool(toolID),
		Points:         points,
		ThicknessScale: thicknessScale,
		StartingLength: startingLength,
		MoveID:         moveID,
	}, nil
}

func readPoint(s *DataStream, version uint8) (Point, error) {
	x, err := s.ReadFloat32()
	if err != nil {
		return Point{}, err
	}
	y, err := s.ReadFloat32()
	if err != nil {
		return Point{}, err
	}

	if version == 1 {
		speedF, err := s.ReadFloat32()
		if err != nil {
			return Point{}, err
		}
		directionF, err := s.ReadFloat32()
		if err != nil {
			return Point{}, err
		}
		widthF, err := s.ReadFloat32()
		if err != nil {
			return Point{}, err
		}
		pressureF, err := s.ReadFloat32()
		if err != nil {
			return Point{}, err
		}

		speed := uint16(math.Round(float64(speedF * 4.0)))
		direction := uint8(math.Round(float64(255.0 * directionF / (2.0 * math.Pi))))
		width := uint16(math.Round(float64(widthF * 4.0)))
		pressure := uint8(math.Round(float64(pressureF * 255.0)))

		return Point{
			X:         x,
			Y:         y,
			Speed:     speed,
			Direction: direction,
			Width:     width,
			Pressure:  pressure,
		}, nil
	}

	// Version 2: 14 bytes packed
	speed, err := s.ReadUint16()
	if err != nil {
		return Point{}, err
	}
	width, err := s.ReadUint16()
	if err != nil {
		return Point{}, err
	}
	direction, err := s.ReadUint8()
	if err != nil {
		return Point{}, err
	}
	pressure, err := s.ReadUint8()
	if err != nil {
		return Point{}, err
	}

	return Point{
		X:         x,
		Y:         y,
		Speed:     speed,
		Direction: direction,
		Width:     width,
		Pressure:  pressure,
	}, nil
}

func readSceneGroupItemBlock(s *DataStream, info MainBlockInfo) (*SceneGroupItemBlock, error) {
	parentID, err := s.ReadId(1)
	if err != nil {
		return nil, err
	}
	itemID, err := s.ReadId(2)
	if err != nil {
		return nil, err
	}
	leftID, err := s.ReadId(3)
	if err != nil {
		return nil, err
	}
	rightID, err := s.ReadId(4)
	if err != nil {
		return nil, err
	}
	deletedLen, err := s.ReadTaggedInt(5)
	if err != nil {
		return nil, err
	}

	var groupVal CrdtId
	if s.CheckTag(6, TagLength4) {
		if _, _, err := s.ReadTag(6, TagLength4); err != nil {
			return nil, err
		}
		subLen, err := s.ReadUint32()
		if err != nil {
			return nil, err
		}
		subStart := s.Tell()

		_, err = s.ReadUint8() // itemType (0x02)
		if err != nil {
			return nil, err
		}

		groupVal, err = s.ReadId(2)
		if err != nil {
			return nil, err
		}

		consumed := s.Tell() - subStart
		if consumed < int64(subLen) {
			if _, err := s.ReadBytes(int(int64(subLen) - consumed)); err != nil {
				return nil, err
			}
		}
	}

	return &SceneGroupItemBlock{
		ParentID: parentID,
		Item: Item[CrdtId]{
			ItemID:        itemID,
			LeftID:        leftID,
			RightID:       rightID,
			DeletedLength: deletedLen,
			Value:         groupVal,
		},
	}, nil
}

func readTreeNodeBlock(s *DataStream, info MainBlockInfo) (*TreeNodeBlock, error) {
	nodeID, err := s.ReadId(1)
	if err != nil {
		return nil, err
	}
	label, err := s.ReadLwwString(2)
	if err != nil {
		return nil, err
	}
	visible, err := s.ReadLwwBool(3)
	if err != nil {
		return nil, err
	}

	grp := Group{
		NodeID:  nodeID,
		Label:   label.Value,
		Visible: visible.Value,
	}

	// Optional anchor tags if remaining bytes exist in block
	if s.Tell() < info.Offset+int64(info.Size) && s.CheckTag(7, TagLength4) {
		anchorID, err := s.ReadLwwId(7)
		if err == nil {
			grp.AnchorID = &anchorID.Value
		}
	}
	if s.Tell() < info.Offset+int64(info.Size) && s.CheckTag(8, TagLength4) {
		anchorType, err := s.ReadLwwByte(8)
		if err == nil {
			grp.AnchorType = &anchorType.Value
		}
	}
	if s.Tell() < info.Offset+int64(info.Size) && s.CheckTag(9, TagLength4) {
		anchorThreshold, err := s.ReadLwwFloat(9)
		if err == nil {
			grp.AnchorThreshold = &anchorThreshold.Value
		}
	}
	if s.Tell() < info.Offset+int64(info.Size) && s.CheckTag(10, TagLength4) {
		anchorOriginX, err := s.ReadLwwFloat(10)
		if err == nil {
			grp.AnchorOriginX = &anchorOriginX.Value
		}
	}

	return &TreeNodeBlock{Group: grp}, nil
}

func readMigrationInfoBlock(s *DataStream, info MainBlockInfo) (*MigrationInfoBlock, error) {
	migID, err := s.ReadId(1)
	if err != nil {
		return nil, err
	}
	isDev, err := s.ReadTaggedBool(2)
	if err != nil {
		return nil, err
	}
	var unk bool
	if s.Tell() < info.Offset+int64(info.Size) && s.CheckTag(3, TagByte1) {
		unk, _ = s.ReadTaggedBool(3)
	}
	return &MigrationInfoBlock{MigrationID: migID, IsDevice: isDev, Unknown: unk}, nil
}

func readSceneTreeBlock(s *DataStream, info MainBlockInfo) (*SceneTreeBlock, error) {
	treeID, err := s.ReadId(1)
	if err != nil {
		return nil, err
	}
	nodeID, err := s.ReadId(2)
	if err != nil {
		return nil, err
	}
	isUpdate, err := s.ReadTaggedBool(3)
	if err != nil {
		return nil, err
	}
	if _, _, err := s.ReadTag(4, TagLength4); err != nil {
		return nil, err
	}
	subLen, err := s.ReadUint32()
	if err != nil {
		return nil, err
	}
	subStart := s.Tell()
	parentID, err := s.ReadId(1)
	if err != nil {
		return nil, err
	}
	consumed := s.Tell() - subStart
	if consumed < int64(subLen) {
		if _, err := s.ReadBytes(int(int64(subLen) - consumed)); err != nil {
			return nil, err
		}
	}
	return &SceneTreeBlock{
		TreeID:   treeID,
		NodeID:   nodeID,
		IsUpdate: isUpdate,
		ParentID: parentID,
	}, nil
}

func readAuthorIdsBlock(s *DataStream, info MainBlockInfo) (*AuthorIdsBlock, error) {
	numSubblocks, err := s.ReadVarUint()
	if err != nil {
		return nil, err
	}
	authors := make(map[uint16][]byte, numSubblocks)
	for i := uint64(0); i < numSubblocks; i++ {
		if _, _, err := s.ReadTag(0, TagLength4); err != nil {
			return nil, err
		}
		subLen, err := s.ReadUint32()
		if err != nil {
			return nil, err
		}
		subStart := s.Tell()

		uuidLen, err := s.ReadVarUint()
		if err != nil {
			return nil, err
		}
		uuidBytes, err := s.ReadBytes(int(uuidLen))
		if err != nil {
			return nil, err
		}
		authorID, err := s.ReadUint16()
		if err != nil {
			return nil, err
		}
		authors[authorID] = uuidBytes

		consumed := s.Tell() - subStart
		if consumed < int64(subLen) {
			if _, err := s.ReadBytes(int(int64(subLen) - consumed)); err != nil {
				return nil, err
			}
		}
	}
	return &AuthorIdsBlock{Authors: authors}, nil
}

func readPageInfoBlock(s *DataStream, info MainBlockInfo) (*PageInfoBlock, error) {
	loads, err := s.ReadTaggedInt(1)
	if err != nil {
		return nil, err
	}
	merges, err := s.ReadTaggedInt(2)
	if err != nil {
		return nil, err
	}
	chars, err := s.ReadTaggedInt(3)
	if err != nil {
		return nil, err
	}
	lines, err := s.ReadTaggedInt(4)
	if err != nil {
		return nil, err
	}
	var folio uint32
	if s.Tell() < info.Offset+int64(info.Size) && s.CheckTag(5, TagByte4) {
		folio, _ = s.ReadTaggedInt(5)
	}
	return &PageInfoBlock{
		LoadsCount:        loads,
		MergesCount:       merges,
		TextCharsCount:    chars,
		TextLinesCount:    lines,
		TypeFolioUseCount: folio,
	}, nil
}

func readSceneInfoBlock(s *DataStream, info MainBlockInfo) (*SceneInfoBlock, error) {
	currentLayer, err := s.ReadLwwId(1)
	if err != nil {
		return nil, err
	}
	bgVisible, err := s.ReadLwwBool(2)
	if err != nil {
		return nil, err
	}
	rootDocVisible, err := s.ReadLwwBool(3)
	if err != nil {
		return nil, err
	}
	return &SceneInfoBlock{
		CurrentLayer:        currentLayer,
		BackgroundVisible:   bgVisible,
		RootDocumentVisible: rootDocVisible,
	}, nil
}

func readUnknownBlock(s *DataStream, info MainBlockInfo) (*UnknownBlock, error) {
	data, err := s.ReadBytes(int(info.Size))
	if err != nil {
		return nil, err
	}
	return &UnknownBlock{
		Type: info.BlockType,
		Info: info,
		Data: data,
	}, nil
}
