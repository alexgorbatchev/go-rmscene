package rmscene_test

import (
	"bytes"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/alexgorbatchev/go-rmscene"
)

type limitWriter struct {
	remaining int
}

func (l *limitWriter) Write(p []byte) (n int, err error) {
	if l.remaining <= 0 {
		return 0, errors.New("limit reached")
	}
	if len(p) > l.remaining {
		n = l.remaining
		l.remaining = 0
		return n, errors.New("limit reached")
	}
	l.remaining -= len(p)
	return len(p), nil
}

func TestCoverageBoost_TruncatedBlockPrefixes(t *testing.T) {
	// Build a buffer containing valid blocks of all types
	var buf bytes.Buffer
	dw := rmscene.NewDataWriter(&buf)
	ts := rmscene.CrdtId{Part1: 1, Part2: 1}

	// 1. SceneTreeBlock
	_ = dw.WriteBlock(rmscene.BlockTypeSceneTree, 1, 1, func(bw *rmscene.DataWriter) error {
		_ = bw.WriteId(1, ts)
		_ = bw.WriteId(2, ts)
		_ = bw.WriteTaggedBool(3, true)
		return bw.WriteSubblock(4, func(sw *rmscene.DataWriter) error {
			return sw.WriteId(1, ts)
		})
	})

	// 2. SceneGroupItemBlock
	_ = dw.WriteBlock(rmscene.BlockTypeSceneGroup, 1, 1, func(bw *rmscene.DataWriter) error {
		_ = bw.WriteId(1, ts)
		_ = bw.WriteId(2, ts)
		_ = bw.WriteId(3, ts)
		_ = bw.WriteId(4, ts)
		_ = bw.WriteTaggedInt(5, 0)
		return bw.WriteSubblock(6, func(sw *rmscene.DataWriter) error {
			_ = sw.WriteUint8(2)
			return sw.WriteId(2, ts)
		})
	})

	// 3. AuthorIdsBlock
	_ = dw.WriteBlock(rmscene.BlockTypeAuthorIds, 1, 1, func(bw *rmscene.DataWriter) error {
		_ = bw.WriteVarUint(1)
		return bw.WriteSubblock(0, func(sw *rmscene.DataWriter) error {
			_ = sw.WriteVarUint(4)
			_ = sw.WriteBytes([]byte("uuid"))
			return sw.WriteUint16(42)
		})
	})

	// 4. PageInfoBlock
	_ = dw.WriteBlock(rmscene.BlockTypePageInfo, 1, 1, func(bw *rmscene.DataWriter) error {
		_ = bw.WriteTaggedInt(1, 1)
		_ = bw.WriteTaggedInt(2, 2)
		_ = bw.WriteTaggedInt(3, 3)
		_ = bw.WriteTaggedInt(4, 4)
		return bw.WriteTaggedInt(5, 5)
	})

	// 5. SceneInfoBlock
	_ = dw.WriteBlock(rmscene.BlockTypeSceneInfo, 1, 1, func(bw *rmscene.DataWriter) error {
		_ = bw.WriteLwwId(1, ts, ts)
		_ = bw.WriteLwwBool(2, ts, true)
		return bw.WriteLwwBool(3, ts, false)
	})

	// 6. TreeNodeBlock
	_ = dw.WriteBlock(rmscene.BlockTypeTreeNode, 1, 1, func(bw *rmscene.DataWriter) error {
		_ = bw.WriteId(1, ts)
		_ = bw.WriteLwwString(2, ts, "Layer")
		_ = bw.WriteLwwBool(3, ts, true)
		_ = bw.WriteLwwId(7, ts, ts)
		_ = bw.WriteLwwByte(8, ts, 1)
		return bw.WriteLwwFloat(9, ts, 0.5)
	})

	// 7. SceneLineItemBlock
	pt := rmscene.Point{X: 1, Y: 2, Speed: 10, Direction: 1, Width: 2, Pressure: 100}
	line := &rmscene.Line{
		Tool:           rmscene.PenToolBallpoint1,
		Color:          rmscene.PenColorBlack,
		ThicknessScale: 1.0,
		Points:         []rmscene.Point{pt},
	}
	_ = dw.WriteBlock(rmscene.BlockTypeSceneLine, 1, 1, func(bw *rmscene.DataWriter) error {
		_ = bw.WriteId(1, ts)
		_ = bw.WriteId(2, ts)
		_ = bw.WriteId(3, ts)
		_ = bw.WriteId(4, ts)
		_ = bw.WriteTaggedInt(5, 0)
		return bw.WriteSubblock(6, func(sw *rmscene.DataWriter) error {
			_ = sw.WriteUint8(3)
			return sw.WriteLine(line, 2)
		})
	})

	fullBytes := buf.Bytes()
	// Test truncating at every single byte offset to exercise all reader error branches
	for i := 1; i < len(fullBytes); i += 2 {
		s := rmscene.NewDataStream(bytes.NewReader(fullBytes[:i]))
		for {
			_, err := rmscene.ReadBlock(s)
			if err != nil {
				break
			}
		}
	}
}

