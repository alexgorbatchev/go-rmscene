package rmscene_test

import (
	"bytes"
	"math"
	"testing"

	"github.com/alexgorbatchev/go-rmscene"
)

func TestReadSceneLineItemBlock_Version2(t *testing.T) {
	var buf bytes.Buffer
	w := rmscene.NewDataWriter(&buf)

	pts := []rmscene.Point{
		{X: 10.5, Y: 20.25, Speed: 100, Direction: 50, Width: 16, Pressure: 200},
		{X: 15.0, Y: 25.5, Speed: 110, Direction: 55, Width: 18, Pressure: 210},
	}
	line := &rmscene.Line{
		Color:          rmscene.PenColorBlue,
		Tool:           rmscene.PenToolBallpoint1,
		Points:         pts,
		ThicknessScale: 1.5,
		StartingLength: 0.0,
	}
	block := &rmscene.SceneLineItemBlock{
		ParentID: rmscene.CrdtId{Part1: 1, Part2: 1},
		Item: rmscene.Item[*rmscene.Line]{
			ItemID:        rmscene.CrdtId{Part1: 1, Part2: 2},
			LeftID:        rmscene.CrdtId{Part1: 0, Part2: 0},
			RightID:       rmscene.CrdtId{Part1: 0, Part2: 0},
			DeletedLength: 0,
			Value:         line,
		},
		ExtraValueData: []byte{0x84, 0x01, 0x11, 0x22, 0x33, 0x00},
	}

	if err := w.WriteSceneLineItemBlock(block, 2); err != nil {
		t.Fatalf("failed to write SceneLineItemBlock: %v", err)
	}

	s := rmscene.NewDataStream(bytes.NewReader(buf.Bytes()))
	readB, err := rmscene.ReadBlock(s)
	if err != nil {
		t.Fatalf("failed to read block: %v", err)
	}

	lineBlock, ok := readB.(*rmscene.SceneLineItemBlock)
	if !ok {
		t.Fatalf("expected *SceneLineItemBlock, got %T", readB)
	}

	if lineBlock.ParentID != block.ParentID {
		t.Errorf("expected parentID %v, got %v", block.ParentID, lineBlock.ParentID)
	}
	if lineBlock.Item.ItemID != block.Item.ItemID {
		t.Errorf("expected itemID %v, got %v", block.Item.ItemID, lineBlock.Item.ItemID)
	}
	if lineBlock.Item.Value == nil {
		t.Fatal("expected non-nil line value")
	}

	readLine := lineBlock.Item.Value
	if readLine.Tool != rmscene.PenToolBallpoint1 {
		t.Errorf("expected tool %v, got %v", rmscene.PenToolBallpoint1, readLine.Tool)
	}
	if readLine.Color != rmscene.PenColorBlue {
		t.Errorf("expected color %v, got %v", rmscene.PenColorBlue, readLine.Color)
	}
	if math.Abs(readLine.ThicknessScale-1.5) > 1e-4 {
		t.Errorf("expected thickness scale 1.5, got %f", readLine.ThicknessScale)
	}
	if len(readLine.Points) != 2 {
		t.Fatalf("expected 2 points, got %d", len(readLine.Points))
	}
	if readLine.Points[0].X != 10.5 || readLine.Points[0].Y != 20.25 {
		t.Errorf("expected point 0 to be (10.5, 20.25), got (%f, %f)", readLine.Points[0].X, readLine.Points[0].Y)
	}
	if !bytes.Equal(lineBlock.ExtraValueData, block.ExtraValueData) {
		t.Errorf("expected extra_value_data %v, got %v", block.ExtraValueData, lineBlock.ExtraValueData)
	}
}

