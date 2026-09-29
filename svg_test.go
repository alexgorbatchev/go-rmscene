package rmscene_test

import (
	"strings"
	"testing"

	"github.com/alexgorbatchev/go-rmscene"
)

func TestRenderToSVG_Empty(t *testing.T) {
	svg := rmscene.RenderToSVG(nil)
	if !strings.HasPrefix(svg, "<svg") || !strings.HasSuffix(strings.TrimSpace(svg), "</svg>") {
		t.Fatalf("expected valid empty svg string, got: %s", svg)
	}
	if !strings.Contains(svg, "viewBox=\"-223.937 0.0 447.874 595.275\"") {
		t.Fatalf("expected default viewBox, got: %s", svg)
	}
}

func TestRenderToSVG_CustomOptions(t *testing.T) {
	svg := rmscene.RenderToSVG(nil,
		rmscene.WithDimensions(800, 600),
		rmscene.WithViewBox("0 0 800 600"),
	)
	if !strings.Contains(svg, "width=\"800.000\"") || !strings.Contains(svg, "height=\"600.000\"") {
		t.Fatalf("expected custom dimensions in svg: %s", svg)
	}
	if !strings.Contains(svg, "viewBox=\"0 0 800 600\"") {
		t.Fatalf("expected custom viewBox in svg: %s", svg)
	}
}

func TestRenderToSVG_PaintersOrderAndPaperProColor(t *testing.T) {
	// Create a Pen stroke
	penLine := &rmscene.Line{
		Color:          rmscene.PenColorBlack,
		Tool:           rmscene.PenToolBallpoint1,
		Points:         []rmscene.Point{{X: 10, Y: 10, Speed: 50, Direction: 20, Width: 4, Pressure: 150}, {X: 20, Y: 20, Speed: 50, Direction: 20, Width: 4, Pressure: 150}},
		ThicknessScale: 1.0,
	}
	penBlock := &rmscene.SceneLineItemBlock{
		Item: rmscene.Item[*rmscene.Line]{Value: penLine},
	}

	// Create a Highlighter stroke with Paper Pro Cyan extra_value_data
	hlLine := &rmscene.Line{
		Color:          rmscene.PenColorHighlight,
		Tool:           rmscene.PenToolHighlighter1,
		Points:         []rmscene.Point{{X: 30, Y: 30}, {X: 60, Y: 30}},
		ThicknessScale: 1.0,
	}
	// BGRA for RGB(139, 208, 229) -> B: 229 (0xE5), G: 208 (0xD0), R: 139 (0x8B), A: 0
	paperProCyanExtra := []byte{0x84, 0x01, 0xE5, 0xD0, 0x8B, 0x00}
	hlBlock := &rmscene.SceneLineItemBlock{
		Item:           rmscene.Item[*rmscene.Line]{Value: hlLine},
		ExtraValueData: paperProCyanExtra,
	}

	// Supply blocks: even if pen is supplied first in the blocks slice,
	// highlighters MUST be rendered underneath pen strokes in SVG painter's order!
	blocks := []rmscene.Block{penBlock, hlBlock}

	svg := rmscene.RenderToSVG(blocks)

	// Check painter's order: <g id="highlighters"> must come BEFORE <g id="pen_strokes">
	hlIdx := strings.Index(svg, "<g id=\"highlighters\">")
	penIdx := strings.Index(svg, "<g id=\"pen_strokes\">")
	if hlIdx == -1 {
		t.Fatal("missing <g id=\"highlighters\"> group in SVG")
	}
	if penIdx == -1 {
		t.Fatal("missing <g id=\"pen_strokes\"> group in SVG")
	}
	if hlIdx >= penIdx {
		t.Fatalf("expected highlighters (idx %d) to appear BEFORE pen strokes (idx %d)", hlIdx, penIdx)
	}

	// Check Paper Pro Cyan RGB color rendered
	if !strings.Contains(svg, "stroke=\"rgb(139,208,229)\"") {
		t.Fatalf("expected Paper Pro Cyan color rgb(139,208,229) in highlighters, got:\n%s", svg)
	}

	// Check highlighter stroke properties
	if !strings.Contains(svg, "stroke-opacity=\"0.45\"") {
		t.Fatalf("expected stroke-opacity=\"0.45\" in highlighters, got:\n%s", svg)
	}
	if !strings.Contains(svg, "stroke-linecap=\"square\"") {
		t.Fatalf("expected stroke-linecap=\"square\" in highlighters, got:\n%s", svg)
	}
	if !strings.Contains(svg, "stroke-width=\"10.0\"") {
		t.Fatalf("expected stroke-width=\"10.0\" in highlighters, got:\n%s", svg)
	}

	// Check pen stroke properties
	if !strings.Contains(svg, "stroke-linecap=\"round\"") {
		t.Fatalf("expected stroke-linecap=\"round\" in pen strokes, got:\n%s", svg)
	}
}