func TestCoverageBoost_TagMismatches(t *testing.T) {
	// Tag index mismatch
	var bufIdx bytes.Buffer
	dwIdx := rmscene.NewDataWriter(&bufIdx)
	_ = dwIdx.WriteTag(2, rmscene.TagByte1)
	sIdx := rmscene.NewDataStream(bytes.NewReader(bufIdx.Bytes()))
	if _, err := sIdx.ReadTaggedBool(1); err == nil {
		t.Error("expected tag index mismatch for ReadTaggedBool")
	}

	var bufByte bytes.Buffer
	dwByte := rmscene.NewDataWriter(&bufByte)
	_ = dwByte.WriteTag(2, rmscene.TagByte1)
	sByte := rmscene.NewDataStream(bytes.NewReader(bufByte.Bytes()))
	if _, err := sByte.ReadTaggedByte(1); err == nil {
		t.Error("expected tag index mismatch for ReadTaggedByte")
	}

	var bufInt bytes.Buffer
	dwInt := rmscene.NewDataWriter(&bufInt)
	_ = dwInt.WriteTag(2, rmscene.TagByte4)
	sInt := rmscene.NewDataStream(bytes.NewReader(bufInt.Bytes()))
	if _, err := sInt.ReadTaggedInt(1); err == nil {
		t.Error("expected tag index mismatch for ReadTaggedInt")
	}

	var bufFlt bytes.Buffer
	dwFlt := rmscene.NewDataWriter(&bufFlt)
	_ = dwFlt.WriteTag(2, rmscene.TagByte4)
	sFlt := rmscene.NewDataStream(bytes.NewReader(bufFlt.Bytes()))
	if _, err := sFlt.ReadTaggedFloat(1); err == nil {
		t.Error("expected tag index mismatch for ReadTaggedFloat")
	}

	var bufDbl bytes.Buffer
	dwDbl := rmscene.NewDataWriter(&bufDbl)
	_ = dwDbl.WriteTag(2, rmscene.TagByte8)
	sDbl := rmscene.NewDataStream(bytes.NewReader(bufDbl.Bytes()))
	if _, err := sDbl.ReadTaggedDouble(1); err == nil {
		t.Error("expected tag index mismatch for ReadTaggedDouble")
	}

	// Tag type mismatch (same index 1, but TagByte4 instead of TagByte1)
	var bufType bytes.Buffer
	dwType := rmscene.NewDataWriter(&bufType)
	_ = dwType.WriteTag(1, rmscene.TagByte4)
	sType := rmscene.NewDataStream(bytes.NewReader(bufType.Bytes()))
	if _, err := sType.ReadTaggedBool(1); err == nil {
		t.Error("expected tag type mismatch for ReadTaggedBool")
	}
}

func TestCoverageBoost_SubblockOverflow(t *testing.T) {
	var buf bytes.Buffer
	dw := rmscene.NewDataWriter(&buf)
	_ = dw.WriteTag(1, rmscene.TagLength4)
	_ = dw.WriteUint32(3)                                 // subLen = 3, but payload is 5 bytes!
	_ = dw.WriteId(1, rmscene.CrdtId{Part1: 1, Part2: 1}) // 3 bytes
	_ = dw.WriteTaggedBool(2, true)                       // 2 bytes
	s := rmscene.NewDataStream(bytes.NewReader(buf.Bytes()))
	if _, err := s.ReadLwwBool(1); !errors.Is(err, rmscene.ErrBlockOverflow) {
		t.Errorf("expected ErrBlockOverflow, got %v", err)
	}

	// Truncated padding
	var buf2 bytes.Buffer
	dw2 := rmscene.NewDataWriter(&buf2)
	_ = dw2.WriteTag(1, rmscene.TagLength4)
	_ = dw2.WriteUint32(20) // subLen = 20, but payload is only 5 bytes
	_ = dw2.WriteId(1, rmscene.CrdtId{Part1: 1, Part2: 1})
	_ = dw2.WriteTaggedBool(2, true)
	s2 := rmscene.NewDataStream(bytes.NewReader(buf2.Bytes()))
	if _, err := s2.ReadLwwBool(1); err == nil {
		t.Error("expected error on truncated padding")
	}
}

