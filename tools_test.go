package rmscene_test

import (
	"strings"
	"testing"

	"github.com/alexgorbatchev/go-rmscene"
)

func TestToolInstantiation(t *testing.T) {
	tests := []struct {
		tool            rmscene.PenTool
		colorID         rmscene.PenColor
		scale           float64
		expectedName    string
		expectedLinecap string
	}{
		{rmscene.PenToolBallpoint1, rmscene.PenColorBlack, 2.0, "Ballpoint", "round"},
		{rmscene.PenToolFineliner1, rmscene.PenColorBlue, 1.5, "Fineliner", "round"},
		{rmscene.PenToolMarker1, rmscene.PenColorRed, 2.0, "Marker", "round"},
		{rmscene.PenToolHighlighter1, rmscene.PenColorHighlight, 1.0, "Highlighter", "square"},
		{rmscene.PenToolPencil1, rmscene.PenColorGray, 1.0, "Pencil", "round"},
		{rmscene.PenToolMechanicalPencil1, rmscene.PenColorBlack, 2.0, "Mechanical Pencil", "round"},
		{rmscene.PenToolPaintbrush1, rmscene.PenColorGreen, 2.0, "Brush", "round"},
		{rmscene.PenToolCalligraphy, rmscene.PenColorBlack, 2.0, "Calligraphy", "round"},
		{rmscene.PenToolShader, rmscene.PenColorBlack, 1.0, "Shader", "round"},
		{rmscene.PenToolEraser, rmscene.PenColorWhite, 2.0, "Eraser", "square"},
		{rmscene.PenToolEraserArea, rmscene.PenColorWhite, 2.0, "Erase Area", "square"},
	}

	for _, tc := range tests {
		pen := rmscene.CreatePen(tc.tool, tc.colorID, tc.scale)
		if pen.Name() != tc.expectedName {
			t.Errorf("tool %v: expected name %q, got %q", tc.tool, tc.expectedName, pen.Name())
		}
		if pen.StrokeLinecap() != tc.expectedLinecap {
			t.Errorf("tool %v: expected linecap %q, got %q", tc.tool, tc.expectedLinecap, pen.StrokeLinecap())
		}
	}
}

func TestBallpointDynamics(t *testing.T) {
	pen := rmscene.CreatePen(rmscene.PenToolBallpoint1, rmscene.PenColorBlack, 2.0)

	// High pressure should increase segment width
	lowPressureWidth := pen.SegmentWidth(100, 50, 4, 50, 0)
	highPressureWidth := pen.SegmentWidth(100, 50, 4, 250, 0)
	if highPressureWidth <= lowPressureWidth {
		t.Fatalf("expected high pressure width (%f) > low pressure width (%f)", highPressureWidth, lowPressureWidth)
	}

	// High speed should decrease width
	fastWidth := pen.SegmentWidth(500, 50, 4, 150, 0)
	slowWidth := pen.SegmentWidth(10, 50, 4, 150, 0)
	if fastWidth >= slowWidth {
		t.Fatalf("expected fast width (%f) < slow width (%f)", fastWidth, slowWidth)
	}

	// Ballpoint segment color should be grayscale rgb string
	colorStr := pen.SegmentColor(100, 50, 4, 150, 0)
	if !strings.HasPrefix(colorStr, "rgb(") {
		t.Fatalf("expected rgb(...) color format, got %s", colorStr)
	}
}

func TestPencilDynamics(t *testing.T) {
	pen := rmscene.CreatePen(rmscene.PenToolPencil1, rmscene.PenColorGray, 1.0)

	// Pencil opacity should vary with pressure
	lowPressureOp := pen.SegmentOpacity(50, 10, 4, 50, 0)
	highPressureOp := pen.SegmentOpacity(50, 10, 4, 250, 0)
	if highPressureOp <= lowPressureOp {
		t.Fatalf("expected high pressure opacity (%f) > low pressure opacity (%f)", highPressureOp, lowPressureOp)
	}

	// Pencil segment width should never exceed max width (baseWidth * 10)
	w := pen.SegmentWidth(0, 0, 1000, 255, 0)
	if w > pen.BaseWidth()*10.0 {
		t.Fatalf("expected width %f <= %f", w, pen.BaseWidth()*10.0)
	}
}

func TestHighlighterIsHighlighter(t *testing.T) {
	if !rmscene.PenToolHighlighter1.IsHighlighter() {
		t.Fatal("expected PenToolHighlighter1.IsHighlighter() to be true")
	}
	if !rmscene.PenToolHighlighter2.IsHighlighter() {
		t.Fatal("expected PenToolHighlighter2.IsHighlighter() to be true")
	}
	if rmscene.PenToolBallpoint1.IsHighlighter() {
		t.Fatal("expected PenToolBallpoint1.IsHighlighter() to be false")
	}
}
