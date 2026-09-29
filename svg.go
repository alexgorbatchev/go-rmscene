package rmscene

import (
	"fmt"
	"strings"
)

const (
	// DefaultWidthPt is the standard reMarkable A5 portrait width in points.
	DefaultWidthPt = 447.874
	// DefaultHeightPt is the standard reMarkable A5 portrait height in points.
	DefaultHeightPt = 595.275
	// DefaultHighlighterOpacity is the standard stroke opacity for highlighter ink.
	DefaultHighlighterOpacity = 0.45
	// DefaultHighlighterWidth is the standard stroke width for highlighters.
	DefaultHighlighterWidth = 10.0
	// ScreenDPI is the hardware screen DPI of the reMarkable tablet (226 DPI).
	ScreenDPI = 226.0
	// ScaleFactor converts tablet screen units to standard 72 DPI points (72 / 226).
	ScaleFactor = 72.0 / ScreenDPI
)

// Scale converts a raw coordinate or dimension from screen units to SVG points.
func Scale(unit float64) float64 {
	return unit * ScaleFactor
}

// SVGOptions configures the SVG exporter output.
type SVGOptions struct {
	WidthPt            float64
	HeightPt           float64
	ViewBox            string
	HighlighterOpacity float64
	HighlighterWidth   float64
}

// SVGOption is a functional option for configuring SVG output.
type SVGOption func(*SVGOptions)

// WithDimensions sets explicit SVG width and height in points.
func WithDimensions(widthPt, heightPt float64) SVGOption {
	return func(o *SVGOptions) {
		o.WidthPt = widthPt
		o.HeightPt = heightPt
	}
}

// WithViewBox sets an explicit viewBox attribute on the root SVG element.
func WithViewBox(viewBox string) SVGOption {
	return func(o *SVGOptions) {
		o.ViewBox = viewBox
	}
}

// WithHighlighterOpacity configures the opacity used for highlighter strokes.
func WithHighlighterOpacity(opacity float64) SVGOption {
	return func(o *SVGOptions) {
		o.HighlighterOpacity = opacity
	}
}

// WithHighlighterWidth configures the stroke width used for highlighters.
func WithHighlighterWidth(width float64) SVGOption {
	return func(o *SVGOptions) {
		o.HighlighterWidth = width
	}
}

func defaultSVGOptions() SVGOptions {
	return SVGOptions{
		WidthPt:            DefaultWidthPt,
		HeightPt:           DefaultHeightPt,
		HighlighterOpacity: DefaultHighlighterOpacity,
		HighlighterWidth:   DefaultHighlighterWidth,
	}
}

type highlighterEntry struct {
	Line  *Line
	Color Color
}

// RenderToSVG parses the provided blocks and exports a layered SVG string.
//
// Highlighters are rendered in painter's order underneath pen strokes with
// true Paper Pro RGB colors, stroke-opacity="0.45", and square linecaps.
// Pen strokes are rendered on top with their individual segment dynamic widths and colors.
func RenderToSVG(blocks []Block, opts ...SVGOption) string {
	config := defaultSVGOptions()
	for _, opt := range opts {
		opt(&config)
	}

	halfW := config.WidthPt / 2.0
	viewBox := config.ViewBox
	if viewBox == "" {
		viewBox = fmt.Sprintf("-%.3f 0.0 %.3f %.3f", halfW, config.WidthPt, config.HeightPt)
	}

	if len(blocks) == 0 {
		return fmt.Sprintf("<svg xmlns=\"http://www.w3.org/2000/svg\" height=\"%.3f\" width=\"%.3f\" viewBox=\"%s\"></svg>",
			config.HeightPt, config.WidthPt, viewBox)
	}

	var highlighters []highlighterEntry
	var penStrokes []*Line

	for _, b := range blocks {
		if lineBlock, ok := b.(*SceneLineItemBlock); ok && lineBlock.Item.Value != nil {
			line := lineBlock.Item.Value
			if line.Tool.IsHighlighter() {
				colorRGB := ExtractHighlighterColor(lineBlock.ExtraValueData)
				highlighters = append(highlighters, highlighterEntry{
					Line:  line,
					Color: colorRGB,
				})
			} else {
				penStrokes = append(penStrokes, line)
			}
		}
	}

	var out strings.Builder
	out.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n")
	out.WriteString(fmt.Sprintf("<svg xmlns=\"http://www.w3.org/2000/svg\" height=\"%.3f\" width=\"%.3f\" viewBox=\"%s\">\n",
		config.HeightPt, config.WidthPt, viewBox))

	// 1. Render Highlighters first (layer underneath ink in painter's order)
	out.WriteString("\t<g id=\"highlighters\">\n")
	for _, hl := range highlighters {
		if len(hl.Line.Points) == 0 {
			continue
		}
		var pts strings.Builder
		for i, p := range hl.Line.Points {
			if i > 0 {
				pts.WriteByte(' ')
			}
			pts.WriteString(fmt.Sprintf("%.3f,%.3f", Scale(float64(p.X)), Scale(float64(p.Y))))
		}
		opacity := config.HighlighterOpacity
		if hl.Color.Alpha > 0 {
			opacity = hl.Color.Alpha
		}
		strokeWidth := config.HighlighterWidth
		if hl.Line.Tool == PenToolShader {
			strokeWidth = 16.0
		}

		out.WriteString(fmt.Sprintf("\t\t<polyline fill=\"none\" stroke=\"rgb(%d,%d,%d)\" stroke-opacity=\"%.2f\" stroke-width=\"%.1f\" stroke-linecap=\"square\" stroke-linejoin=\"round\" points=\"%s\" />\n",
			hl.Color.R, hl.Color.G, hl.Color.B, opacity, strokeWidth, pts.String()))
	}
	out.WriteString("\t</g>\n")

	// 2. Render Pen strokes on top
	out.WriteString("\t<g id=\"pen_strokes\">\n")
	for _, line := range penStrokes {
		if len(line.Points) == 0 {
			continue
		}
		pen := CreatePen(line.Tool, line.Color, line.ThicknessScale)
		lastX, lastY := -1.0, -1.0
		lastW := 0.0

		for idx, p := range line.Points {
			segLen := pen.SegmentLength()
			if segLen <= 0 {
				segLen = 1000
			}

			if idx%segLen == 0 {
				if lastX != -1.0 {
					out.WriteString("\" />\n")
				}
				col := pen.SegmentColor(float64(p.Speed), float64(p.Direction), float64(p.Width), float64(p.Pressure), lastW)
				w := pen.SegmentWidth(float64(p.Speed), float64(p.Direction), float64(p.Width), float64(p.Pressure), lastW)
				op := pen.SegmentOpacity(float64(p.Speed), float64(p.Direction), float64(p.Width), float64(p.Pressure), lastW)

				out.WriteString(fmt.Sprintf("\t\t<polyline fill=\"none\" stroke=\"%s\" stroke-opacity=\"%.3f\" stroke-width=\"%.3f\" stroke-linecap=\"%s\" stroke-linejoin=\"round\" points=\"",
					col, op, Scale(w), pen.StrokeLinecap()))
				if lastX != -1.0 {
					out.WriteString(fmt.Sprintf("%.3f,%.3f ", Scale(lastX), Scale(lastY)))
				}
				lastW = w
			}
			lastX, lastY = float64(p.X), float64(p.Y)
			out.WriteString(fmt.Sprintf("%.3f,%.3f ", Scale(float64(p.X)), Scale(float64(p.Y))))
		}
		out.WriteString("\" />\n")
	}
	out.WriteString("\t</g>\n")
	out.WriteString("</svg>\n")

	return out.String()
}