func TestReadLwwDirect(t *testing.T) {
	var buf bytes.Buffer
	dw := rmscene.NewDataWriter(&buf)
	ts := rmscene.CrdtId{Part1: 1, Part2: 2}
	_ = dw.WriteLwwBool(1, ts, true)
	_ = dw.WriteLwwByte(2, ts, 42)
	_ = dw.WriteLwwFloat(3, ts, 3.14)
	_ = dw.WriteLwwId(4, ts, rmscene.CrdtId{Part1: 3, Part2: 4})
	_ = dw.WriteLwwString(5, ts, "hello")
	_ = dw.WriteString(6, "direct string")

	s := rmscene.NewDataStream(bytes.NewReader(buf.Bytes()))
	bVal, err := s.ReadLwwBool(1)
	if err != nil || !bVal.Value {
		t.Fatal(err)
	}
	byVal, err := s.ReadLwwByte(2)
	if err != nil || byVal.Value != 42 {
		t.Fatal(err)
	}
	fVal, err := s.ReadLwwFloat(3)
	if err != nil || fVal.Value < 3.13 {
		t.Fatal(err)
	}
	idVal, err := s.ReadLwwId(4)
	if err != nil || idVal.Value.Part1 != 3 {
		t.Fatal(err)
	}
	sVal, err := s.ReadLwwString(5)
	if err != nil || sVal.Value != "hello" {
		t.Fatal(err)
	}
	strVal, err := s.ReadString(6)
	if err != nil || strVal != "direct string" {
		t.Fatal(err)
	}

	// ReadLww subblock truncated errors
	var truncBuf bytes.Buffer
	dwTrunc := rmscene.NewDataWriter(&truncBuf)
	_ = dwTrunc.WriteTag(1, rmscene.TagLength4)
	_ = dwTrunc.WriteUint32(50) // claims 50 bytes, but ends
	sTrunc := rmscene.NewDataStream(bytes.NewReader(truncBuf.Bytes()))
	if _, err := sTrunc.ReadLwwBool(1); err == nil {
		t.Error("expected error on truncated ReadLwwBool")
	}
	sTrunc2 := rmscene.NewDataStream(bytes.NewReader(truncBuf.Bytes()))
	if _, err := sTrunc2.ReadLwwByte(1); err == nil {
		t.Error("expected error on truncated ReadLwwByte")
	}
	sTrunc3 := rmscene.NewDataStream(bytes.NewReader(truncBuf.Bytes()))
	if _, err := sTrunc3.ReadLwwFloat(1); err == nil {
		t.Error("expected error on truncated ReadLwwFloat")
	}
	sTrunc4 := rmscene.NewDataStream(bytes.NewReader(truncBuf.Bytes()))
	if _, err := sTrunc4.ReadLwwId(1); err == nil {
		t.Error("expected error on truncated ReadLwwId")
	}
	sTrunc5 := rmscene.NewDataStream(bytes.NewReader(truncBuf.Bytes()))
	if _, err := sTrunc5.ReadLwwString(1); err == nil {
		t.Error("expected error on truncated ReadLwwString")
	}
	sTrunc6 := rmscene.NewDataStream(bytes.NewReader(truncBuf.Bytes()))
	if _, err := sTrunc6.ReadString(1); err == nil {
		t.Error("expected error on truncated ReadString")
	}
}