func TestReadSceneLineItemBlock_Version1Points(t *testing.T) {
	var buf bytes.Buffer
	w := rmscene.NewDataWriter(&buf)

	pts := []rmscene.Point{
		{X: 5.0, Y: 15.0, Speed: 100, Direction: 50, Width: 16, Pressure: 200},
	}
	line := &rmscene.Line{
		Color:          rmscene.PenColorBlack,
		Tool:           rmscene.PenToolFineliner1,
		Points:         pts,
		ThicknessScale: 1.0,
		StartingLength: 0.0,
	}
	block := &rmscene.SceneLineItemBlock{
		ParentID: rmscene.CrdtId{Part1: 1, Part2: 1},
		Item: rmscene.Item[*rmscene.Line]{
			ItemID:        rmscene.CrdtId{Part1: 1, Part2: 3},
			LeftID:        rmscene.CrdtId{Part1: 0, Part2: 0},
			RightID:       rmscene.CrdtId{Part1: 0, Part2: 0},
			DeletedLength: 0,
			Value:         line,
		},
	}

	if err := w.WriteSceneLineItemBlock(block, 1); err != nil {
		t.Fatalf("failed to write version 1 block: %v", err)
	}

	s := rmscene.NewDataStream(bytes.NewReader(buf.Bytes()))
	readB, err := rmscene.ReadBlock(s)
	if err != nil {
		t.Fatalf("failed to read version 1 block: %v", err)
	}

	lineBlock, ok := readB.(*rmscene.SceneLineItemBlock)
	if !ok {
		t.Fatalf("expected *SceneLineItemBlock, got %T", readB)
	}
	if len(lineBlock.Item.Value.Points) != 1 {
		t.Fatalf("expected 1 point, got %d", len(lineBlock.Item.Value.Points))
	}
	pt := lineBlock.Item.Value.Points[0]
	if pt.X != 5.0 || pt.Y != 15.0 {
		t.Errorf("expected point (5.0, 15.0), got (%f, %f)", pt.X, pt.Y)
	}
}

func TestReadTreeNodeBlock(t *testing.T) {
	var buf bytes.Buffer
	w := rmscene.NewDataWriter(&buf)

	nodeID := rmscene.CrdtId{Part1: 1, Part2: 10}
	ts := rmscene.CrdtId{Part1: 1, Part2: 1}

	err := w.WriteBlock(rmscene.BlockTypeTreeNode, 1, 1, func(bw *rmscene.DataWriter) error {
		if err := bw.WriteId(1, nodeID); err != nil {
			return err
		}
		// LwwString label in Tag 2
		if err := bw.WriteSubblock(2, func(sw *rmscene.DataWriter) error {
			if err := sw.WriteId(1, ts); err != nil {
				return err
			}
			return sw.WriteSubblock(2, func(ssw *rmscene.DataWriter) error {
				if err := ssw.WriteVarUint(uint64(len("Notes Layer"))); err != nil {
					return err
				}
				if err := ssw.WriteBool(true); err != nil {
					return err
				}
				return ssw.WriteBytes([]byte("Notes Layer"))
			})
		}); err != nil {
			return err
		}
		// LwwBool visible in Tag 3
		return bw.WriteSubblock(3, func(sw *rmscene.DataWriter) error {
			if err := sw.WriteId(1, ts); err != nil {
				return err
			}
			return sw.WriteTaggedBool(2, true)
		})
	})
	if err != nil {
		t.Fatal(err)
	}

	s := rmscene.NewDataStream(bytes.NewReader(buf.Bytes()))
	readB, err := rmscene.ReadBlock(s)
	if err != nil {
		t.Fatalf("failed to read TreeNodeBlock: %v", err)
	}

	treeNode, ok := readB.(*rmscene.TreeNodeBlock)
	if !ok {
		t.Fatalf("expected *TreeNodeBlock, got %T", readB)
	}
	if treeNode.Group.NodeID != nodeID {
		t.Errorf("expected nodeID %v, got %v", nodeID, treeNode.Group.NodeID)
	}
	if treeNode.Group.Label != "Notes Layer" {
		t.Errorf("expected label %q, got %q", "Notes Layer", treeNode.Group.Label)
	}
	if !treeNode.Group.Visible {
		t.Errorf("expected visible=true, got %v", treeNode.Group.Visible)
	}
}

