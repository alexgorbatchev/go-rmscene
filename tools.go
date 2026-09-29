package rmscene

import (
	"fmt"
	"math"
)

// Pen is the interface implemented by all writing and drawing tools.
type Pen interface {
	Name() string
	SegmentLength() int
	StrokeLinecap() string
	BaseWidth() float64
	BaseColor() Color
	SegmentWidth(speed, direction, width, pressure, lastWidth float64) float64
	SegmentColor(speed, direction, width, pressure, lastWidth float64) string
	SegmentOpacity(speed, direction, width, pressure, lastWidth float64) float64
}

func clamp(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func directionToTilt(direction float64) float64 {
	return direction * (math.Pi * 2.0) / 255.0
}

// basePen provides common defaults for pen implementations.
type basePen struct {
	name          string
	baseWidth     float64
	baseColor     Color
	segmentLength int
	strokeLinecap string
	baseOpacity   float64
}

func (p *basePen) Name() string          { return p.name }
func (p *basePen) SegmentLength() int    { return p.segmentLength }
func (p *basePen) StrokeLinecap() string { return p.strokeLinecap }
func (p *basePen) BaseWidth() float64    { return p.baseWidth }
func (p *basePen) BaseColor() Color      { return p.baseColor }

func (p *basePen) SegmentWidth(speed, direction, width, pressure, lastWidth float64) float64 {
	return p.baseWidth
}

func (p *basePen) SegmentColor(speed, direction, width, pressure, lastWidth float64) string {
	return p.baseColor.String()
}

func (p *basePen) SegmentOpacity(speed, direction, width, pressure, lastWidth float64) float64 {
	return p.baseOpacity
}

// Fineliner pen implementation.
type Fineliner struct {
	basePen
}

func NewFineliner(width float64, color Color) *Fineliner {
	return &Fineliner{
		basePen: basePen{
			name:          "Fineliner",
			baseWidth:     width * 1.8,
			baseColor:     color,
			segmentLength: 1000,
			strokeLinecap: "round",
			baseOpacity:   1.0,
		},
	}
}

// Ballpoint pen implementation with dynamic width and ink pressure variation.
type Ballpoint struct {
	basePen
}

func NewBallpoint(width float64, color Color) *Ballpoint {
	return &Ballpoint{
		basePen: basePen{
			name:          "Ballpoint",
			baseWidth:     width,
			baseColor:     color,
			segmentLength: 5,
			strokeLinecap: "round",
			baseOpacity:   1.0,
		},
	}
}

func (p *Ballpoint) SegmentWidth(speed, direction, width, pressure, lastWidth float64) float64 {
	w := (0.5 + pressure/255.0) + (width / 4.0) - 0.5*((speed/4.0)/50.0)
	if w < 0.1 {
		return 0.1
	}
	return w
}

func (p *Ballpoint) SegmentColor(speed, direction, width, pressure, lastWidth float64) string {
	intensity := (0.1 * -((speed / 4.0) / 35.0)) + (1.2 * pressure / 255.0) + 0.5
	intensity = clamp(intensity, 0.0, 1.0)
	val := int(math.Min(math.Abs(intensity-1.0)*255.0, 60.0))
	return fmt.Sprintf("rgb(%d,%d,%d)", val, val, val)
}

// Marker pen implementation with tilt-sensitive width.
type Marker struct {
	basePen
}

func NewMarker(width float64, color Color) *Marker {
	return &Marker{
		basePen: basePen{
			name:          "Marker",
			baseWidth:     width,
			baseColor:     color,
			segmentLength: 3,
			strokeLinecap: "round",
			baseOpacity:   1.0,
		},
	}
}

func (p *Marker) SegmentWidth(speed, direction, width, pressure, lastWidth float64) float64 {
	tilt := directionToTilt(direction)
	w := 0.9*((width/4.0)-0.4*tilt) + (0.1 * lastWidth)
	if w < 0.1 {
		return 0.1
	}
	return w
}

// Pencil implementation with pressure, tilt and speed curves.
type Pencil struct {
	basePen
}

func NewPencil(width float64, color Color) *Pencil {
	return &Pencil{
		basePen: basePen{
			name:          "Pencil",
			baseWidth:     width,
			baseColor:     color,
			segmentLength: 2,
			strokeLinecap: "round",
			baseOpacity:   1.0,
		},
	}
}

func (p *Pencil) SegmentWidth(speed, direction, width, pressure, lastWidth float64) float64 {
	tilt := directionToTilt(direction)
	w := 0.7 * ((((0.8 * p.baseWidth) + (0.5 * pressure / 255.0)) * (width / 4.0)) - (0.25 * math.Pow(tilt, 1.8)) - (0.6 * (speed / 4.0) / 50.0))
	maxW := p.baseWidth * 10.0
	if w > maxW {
		w = maxW
	}
	if w < 0.1 {
		return 0.1
	}
	return w
}

func (p *Pencil) SegmentOpacity(speed, direction, width, pressure, lastWidth float64) float64 {
	op := (0.1 * -((speed / 4.0) / 35.0)) + (1.0 * pressure / 255.0)
	op = clamp(op, 0.0, 1.0) - 0.1
	if op < 0.0 {
		return 0.0
	}
	return op
}

// MechanicalPencil implementation.
type MechanicalPencil struct {
	basePen
}

func NewMechanicalPencil(width float64, color Color) *MechanicalPencil {
	return &MechanicalPencil{
		basePen: basePen{
			name:          "Mechanical Pencil",
			baseWidth:     width * width,
			baseColor:     color,
			segmentLength: 1000,
			strokeLinecap: "round",
			baseOpacity:   0.7,
		},
	}
}

// Brush (Paintbrush) implementation with pressure-sensitive color and width.
type Brush struct {
	basePen
}

func NewBrush(width float64, color Color) *Brush {
	return &Brush{
		basePen: basePen{
			name:          "Brush",
			baseWidth:     width,
			baseColor:     color,
			segmentLength: 2,
			strokeLinecap: "round",
			baseOpacity:   1.0,
		},
	}
}

func (p *Brush) SegmentWidth(speed, direction, width, pressure, lastWidth float64) float64 {
	tilt := directionToTilt(direction)
	w := 0.7 * (((1.0 + (1.4 * pressure / 255.0)) * (width / 4.0)) - (0.5 * tilt) - ((speed / 4.0) / 50.0))
	if w < 0.1 {
		return 0.1
	}
	return w
}

func (p *Brush) SegmentColor(speed, direction, width, pressure, lastWidth float64) string {
	intensity := (math.Pow(pressure/255.0, 1.5) - 0.2*((speed/4.0)/50.0)) * 1.5
	intensity = clamp(intensity, 0.0, 1.0)
	revIntensity := math.Abs(intensity - 1.0)
	r := int(revIntensity * float64(255-p.baseColor.R))
	g := int(revIntensity * float64(255-p.baseColor.G))
	b := int(revIntensity * float64(255-p.baseColor.B))
	return fmt.Sprintf("rgb(%d,%d,%d)", r, g, b)
}

// Calligraphy pen implementation.
type Calligraphy struct {
	basePen
}

func NewCalligraphy(width float64, color Color) *Calligraphy {
	return &Calligraphy{
		basePen: basePen{
			name:          "Calligraphy",
			baseWidth:     width,
			baseColor:     color,
			segmentLength: 2,
			strokeLinecap: "round",
			baseOpacity:   1.0,
		},
	}
}

func (p *Calligraphy) SegmentWidth(speed, direction, width, pressure, lastWidth float64) float64 {
	tilt := directionToTilt(direction)
	w := 0.9*(((1.0+pressure/255.0)*(width/4.0))-0.3*tilt) + (0.1 * lastWidth)
	if w < 0.1 {
		return 0.1
	}
	return w
}

// Highlighter implementation.
type Highlighter struct {
	basePen
}

func NewHighlighter(width float64, color Color) *Highlighter {
	return &Highlighter{
		basePen: basePen{
			name:          "Highlighter",
			baseWidth:     15.0,
			baseColor:     color,
			segmentLength: 1000,
			strokeLinecap: "square",
			baseOpacity:   0.3,
		},
	}
}

// Shader implementation.
type Shader struct {
	basePen
}

func NewShader(width float64, color Color) *Shader {
	return &Shader{
		basePen: basePen{
			name:          "Shader",
			baseWidth:     12.0,
			baseColor:     color,
			segmentLength: 1000,
			strokeLinecap: "round",
			baseOpacity:   0.1,
		},
	}
}

// Eraser implementation.
type Eraser struct {
	basePen
}

func NewEraser(width float64, color Color) *Eraser {
	return &Eraser{
		basePen: basePen{
			name:          "Eraser",
			baseWidth:     width * 2.0,
			baseColor:     Color{R: 255, G: 255, B: 255},
			segmentLength: 1000,
			strokeLinecap: "square",
			baseOpacity:   1.0,
		},
	}
}

// EraseArea implementation.
type EraseArea struct {
	basePen
}

func NewEraseArea(width float64, color Color) *EraseArea {
	return &EraseArea{
		basePen: basePen{
			name:          "Erase Area",
			baseWidth:     width,
			baseColor:     Color{R: 255, G: 255, B: 255},
			segmentLength: 1000,
			strokeLinecap: "square",
			baseOpacity:   0.0,
		},
	}
}

// CreatePen instantiates the appropriate Pen model given tool enum, color enum, and thickness scale.
func CreatePen(tool PenTool, colorID PenColor, thicknessScale float64) Pen {
	return CreatePenWithColor(tool, GetColor(colorID), thicknessScale)
}

// CreatePenWithColor instantiates the appropriate Pen model given tool enum, resolved Color, and thickness scale.
func CreatePenWithColor(tool PenTool, color Color, thicknessScale float64) Pen {
	switch tool {
	case PenToolPaintbrush1, PenToolPaintbrush2:
		return NewBrush(thicknessScale, color)
	case PenToolCalligraphy:
		return NewCalligraphy(thicknessScale, color)
	case PenToolMarker1, PenToolMarker2:
		return NewMarker(thicknessScale, color)
	case PenToolBallpoint1, PenToolBallpoint2:
		return NewBallpoint(thicknessScale, color)
	case PenToolFineliner1, PenToolFineliner2:
		return NewFineliner(thicknessScale, color)
	case PenToolPencil1, PenToolPencil2:
		return NewPencil(thicknessScale, color)
	case PenToolMechanicalPencil1, PenToolMechanicalPencil2:
		return NewMechanicalPencil(thicknessScale, color)
	case PenToolHighlighter1, PenToolHighlighter2:
		return NewHighlighter(thicknessScale, color)
	case PenToolShader:
		return NewShader(thicknessScale, color)
	case PenToolEraser:
		return NewEraser(thicknessScale, color)
	case PenToolEraserArea:
		return NewEraseArea(thicknessScale, color)
	default:
		return NewFineliner(thicknessScale, color)
	}
}