func TestCoverageBoost_PointVersions(t *testing.T) {
	pt := rmscene.Point{X: 10.5, Y: 20.5, Speed: 100, Direction: 128, Width: 50, Pressure: 200}

	// 1. Version 1 roundtrip
	var buf1 bytes.Buffer
	dw1 := rmscene.NewDataWriter(&buf1)
	if err := dw1.WritePoint(pt, 1); err != nil {
		t.Fatal(err)
	}
	line1 := &rmscene.Line{
		Tool:           rmscene.PenToolBallpoint1,
		Color:          rmscene.PenColorBlack,
		ThicknessScale: 1.0,
		Points:         []rmscene.Point{pt},
	}
	var blockBuf1 bytes.Buffer
	dwB1 := rmscene.NewDataWriter(&blockBuf1)
	err := dwB1.WriteBlock(rmscene.BlockTypeSceneLine, 1, 1, func(bw *rmscene.DataWriter) error {
		_ = bw.WriteId(1, rmscene.CrdtId{Part1: 1, Part2: 1})
		_ = bw.WriteId(2, rmscene.CrdtId{Part1: 2, Part2: 2})
		_ = bw.WriteId(3, rmscene.CrdtId{Part1: 3, Part2: 3})
		_ = bw.WriteId(4, rmscene.CrdtId{Part1: 4, Part2: 4})
		_ = bw.WriteTaggedInt(5, 0)
		return bw.WriteSubblock(6, func(sw *rmscene.DataWriter) error {
			_ = sw.WriteUint8(3)
			return sw.WriteLine(line1, 1)
		})
	})
	if err != nil {
		t.Fatal(err)
	}

	stream1 := rmscene.NewDataStream(bytes.NewReader(blockBuf1.Bytes()))
	b1, err := rmscene.ReadBlock(stream1)
	if err != nil {
		t.Fatalf("reading v1 line block: %v", err)
	}
	if b1 == nil {
		t.Fatal("expected non-nil block")
	}
}

func TestCoverageBoost_WriterErrorBranches(t *testing.T) {
	pt := rmscene.Point{X: 1, Y: 2, Speed: 100, Direction: 1, Width: 20, Pressure: 150}
	line := &rmscene.Line{
		Tool:           rmscene.PenToolBallpoint1,
		Color:          rmscene.PenColorBlack,
		ThicknessScale: 1.0,
		StartingLength: 0.5,
		Points:         []rmscene.Point{pt},
	}
	block := &rmscene.SceneLineItemBlock{
		Item:           rmscene.Item[*rmscene.Line]{Value: line},
		ExtraValueData: []byte{1, 2, 3},
	}

	for limit := 0; limit < 40; limit++ {
		dw := rmscene.NewDataWriter(&limitWriter{remaining: limit})
		_ = dw.WritePoint(pt, 1)
		_ = dw.WritePoint(pt, 2)
		_ = dw.WriteLine(line, 1)
		_ = dw.WriteLine(line, 2)
		_ = dw.WriteSceneLineItemBlock(block, 1)
		_ = dw.WriteSceneLineItemBlock(block, 2)
		_ = dw.WriteId(1, rmscene.CrdtId{Part1: 1, Part2: 1})
		_ = dw.WriteTaggedBool(1, true)
		_ = dw.WriteTaggedByte(1, 2)
		_ = dw.WriteTaggedInt(1, 3)
		_ = dw.WriteTaggedFloat(1, 4.0)
		_ = dw.WriteTaggedDouble(1, 5.0)
		_ = dw.WriteString(1, "test")
		_ = dw.WriteLwwString(1, rmscene.CrdtId{Part1: 1, Part2: 1}, "test")
		_ = dw.WriteLwwBool(1, rmscene.CrdtId{Part1: 1, Part2: 1}, true)
		_ = dw.WriteLwwId(1, rmscene.CrdtId{Part1: 1, Part2: 1}, rmscene.CrdtId{Part1: 2, Part2: 2})
		_ = dw.WriteLwwByte(1, rmscene.CrdtId{Part1: 1, Part2: 1}, 5)
		_ = dw.WriteLwwFloat(1, rmscene.CrdtId{Part1: 1, Part2: 1}, 6.0)
	}
}

func TestCoverageBoost_Types(t *testing.T) {
	tagTypes := []rmscene.TagType{
		rmscene.TagByte1,
		rmscene.TagByte4,
		rmscene.TagByte8,
		rmscene.TagLength4,
		rmscene.TagID,
		rmscene.TagType(0xAA),
	}
	for _, tt := range tagTypes {
		_ = tt.String()
	}

	colors := []rmscene.PenColor{
		rmscene.PenColorGrayOverlap,
		rmscene.PenColorGreen2,
		rmscene.PenColorCyan,
		rmscene.PenColorMagenta,
		rmscene.PenColorYellow2,
	}
	for _, c := range colors {
		_ = c.String()
		_ = rmscene.GetColor(c)
	}
}

