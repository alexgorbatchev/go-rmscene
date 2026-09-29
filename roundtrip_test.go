package rmscene_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/alexgorbatchev/go-rmscene"
)

func TestEndToEnd_ParseAndRender(t *testing.T) {
	var buf bytes.Buffer
	w := rmscene.NewDataWriter(&buf)

	// 1. Write Header
	if err := w.WriteHeader(); err != nil {
		t.Fatal(err)
	}

	// 2. Write MigrationInfoBlock
	migBlock := &rmscene.MigrationInfoBlock{
		MigrationID: rmscene.CrdtId{Part1: 1, Part2: 1},
		IsDevice:    true,
	}
	err := w.WriteBlock(rmscene.BlockTypeMigrationInfo, 1, 1, func(bw *rmscene.DataWriter) error {
		if err := bw.WriteId(1, migBlock.MigrationID); err != nil {
			return err
		}
		return bw.WriteTaggedBool(2, migBlock.IsDevice)
	})
	if err != nil {
		t.Fatal(err)
	}

	// 3. Write Ballpoint Pen Stroke
	ballpointLine := &rmscene.Line{
		Color: rmscene.PenColorBlack,
		Tool:  rmscene.PenToolBallpoint1,
		Points: []rmscene.Point{
			{X: 100.0, Y: 150.0, Speed: 120, Direction: 45, Width: 12, Pressure: 180},
			{X: 105.0, Y: 155.0, Speed: 125, Direction: 48, Width: 12, Pressure: 185},
			{X: 110.0, Y: 160.0, Speed: 130, Direction: 50, Width: 12, Pressure: 190},
		},
		ThicknessScale: 1.5,
		StartingLength: 0.0,
	}
	ballpointBlock := &rmscene.SceneLineItemBlock{
		ParentID: rmscene.CrdtId{Part1: 1, Part2: 1},
		Item: rmscene.Item[*rmscene.Line]{
			ItemID:        rmscene.CrdtId{Part1: 1, Part2: 10},
			LeftID:        rmscene.CrdtId{Part1: 0, Part2: 0},
			RightID:       rmscene.CrdtId{Part1: 0, Part2: 0},
			DeletedLength: 0,
			Value:         ballpointLine,
		},
	}
	if err := w.WriteSceneLineItemBlock(ballpointBlock, 2); err != nil {
		t.Fatal(err)
	}

	// 4. Write Paper Pro Magenta Highlighter Stroke
	// Paper Pro Magenta in BGRA: Magenta RGB is (242, 158, 255)
	// B: 255 (0xFF), G: 158 (0x9E), R: 242 (0xF2), A: 0
	magentaExtra := []byte{0x84, 0x01, 0xFF, 0x9E, 0xF2, 0x00}
	hlLine := &rmscene.Line{
		Color: rmscene.PenColorHighlight,
		Tool:  rmscene.PenToolHighlighter1,
		Points: []rmscene.Point{
			{X: 80.0, Y: 140.0, Speed: 60, Direction: 0, Width: 20, Pressure: 200},
			{X: 130.0, Y: 140.0, Speed: 60, Direction: 0, Width: 20, Pressure: 200},
		},
		ThicknessScale: 1.0,
		StartingLength: 0.0,
	}
	hlBlock := &rmscene.SceneLineItemBlock{
		ParentID: rmscene.CrdtId{Part1: 1, Part2: 1},
		Item: rmscene.Item[*rmscene.Line]{
			ItemID:        rmscene.CrdtId{Part1: 1, Part2: 11},
			LeftID:        rmscene.CrdtId{Part1: 1, Part2: 10},
			RightID:       rmscene.CrdtId{Part1: 0, Part2: 0},
			DeletedLength: 0,
			Value:         hlLine,
		},
		ExtraValueData: magentaExtra,
	}
	if err := w.WriteSceneLineItemBlock(hlBlock, 2); err != nil {
		t.Fatal(err)
	}

	rawBytes := buf.Bytes()

	// Parse file
	scene, err := rmscene.Parse(rawBytes)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(scene.Blocks) != 3 {
		t.Fatalf("expected 3 blocks, got %d", len(scene.Blocks))
	}

	// Verify pen strokes and highlighters separation
	penLines := scene.Lines()
	if len(penLines) != 1 {
		t.Fatalf("expected 1 pen line, got %d", len(penLines))
	}
	if penLines[0].Tool != rmscene.PenToolBallpoint1 {
		t.Errorf("expected ballpoint tool, got %v", penLines[0].Tool)
	}

	hlLines := scene.Highlighters()
	if len(hlLines) != 1 {
		t.Fatalf("expected 1 highlighter line, got %d", len(hlLines))
	}
	if hlLines[0].Tool != rmscene.PenToolHighlighter1 {
		t.Errorf("expected highlighter tool, got %v", hlLines[0].Tool)
	}

	// Render to SVG
	svg, err := rmscene.RenderRMToSVG(rawBytes)
	if err != nil {
		t.Fatalf("RenderRMToSVG failed: %v", err)
	}

	// Verify SVG structure
	if !strings.Contains(svg, "<svg") || !strings.Contains(svg, "</svg>") {
		t.Fatal("invalid SVG output")
	}

	// Highlighters must precede pen strokes in DOM
	hlPos := strings.Index(svg, "<g id=\"highlighters\">")
	penPos := strings.Index(svg, "<g id=\"pen_strokes\">")
	if hlPos == -1 || penPos == -1 || hlPos >= penPos {
		t.Fatalf("highlighters must be rendered underneath pen strokes (hlPos=%d, penPos=%d)", hlPos, penPos)
	}

	// Check Magenta RGB (242, 158, 255) in SVG
	if !strings.Contains(svg, "rgb(242,158,255)") {
		t.Fatalf("expected Paper Pro Magenta rgb(242,158,255), got:\n%s", svg)
	}
}
