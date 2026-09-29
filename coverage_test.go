package rmscene_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/alexgorbatchev/go-rmscene"
)

func TestCoverageBoost_TypesAndBlocks(t *testing.T) {
	// 1. BlockType and ExtraData on all block structs
	blocks := []rmscene.Block{
		&rmscene.SceneLineItemBlock{},
		&rmscene.SceneGroupItemBlock{},
		&rmscene.SceneTreeBlock{},
		&rmscene.TreeNodeBlock{},
		&rmscene.MigrationInfoBlock{},
		&rmscene.AuthorIdsBlock{},
		&rmscene.PageInfoBlock{},
		&rmscene.SceneInfoBlock{},
		&rmscene.UnknownBlock{},
	}
	for _, b := range blocks {
		_ = b.BlockType()
		_ = b.ExtraData()
	}

	// 2. Types String() methods
	tools := []rmscene.PenTool{
		rmscene.PenToolPaintbrush1,
		rmscene.PenToolPaintbrush2,
		rmscene.PenToolPencil1,
		rmscene.PenToolPencil2,
		rmscene.PenToolBallpoint1,
		rmscene.PenToolBallpoint2,
		rmscene.PenToolMarker1,
		rmscene.PenToolMarker2,
		rmscene.PenToolFineliner1,
		rmscene.PenToolFineliner2,
		rmscene.PenToolHighlighter1,
		rmscene.PenToolHighlighter2,
		rmscene.PenToolEraser,
		rmscene.PenToolEraserArea,
		rmscene.PenToolCalligraphy,
		rmscene.PenToolMechanicalPencil1,
		rmscene.PenToolMechanicalPencil2,
		rmscene.PenToolShader,
		rmscene.PenTool(999),
	}
	for _, tool := range tools {
		_ = tool.String()
		_ = tool.IsHighlighter()
	}

	colors := []rmscene.PenColor{
		rmscene.PenColorBlack,
		rmscene.PenColorGray,
		rmscene.PenColorWhite,
		rmscene.PenColorYellow,
		rmscene.PenColorGreen,
		rmscene.PenColorPink,
		rmscene.PenColorBlue,
		rmscene.PenColorRed,
		rmscene.PenColorHighlight,
		rmscene.PenColor(999),
	}
	for _, col := range colors {
		_ = col.String()
		_ = rmscene.GetColor(col)
	}

	// 3. CrdtId
	id := rmscene.CrdtId{Part1: 1, Part2: 2}
	if id.String() != "CrdtId(1, 2)" {
		t.Errorf("expected CrdtId(1, 2), got %s", id.String())
	}
	if id.IsZero() {
		t.Error("expected non-zero")
	}
	zeroID := rmscene.CrdtId{}
	if !zeroID.IsZero() {
		t.Error("expected zero")
	}

	// 4. SVG Options
	_ = rmscene.WithHighlighterOpacity(0.5)
	_ = rmscene.WithHighlighterWidth(12.0)
	_ = rmscene.WithViewBox("0 0 100 100")
	_ = rmscene.Scale(1.0)
}

func TestCoverageBoost_Tools(t *testing.T) {
	col := rmscene.Color{R: 10, G: 20, B: 30, Alpha: 0.8}

	pens := []rmscene.Pen{
		rmscene.NewBallpoint(2.0, col),
		rmscene.NewBrush(3.0, col),
		rmscene.NewCalligraphy(2.5, col),
		rmscene.NewEraser(4.0, col),
		rmscene.NewEraseArea(4.0, col),
		rmscene.NewFineliner(1.0, col),
		rmscene.NewHighlighter(10.0, col),
		rmscene.NewMarker(3.0, col),
		rmscene.NewMechanicalPencil(0.7, col),
		rmscene.NewPencil(1.5, col),
		rmscene.NewShader(16.0, col),
		rmscene.CreatePen(rmscene.PenToolBallpoint1, rmscene.PenColorBlack, 1.0),
		rmscene.CreatePenWithColor(rmscene.PenToolPaintbrush1, col, 1.0),
	}

	for _, pen := range pens {
		_ = pen.BaseColor()
		_ = pen.BaseWidth()
		_ = pen.SegmentLength()
		_ = pen.SegmentWidth(10.0, 1.0, 2.0, 0.5, 1.5)
		_ = pen.SegmentColor(10.0, 1.0, 2.0, 0.5, 1.5)
		_ = pen.SegmentOpacity(10.0, 1.0, 2.0, 0.5, 1.5)
	}
}