func TestCoverageBoost_ParseRealFixture(t *testing.T) {
	data, err := os.ReadFile("testdata/oct1_notes_strokes.rm")
	if err != nil {
		t.Skip("fixture not found")
	}
	scene, err := rmscene.Parse(data)
	if err != nil {
		t.Fatalf("failed parsing fixture: %v", err)
	}
	if len(scene.Blocks) == 0 {
		t.Fatal("expected blocks in fixture")
	}
	svg := rmscene.RenderToSVG(scene.Blocks)
	if !strings.Contains(svg, "<svg") {
		t.Fatal("expected valid SVG")
	}

	// Empty RenderRMToSVG
	svgEmpty, err := rmscene.RenderRMToSVG(nil)
	if err != nil || !strings.Contains(svgEmpty, "<svg") {
		t.Fatalf("expected valid empty SVG")
	}

	// ParseReader error
	_, errReader := rmscene.ParseReader(strings.NewReader("bad header"))
	if errReader == nil {
		t.Error("expected error on bad header in ParseReader")
	}

	// RenderRMToSVG bad bytes
	_, errRender := rmscene.RenderRMToSVG([]byte("bad bytes"))
	if errRender == nil {
		t.Error("expected error on bad bytes in RenderRMToSVG")
	}
}

func TestCoverageBoost_BlockAndStreamErrors(t *testing.T) {
	// 1. Line with invalid pointsLen (not multiple of 14)
	var badPointsBuf bytes.Buffer
	dwBad := rmscene.NewDataWriter(&badPointsBuf)
	_ = dwBad.WriteBlock(rmscene.BlockTypeSceneLine, 1, 1, func(bw *rmscene.DataWriter) error {
		_ = bw.WriteId(1, rmscene.CrdtId{Part1: 1, Part2: 1})
		_ = bw.WriteId(2, rmscene.CrdtId{Part1: 2, Part2: 2})
		_ = bw.WriteId(3, rmscene.CrdtId{Part1: 3, Part2: 3})
		_ = bw.WriteId(4, rmscene.CrdtId{Part1: 4, Part2: 4})
		_ = bw.WriteTaggedInt(5, 0)
		return bw.WriteSubblock(6, func(sw *rmscene.DataWriter) error {
			_ = sw.WriteUint8(3)
			_ = sw.WriteTaggedInt(1, 1)
			_ = sw.WriteTaggedInt(2, 0)
			_ = sw.WriteTaggedDouble(3, 1.0)
			_ = sw.WriteTaggedFloat(4, 0.0)
			_ = sw.WriteTag(5, rmscene.TagLength4)
			_ = sw.WriteUint32(5) // 5 bytes: NOT multiple of 14!
			return sw.WriteBytes([]byte{1, 2, 3, 4, 5})
		})
	})
	sBad := rmscene.NewDataStream(bytes.NewReader(badPointsBuf.Bytes()))
	if _, err := rmscene.ReadBlock(sBad); err == nil {
		t.Error("expected point data size error")
	}

	// 2. Line subblock overflow (consumed > subLen)
	var ovfBuf bytes.Buffer
	dwOvf := rmscene.NewDataWriter(&ovfBuf)
	_ = dwOvf.WriteBlock(rmscene.BlockTypeSceneLine, 1, 1, func(bw *rmscene.DataWriter) error {
		_ = bw.WriteId(1, rmscene.CrdtId{Part1: 1, Part2: 1})
		_ = bw.WriteId(2, rmscene.CrdtId{Part1: 2, Part2: 2})
		_ = bw.WriteId(3, rmscene.CrdtId{Part1: 3, Part2: 3})
		_ = bw.WriteId(4, rmscene.CrdtId{Part1: 4, Part2: 4})
		_ = bw.WriteTaggedInt(5, 0)
		_ = bw.WriteTag(6, rmscene.TagLength4)
		_ = bw.WriteUint32(2)
		_ = bw.WriteUint8(3)
		_ = bw.WriteTaggedInt(1, 1)
		_ = bw.WriteTaggedInt(2, 0)
		return bw.WriteTaggedDouble(3, 1.0)
	})
	sOvf := rmscene.NewDataStream(bytes.NewReader(ovfBuf.Bytes()))
	if _, err := rmscene.ReadBlock(sOvf); err == nil {
		t.Error("expected overflow error")
	}

	// 3. Truncated blocks for all block types
	blockTypes := []uint8{
		rmscene.BlockTypeMigrationInfo,
		rmscene.BlockTypeSceneTree,
		rmscene.BlockTypeTreeNode,
		rmscene.BlockTypeSceneGroup,
		rmscene.BlockTypeSceneLine,
		rmscene.BlockTypeAuthorIds,
		rmscene.BlockTypePageInfo,
		rmscene.BlockTypeSceneInfo,
	}
	for _, bt := range blockTypes {
		var truncBuf bytes.Buffer
		dwT := rmscene.NewDataWriter(&truncBuf)
		_ = dwT.WriteBlock(bt, 1, 1, func(bw *rmscene.DataWriter) error {
			return bw.WriteUint8(42)
		})
		sTrunc := rmscene.NewDataStream(bytes.NewReader(truncBuf.Bytes()))
		_, _ = rmscene.ReadBlock(sTrunc)
	}
}

