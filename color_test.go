package rmscene_test

import (
	"testing"

	"github.com/alexgorbatchev/go-rmscene"
)

func TestExtractHighlighterColor_PaperPro(t *testing.T) {
	// BGRA = Blue: 229 (0xE5), Green: 208 (0xD0), Red: 139 (0x8B), Alpha: 0 (0x00)
	// Expected RGB: (139, 208, 229)
	extraData := []byte{0x84, 0x01, 0xE5, 0xD0, 0x8B, 0x00}
	col := rmscene.ExtractHighlighterColor(extraData)

	expected := rmscene.Color{R: 139, G: 208, B: 229}
	if col != expected {
		t.Fatalf("expected %v, got %v", expected, col)
	}
}

func TestExtractHighlighterColor_WithOffset(t *testing.T) {
	// Tag prefixed and suffixed by other data
	extraData := []byte{
		0x01, 0x02, 0x03, // unrelated prefix
		0x84, 0x01, // tag 8 Byte4
		0x19, 0xF7, 0xFB, 0x00, // BGRA: Blue 25, Green 247, Red 251 -> Yellow
		0xAA, 0xBB, // suffix
	}
	col := rmscene.ExtractHighlighterColor(extraData)

	expected := rmscene.Color{R: 251, G: 247, B: 25}
	if col != expected {
		t.Fatalf("expected %v, got %v", expected, col)
	}
}

func TestExtractHighlighterColor_MissingTag(t *testing.T) {
	extraData := []byte{0x10, 0x20, 0x30}
	col := rmscene.ExtractHighlighterColor(extraData)

	// Should fall back to standard yellow (255, 235, 59)
	if col != rmscene.DefaultHighlighterColor {
		t.Fatalf("expected default highlighter yellow %v, got %v", rmscene.DefaultHighlighterColor, col)
	}
}

func TestExtractHighlighterColor_Empty(t *testing.T) {
	col := rmscene.ExtractHighlighterColor(nil)
	if col != rmscene.DefaultHighlighterColor {
		t.Fatalf("expected default highlighter yellow %v, got %v", rmscene.DefaultHighlighterColor, col)
	}
}

func TestExtractHighlighterColor_Truncated(t *testing.T) {
	// Has 0x84 0x01 but only 2 following bytes instead of 4
	extraData := []byte{0x84, 0x01, 0xE5, 0xD0}
	col := rmscene.ExtractHighlighterColor(extraData)
	if col != rmscene.DefaultHighlighterColor {
		t.Fatalf("expected default fallback for truncated data, got %v", col)
	}
}

func TestExtractHighlighterColorWithFallback(t *testing.T) {
	customFallback := rmscene.Color{R: 200, G: 100, B: 50}
	col := rmscene.ExtractHighlighterColorWithFallback([]byte{0x00}, customFallback)
	if col != customFallback {
		t.Fatalf("expected custom fallback %v, got %v", customFallback, col)
	}
}

func TestColorHexAndString(t *testing.T) {
	c := rmscene.Color{R: 255, G: 128, B: 0}
	if c.Hex() != "#ff8000" {
		t.Fatalf("expected #ff8000, got %s", c.Hex())
	}
	if c.String() != "rgb(255,128,0)" {
		t.Fatalf("expected rgb(255,128,0), got %s", c.String())
	}
}

func TestPaletteDefaults(t *testing.T) {
	black := rmscene.GetColor(rmscene.PenColorBlack)
	if black != (rmscene.Color{R: 0, G: 0, B: 0}) {
		t.Fatalf("expected black color, got %v", black)
	}

	cyan := rmscene.GetColor(rmscene.PenColorCyan)
	if cyan != (rmscene.Color{R: 139, G: 208, B: 229}) {
		t.Fatalf("expected cyan (139, 208, 229), got %v", cyan)
	}
}