func TestCoverageBoost_ParserAndWriter(t *testing.T) {
	header := []byte(rmscene.HeaderV6)
	sc, err := rmscene.ParseReader(bytes.NewReader(header))
	if err != nil || sc == nil {
		t.Fatalf("ParseReader failed: %v", err)
	}

	svg, err := rmscene.RenderRMToSVG(header,
		rmscene.WithDimensions(200, 300),
		rmscene.WithHighlighterOpacity(0.4),
		rmscene.WithHighlighterWidth(15.0),
		rmscene.WithViewBox("-100 0 200 300"),
	)
	if err != nil || !strings.Contains(svg, "<svg") {
		t.Fatalf("RenderRMToSVG failed: %v", err)
	}

	var buf bytes.Buffer
	dw := rmscene.NewDataWriter(&buf)
	if err := dw.WriteHeader(); err != nil {
		t.Fatal(err)
	}
	if err := dw.WriteBool(true); err != nil {
		t.Fatal(err)
	}
	if err := dw.WriteUint8(1); err != nil {
		t.Fatal(err)
	}
	if err := dw.WriteUint16(2); err != nil {
		t.Fatal(err)
	}
	if err := dw.WriteUint32(3); err != nil {
		t.Fatal(err)
	}
	if err := dw.WriteFloat32(1.23); err != nil {
		t.Fatal(err)
	}
	if err := dw.WriteFloat64(4.56); err != nil {
		t.Fatal(err)
	}
	if err := dw.WriteTag(1, rmscene.TagID); err != nil {
		t.Fatal(err)
	}

	pt := rmscene.Point{X: 1, Y: 2, Speed: 3, Direction: 4, Width: 5, Pressure: 128}
	if err := dw.WritePoint(pt, 1); err != nil {
		t.Fatal(err)
	}
	line := rmscene.Line{
		Tool:           rmscene.PenToolBallpoint1,
		Color:          rmscene.PenColorBlack,
		ThicknessScale: 1.0,
		StartingLength: 0,
		Points:         []rmscene.Point{pt},
	}
	if err := dw.WriteLine(&line, 1); err != nil {
		t.Fatal(err)
	}

	block := &rmscene.SceneLineItemBlock{
		Item: rmscene.Item[*rmscene.Line]{
			Value: &line,
		},
	}
	if err := dw.WriteSceneLineItemBlock(block, 1); err != nil {
		t.Fatal(err)
	}
	if err := dw.WriteBlock(rmscene.BlockTypeSceneLine, 1, 1, func(bw *rmscene.DataWriter) error {
		return bw.WriteVarUint(99)
	}); err != nil {
		t.Fatal(err)
	}
}