func TestCoverageBoost_StreamErrors(t *testing.T) {
	// Empty stream
	emptyStream := rmscene.NewDataStream(bytes.NewReader([]byte{}))
	if _, err := emptyStream.ReadUint8(); err == nil {
		t.Error("expected error on empty ReadUint8")
	}
	if _, err := emptyStream.ReadUint16(); err == nil {
		t.Error("expected error on empty ReadUint16")
	}
	if _, err := emptyStream.ReadUint32(); err == nil {
		t.Error("expected error on empty ReadUint32")
	}
	if _, err := emptyStream.ReadFloat32(); err == nil {
		t.Error("expected error on empty ReadFloat32")
	}
	if _, err := emptyStream.ReadFloat64(); err == nil {
		t.Error("expected error on empty ReadFloat64")
	}
	if _, err := emptyStream.ReadBool(); err == nil {
		t.Error("expected error on empty ReadBool")
	}
	if _, err := emptyStream.ReadVarUint(); err == nil {
		t.Error("expected error on empty ReadVarUint")
	}
	if _, err := emptyStream.ReadCrdtId(); err == nil {
		t.Error("expected error on empty ReadCrdtId")
	}
	if _, _, err := emptyStream.ReadTag(1, rmscene.TagID); err == nil {
		t.Error("expected error on empty ReadTag")
	}
	if _, err := emptyStream.ReadId(1); err == nil {
		t.Error("expected error on empty ReadId")
	}
	if _, err := emptyStream.ReadTaggedBool(1); err == nil {
		t.Error("expected error on empty ReadTaggedBool")
	}
	if _, err := emptyStream.ReadTaggedByte(1); err == nil {
		t.Error("expected error on empty ReadTaggedByte")
	}
	if _, err := emptyStream.ReadTaggedInt(1); err == nil {
		t.Error("expected error on empty ReadTaggedInt")
	}
	if _, err := emptyStream.ReadTaggedFloat(1); err == nil {
		t.Error("expected error on empty ReadTaggedFloat")
	}
	if _, err := emptyStream.ReadTaggedDouble(1); err == nil {
		t.Error("expected error on empty ReadTaggedDouble")
	}
	if _, err := emptyStream.ReadString(1); err == nil {
		t.Error("expected error on empty ReadString")
	}
	if _, err := emptyStream.ReadLwwBool(1); err == nil {
		t.Error("expected error on empty ReadLwwBool")
	}
	if _, err := emptyStream.ReadLwwByte(1); err == nil {
		t.Error("expected error on empty ReadLwwByte")
	}
	if _, err := emptyStream.ReadLwwFloat(1); err == nil {
		t.Error("expected error on empty ReadLwwFloat")
	}
	if _, err := emptyStream.ReadLwwId(1); err == nil {
		t.Error("expected error on empty ReadLwwId")
	}
	if _, err := emptyStream.ReadLwwString(1); err == nil {
		t.Error("expected error on empty ReadLwwString")
	}

	// Bad tag mismatch
	var tagBuf bytes.Buffer
	dw := rmscene.NewDataWriter(&tagBuf)
	_ = dw.WriteTag(2, rmscene.TagByte1) // tag index 2
	sMismatch := rmscene.NewDataStream(bytes.NewReader(tagBuf.Bytes()))
	if _, err := sMismatch.ReadId(1); err == nil {
		t.Error("expected tag mismatch error")
	}

	// ValidateHeader error
	if err := rmscene.ValidateHeader(bytes.NewReader([]byte("short"))); err == nil {
		t.Error("expected header error")
	}
	if err := rmscene.ValidateHeaderBytes([]byte("wrong header bytes")); err == nil {
		t.Error("expected header bytes error")
	}

	// Parse errors
	if _, err := rmscene.Parse([]byte("not valid")); err == nil {
		t.Error("expected parse error")
	}
}

