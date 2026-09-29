package rmscene

import (
	"fmt"
)

// TagType represents the data type following a tag in the binary format.
type TagType uint8

const (
	TagByte1   TagType = 0x1 // 1 byte (bool or uint8)
	TagByte4   TagType = 0x4 // 4 bytes (uint32 or float32)
	TagByte8   TagType = 0x8 // 8 bytes (float64 / double)
	TagLength4 TagType = 0xC // 4-byte length prefix for subblock
	TagID      TagType = 0xF // CrdtId (1 byte part1 + varuint part2)
)

func (t TagType) String() string {
	switch t {
	case TagByte1:
		return "Byte1"
	case TagByte4:
		return "Byte4"
	case TagByte8:
		return "Byte8"
	case TagLength4:
		return "Length4"
	case TagID:
		return "ID"
	default:
		return fmt.Sprintf("TagType(0x%X)", uint8(t))
	}
}

// CrdtId is a unique identifier or timestamp in reMarkable CRDT structures.
type CrdtId struct {
	Part1 uint8  `json:"part1"`
	Part2 uint64 `json:"part2"`
}

func (id CrdtId) String() string {
	return fmt.Sprintf("CrdtId(%d, %d)", id.Part1, id.Part2)
}

func (id CrdtId) IsZero() bool {
	return id.Part1 == 0 && id.Part2 == 0
}

// LwwValue holds a value managed with Last-Write-Wins semantics by timestamp.
type LwwValue[T any] struct {
	Timestamp CrdtId `json:"timestamp"`
	Value     T      `json:"value"`
}

// Point represents a single point in a stroke with location and stylus telemetry.
type Point struct {
	X         float32 `json:"x"`
	Y         float32 `json:"y"`
	Speed     uint16  `json:"speed"`
	Direction uint8   `json:"direction"`
	Width     uint16  `json:"width"`
	Pressure  uint8   `json:"pressure"`
}

// PenColor represents the palette color index stored in the stroke line.
type PenColor int

const (
	PenColorBlack       PenColor = 0
	PenColorGray        PenColor = 1
	PenColorWhite       PenColor = 2
	PenColorYellow      PenColor = 3
	PenColorGreen       PenColor = 4
	PenColorPink        PenColor = 5
	PenColorBlue        PenColor = 6
	PenColorRed         PenColor = 7
	PenColorGrayOverlap PenColor = 8
	PenColorHighlight   PenColor = 9
	PenColorGreen2      PenColor = 10
	PenColorCyan        PenColor = 11
	PenColorMagenta     PenColor = 12
	PenColorYellow2     PenColor = 13
)

func (c PenColor) String() string {
	switch c {
	case PenColorBlack:
		return "Black"
	case PenColorGray:
		return "Gray"
	case PenColorWhite:
		return "White"
	case PenColorYellow:
		return "Yellow"
	case PenColorGreen:
		return "Green"
	case PenColorPink:
		return "Pink"
	case PenColorBlue:
		return "Blue"
	case PenColorRed:
		return "Red"
	case PenColorGrayOverlap:
		return "GrayOverlap"
	case PenColorHighlight:
		return "Highlight"
	case PenColorGreen2:
		return "Green2"
	case PenColorCyan:
		return "Cyan"
	case PenColorMagenta:
		return "Magenta"
	case PenColorYellow2:
		return "Yellow2"
	default:
		return fmt.Sprintf("PenColor(%d)", int(c))
	}
}

// PenTool represents the writing or drawing instrument used for a stroke.
type PenTool int