func TestCoverageBoost_ReadBlocks(t *testing.T) {
	var buf bytes.Buffer
	dw := rmscene.NewDataWriter(&buf)

	// 1. Write and Read PageInfoBlock (type 0x0A)
	err := dw.WriteBlock(rmscene.BlockTypePageInfo, 1, 1, func(bw *rmscene.DataWriter) error {
		_ = bw.WriteTaggedInt(1, 10)  // loads
		_ = bw.WriteTaggedInt(2, 20)  // merges
		_ = bw.WriteTaggedInt(3, 300) // chars
		_ = bw.WriteTaggedInt(4, 40)  // lines
		return bw.WriteTaggedInt(5, 50) // folio
	})
	if err != nil {
		t.Fatal(err)
	}

	// 2. Write and Read AuthorIdsBlock (type 0x09) with padding
	err = dw.WriteBlock(rmscene.BlockTypeAuthorIds, 1, 1, func(bw *rmscene.DataWriter) error {
		_ = bw.WriteVarUint(1) // 1 author
		return bw.WriteSubblock(0, func(sw *rmscene.DataWriter) error {
			_ = sw.WriteVarUint(4)
			_ = sw.WriteBytes([]byte("uuid"))
			_ = sw.WriteUint16(42)
			return sw.WriteBytes([]byte("padding-bytes")) // padding
		})
	})
	if err != nil {
		t.Fatal(err)
	}

	// 3. Write and Read SceneTreeBlock (type 0x01) with padding
	err = dw.WriteBlock(rmscene.BlockTypeSceneTree, 1, 1, func(bw *rmscene.DataWriter) error {
		_ = bw.WriteId(1, rmscene.CrdtId{Part1: 1, Part2: 1})
		_ = bw.WriteId(2, rmscene.CrdtId{Part1: 2, Part2: 2})
		_ = bw.WriteTaggedBool(3, true)
		return bw.WriteSubblock(4, func(sw *rmscene.DataWriter) error {
			_ = sw.WriteId(1, rmscene.CrdtId{Part1: 3, Part2: 3})
			return sw.WriteBytes([]byte("tree-padding")) // padding
		})
	})
	if err != nil {
		t.Fatal(err)
	}

	// 4. Write and Read SceneGroupItemBlock (type 0x04) without tag 6
	err = dw.WriteBlock(rmscene.BlockTypeSceneGroup, 1, 1, func(bw *rmscene.DataWriter) error {
		_ = bw.WriteId(1, rmscene.CrdtId{Part1: 1, Part2: 1})
		_ = bw.WriteId(2, rmscene.CrdtId{Part1: 2, Part2: 2})
		_ = bw.WriteId(3, rmscene.CrdtId{Part1: 3, Part2: 3})
		_ = bw.WriteId(4, rmscene.CrdtId{Part1: 4, Part2: 4})
		return bw.WriteTaggedInt(5, 0)
	})
	if err != nil {
		t.Fatal(err)
	}

	// 5. Write and Read TreeNodeBlock (type 0x02) with LWW fields
	ts := rmscene.CrdtId{Part1: 1, Part2: 1}
	err = dw.WriteBlock(rmscene.BlockTypeTreeNode, 1, 1, func(bw *rmscene.DataWriter) error {
		_ = bw.WriteId(1, ts)
		_ = bw.WriteLwwString(2, ts, "Layer 1")
		_ = bw.WriteLwwBool(3, ts, true)
		_ = bw.WriteLwwId(7, ts, rmscene.CrdtId{Part1: 2, Part2: 2})
		_ = bw.WriteLwwByte(8, ts, 1)
		return bw.WriteLwwFloat(9, ts, 0.5)
	})
	if err != nil {
		t.Fatal(err)
	}

	// 6. Write and Read SceneInfoBlock (type 0x0D)
	err = dw.WriteBlock(rmscene.BlockTypeSceneInfo, 1, 1, func(bw *rmscene.DataWriter) error {
		_ = bw.WriteLwwId(1, ts, rmscene.CrdtId{Part1: 3, Part2: 3})
		_ = bw.WriteLwwBool(2, ts, true)
		return bw.WriteLwwBool(3, ts, false)
	})
	if err != nil {
		t.Fatal(err)
	}

	// 7. Write and Read MigrationInfoBlock (type 0x00)
	err = dw.WriteBlock(rmscene.BlockTypeMigrationInfo, 1, 1, func(bw *rmscene.DataWriter) error {
		_ = bw.WriteId(1, rmscene.CrdtId{Part1: 9, Part2: 9})
		_ = bw.WriteTaggedBool(2, true)
		return bw.WriteTaggedBool(3, false)
	})
	if err != nil {
		t.Fatal(err)
	}

	// 8. Unknown block (type 0xFE)
	err = dw.WriteBlock(0xFE, 1, 1, func(bw *rmscene.DataWriter) error {
		return bw.WriteBytes([]byte("unknown payload"))
	})
	if err != nil {
		t.Fatal(err)
	}

	// Read all blocks back
	stream := rmscene.NewDataStream(bytes.NewReader(buf.Bytes()))
	for i := 0; i < 8; i++ {
		b, err := rmscene.ReadBlock(stream)
		if err != nil {
			t.Fatalf("ReadBlock #%d failed: %v", i+1, err)
		}
		if b == nil {
			t.Fatalf("expected non-nil block #%d", i+1)
		}
	}
}