func TestReadUnknownBlock(t *testing.T) {
	var buf bytes.Buffer
	w := rmscene.NewDataWriter(&buf)

	payload := []byte("some future remarkable feature data")
	err := w.WriteBlock(0x99, 1, 1, func(bw *rmscene.DataWriter) error {
		return bw.WriteBytes(payload)
	})
	if err != nil {
		t.Fatal(err)
	}

	s := rmscene.NewDataStream(bytes.NewReader(buf.Bytes()))
	readB, err := rmscene.ReadBlock(s)
	if err != nil {
		t.Fatalf("failed to read unknown block: %v", err)
	}

	ub, ok := readB.(*rmscene.UnknownBlock)
	if !ok {
		t.Fatalf("expected *UnknownBlock, got %T", readB)
	}
	if ub.Type != 0x99 {
		t.Errorf("expected type 0x99, got 0x%X", ub.Type)
	}
	if !bytes.Equal(ub.Data, payload) {
		t.Errorf("expected payload %q, got %q", payload, ub.Data)
	}
}

func TestReadSceneLineItemBlock_NoMoveID_FollowedByTag7(t *testing.T) {
	var buf bytes.Buffer
	w := rmscene.NewDataWriter(&buf)

	pts := []rmscene.Point{
		{X: 10.5, Y: 20.25, Speed: 100, Direction: 50, Width: 16, Pressure: 200},
	}
	line := &rmscene.Line{
		Color:          rmscene.PenColorBlack,
		Tool:           rmscene.PenToolBallpoint1,
		Points:         pts,
		ThicknessScale: 1.0,
		StartingLength: 0.0,
		MoveID:         nil, // NO MoveID
	}
	block := &rmscene.SceneLineItemBlock{
		ParentID: rmscene.CrdtId{Part1: 1, Part2: 1},
		Item: rmscene.Item[*rmscene.Line]{
			ItemID:        rmscene.CrdtId{Part1: 1, Part2: 2},
			LeftID:        rmscene.CrdtId{Part1: 0, Part2: 0},
			RightID:       rmscene.CrdtId{Part1: 0, Part2: 0},
			DeletedLength: 0,
			Value:         line,
		},
		ExtraValueData: nil,
	}

	if err := w.WriteSceneLineItemBlock(block, 2); err != nil {
		t.Fatalf("failed to write block: %v", err)
	}

	raw := buf.Bytes()
	// Trailing tag 7 (TagID = 0xF -> (7 << 4) | 0xF = 0x7F) followed by CrdtId (0x01, 0x05)
	trailingTag7 := []byte{0x7F, 0x01, 0x05}

	var buf2 bytes.Buffer
	w2 := rmscene.NewDataWriter(&buf2)
	blockPayload := append(raw[8:], trailingTag7...)
	if err := w2.WriteUint32(uint32(len(blockPayload))); err != nil {
		t.Fatal(err)
	}
	if err := w2.WriteUint8(0); err != nil {
		t.Fatal(err)
	}
	if err := w2.WriteUint8(1); err != nil {
		t.Fatal(err)
	}
	if err := w2.WriteUint8(2); err != nil {
		t.Fatal(err)
	}
	if err := w2.WriteUint8(uint8(rmscene.BlockTypeSceneLine)); err != nil {
		t.Fatal(err)
	}
	if _, err := buf2.Write(blockPayload); err != nil {
		t.Fatal(err)
	}

	s := rmscene.NewDataStream(bytes.NewReader(buf2.Bytes()))
	readB, err := rmscene.ReadBlock(s)
	if err != nil {
		t.Fatalf("expected successful block read without subblock overflow, got: %v", err)
	}
	lineB, ok := readB.(*rmscene.SceneLineItemBlock)
	if !ok {
		t.Fatalf("expected *SceneLineItemBlock, got %T", readB)
	}
	if lineB.Item.Value.MoveID != nil {
		t.Errorf("expected nil MoveID, got %v", lineB.Item.Value.MoveID)
	}
}