const (
	PenToolPaintbrush1       PenTool = 0
	PenToolPencil1           PenTool = 1
	PenToolBallpoint1        PenTool = 2
	PenToolMarker1           PenTool = 3
	PenToolFineliner1        PenTool = 4
	PenToolHighlighter1      PenTool = 5
	PenToolEraser            PenTool = 6
	PenToolMechanicalPencil1 PenTool = 7
	PenToolEraserArea        PenTool = 8
	PenToolPaintbrush2       PenTool = 12
	PenToolMechanicalPencil2 PenTool = 13
	PenToolPencil2           PenTool = 14
	PenToolBallpoint2        PenTool = 15
	PenToolMarker2           PenTool = 16
	PenToolFineliner2        PenTool = 17
	PenToolHighlighter2      PenTool = 18
	PenToolCalligraphy       PenTool = 21
	PenToolShader            PenTool = 23
)

func (t PenTool) String() string {
	switch t {
	case PenToolPaintbrush1, PenToolPaintbrush2:
		return "Paintbrush"
	case PenToolPencil1, PenToolPencil2:
		return "Pencil"
	case PenToolBallpoint1, PenToolBallpoint2:
		return "Ballpoint"
	case PenToolMarker1, PenToolMarker2:
		return "Marker"
	case PenToolFineliner1, PenToolFineliner2:
		return "Fineliner"
	case PenToolHighlighter1, PenToolHighlighter2:
		return "Highlighter"
	case PenToolEraser:
		return "Eraser"
	case PenToolMechanicalPencil1, PenToolMechanicalPencil2:
		return "Mechanical Pencil"
	case PenToolEraserArea:
		return "Eraser Area"
	case PenToolCalligraphy:
		return "Calligraphy"
	case PenToolShader:
		return "Shader"
	default:
		return fmt.Sprintf("PenTool(%d)", int(t))
	}
}

// IsHighlighter returns true if the tool is any variation of the highlighter or shader wash.
func (t PenTool) IsHighlighter() bool {
	return t == PenToolHighlighter1 || t == PenToolHighlighter2 || t == PenToolShader
}

// Color represents an RGB color with optional alpha transparency.
type Color struct {
	R     uint8   `json:"r"`
	G     uint8   `json:"g"`
	B     uint8   `json:"b"`
	Alpha float64 `json:"alpha,omitempty"`
}

func (c Color) String() string {
	return fmt.Sprintf("rgb(%d,%d,%d)", c.R, c.G, c.B)
}

// Line represents a single stroke on the tablet.
type Line struct {
	Color          PenColor `json:"color"`
	Tool           PenTool  `json:"tool"`
	Points         []Point  `json:"points"`
	ThicknessScale float64  `json:"thickness_scale"`
	StartingLength float32  `json:"starting_length"`
	MoveID         *CrdtId  `json:"move_id,omitempty"`
}

// Item represents a CRDT sequence item wrapping a value.
type Item[T any] struct {
	ItemID        CrdtId `json:"item_id"`
	LeftID        CrdtId `json:"left_id"`
	RightID       CrdtId `json:"right_id"`
	DeletedLength uint32 `json:"deleted_length"`
	Value         T      `json:"value"`
}

// Group represents a scene tree group node (e.g. layer).
type Group struct {
	NodeID          CrdtId   `json:"node_id"`
	Label           string   `json:"label"`
	Visible         bool     `json:"visible"`
	AnchorID        *CrdtId  `json:"anchor_id,omitempty"`
	AnchorType      *uint8   `json:"anchor_type,omitempty"`
	AnchorThreshold *float32 `json:"anchor_threshold,omitempty"`
	AnchorOriginX   *float32 `json:"anchor_origin_x,omitempty"`
}

// Block is the common interface for top-level tagged blocks in a v6 .rm file.
type Block interface {
	BlockType() uint8
	ExtraData() []byte
}

// MainBlockInfo holds metadata about a top-level block.
type MainBlockInfo struct {
	Offset         int64
	Size           uint32
	Unknown        uint8
	MinVersion     uint8
	CurrentVersion uint8
	BlockType      uint8
	ExtraData      []byte
}

// SubBlockInfo holds metadata about a subblock.
type SubBlockInfo struct {
	Offset    int64
	Size      uint32
	ExtraData []byte
}