func TestCoverageBoost_WriteLineVariations(t *testing.T) {
	var buf bytes.Buffer
	dw := rmscene.NewDataWriter(&buf)

	// Points in v1 and v2 formats
	pt1 := rmscene.Point{X: 10, Y: 20, Speed: 100, Direction: 1, Width: 20, Pressure: 150}
	if err := dw.WritePoint(pt1, 1); err != nil {
		t.Fatal(err)
	}
	if err := dw.WritePoint(pt1, 2); err != nil {
		t.Fatal(err)
	}

	// Line with multiple points and starting length
	line := &rmscene.Line{
		Tool:           rmscene.PenToolHighlighter1,
		Color:          rmscene.PenColorHighlight,
		ThicknessScale: 2.0,
		StartingLength: 5.5,
		Points:         []rmscene.Point{pt1},
	}
	if err := dw.WriteLine(line, 2); err != nil {
		t.Fatalf("WriteLine version 2 failed: %v", err)
	}
	if err := dw.WriteLine(line, 1); err != nil {
		t.Fatalf("WriteLine version 1 failed: %v", err)
	}

	// Line with nil Value
	emptyBlock := &rmscene.SceneLineItemBlock{
		Item: rmscene.Item[*rmscene.Line]{Value: nil},
	}
	if err := dw.WriteSceneLineItemBlock(emptyBlock, 1); err != nil {
		t.Fatal(err)
	}

	// Block with ExtraValueData
	extraBlock := &rmscene.SceneLineItemBlock{
		Item:           rmscene.Item[*rmscene.Line]{Value: line},
		ExtraValueData: []byte{0x84, 0x01, 100, 200, 50, 115},
	}
	if err := dw.WriteSceneLineItemBlock(extraBlock, 2); err != nil {
		t.Fatal(err)
	}

	// Error callbacks in WriteBlock and WriteSubblock
	failErr := errors.New("fail callback")
	if err := dw.WriteBlock(rmscene.BlockTypeSceneLine, 1, 1, func(bw *rmscene.DataWriter) error {
		return failErr
	}); !errors.Is(err, failErr) {
		t.Errorf("expected failErr from WriteBlock, got %v", err)
	}
	if err := dw.WriteSubblock(1, func(sw *rmscene.DataWriter) error {
		return failErr
	}); !errors.Is(err, failErr) {
		t.Errorf("expected failErr from WriteSubblock, got %v", err)
	}
}

func TestCoverageBoost_UnknownBlocks(t *testing.T) {
	var buf bytes.Buffer
	dw := rmscene.NewDataWriter(&buf)

	types := []uint8{
		rmscene.BlockTypeSceneGlyph,
		rmscene.BlockTypeSceneText,
		rmscene.BlockTypeRootText,
		rmscene.BlockTypeSceneTombstone,
	}
	for _, bt := range types {
		err := dw.WriteBlock(bt, 1, 1, func(bw *rmscene.DataWriter) error {
			return bw.WriteVarUint(123)
		})
		if err != nil {
			t.Fatal(err)
		}
	}

	stream := rmscene.NewDataStream(bytes.NewReader(buf.Bytes()))
	for _, bt := range types {
		b, err := rmscene.ReadBlock(stream)
		if err != nil {
			t.Fatalf("ReadBlock for type 0x%X failed: %v", bt, err)
		}
		if b.BlockType() != bt {
			t.Errorf("expected block type 0x%X, got 0x%X", bt, b.BlockType())
		}
	}
}
