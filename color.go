package rmscene

import (
	"bytes"
	"fmt"
)

// DefaultHighlighterColor is standard highlighter yellow (255, 235, 59).
var DefaultHighlighterColor = Color{R: 255, G: 235, B: 59}

// DefaultPalette maps PenColor enum values to their standard display RGB colors.
// Values incorporate Paper Pro color palette definitions.
var DefaultPalette = map[PenColor]Color{
	PenColorBlack:       {R: 0, G: 0, B: 0},
	PenColorGray:        {R: 144, G: 144, B: 144},
	PenColorWhite:       {R: 255, G: 255, B: 255},
	PenColorYellow:      {R: 251, G: 247, B: 25},
	PenColorGreen:       {R: 0, G: 255, B: 0},
	PenColorPink:        {R: 255, G: 192, B: 203},
	PenColorBlue:        {R: 78, G: 105, B: 201},
	PenColorRed:         {R: 179, G: 62, B: 57},
	PenColorGrayOverlap: {R: 125, G: 125, B: 125},
	PenColorHighlight:   {R: 251, G: 247, B: 25},
	PenColorGreen2:      {R: 161, G: 216, B: 125},
	PenColorCyan:        {R: 139, G: 208, B: 229},
	PenColorMagenta:     {R: 242, G: 158, B: 255},
	PenColorYellow2:     {R: 247, G: 232, B: 81},
}

// GetColor returns the RGB color for a given PenColor, falling back to black if unknown.
func GetColor(c PenColor) Color {
	if col, ok := DefaultPalette[c]; ok {
		return col
	}
	return Color{R: 0, G: 0, B: 0}
}

// Hex returns the #RRGGBB hexadecimal representation of the color.
func (c Color) Hex() string {
	return fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B)
}

// ExtractHighlighterColor extracts a 24-bit RGB color from Paper Pro extra_value_data.
//
// Paper Pro stores color in little-endian BGRA format following tag 0x84 0x01
// (tag index 8, type Byte4).
// Defaults to standard highlighter yellow (255, 235, 59) if tag 0x84 0x01 is absent or malformed.
func ExtractHighlighterColor(extraBytes []byte) Color {
	return ExtractHighlighterColorWithFallback(extraBytes, DefaultHighlighterColor)
}

// ExtractHighlighterColorWithFallback extracts a 24-bit RGB color from Paper Pro extra_value_data,
// returning fallback if tag 0x84 0x01 is absent or malformed.
func ExtractHighlighterColorWithFallback(extraBytes []byte, fallback Color) Color {
	if len(extraBytes) < 6 {
		return fallback
	}
	idx := bytes.Index(extraBytes, []byte{0x84, 0x01})
	if idx == -1 || len(extraBytes) < idx+6 {
		return fallback
	}
	bVal := extraBytes[idx+2]
	gVal := extraBytes[idx+3]
	rVal := extraBytes[idx+4]
	// extraBytes[idx+5] is alpha (typically 0x00 or ignored for stroke color)
	return Color{R: rVal, G: gVal, B: bVal}
}
