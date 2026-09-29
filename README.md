# go-rmscene

[![Go Reference](https://pkg.go.dev/badge/github.com/alexgorbatchev/go-rmscene.svg)](https://pkg.go.dev/github.com/alexgorbatchev/go-rmscene)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

A high-performance, zero-dependency Go library for parsing reMarkable tablet **v6 binary scene files** (`.rm` lines format) and rendering them to layered, publication-quality **SVG documents**.

Designed for reMarkable OS 3.x+ notebooks and documents, including full support for the **reMarkable Paper Pro** color canvas, 24-bit BGRA highlighter colors, and physics-based pen dynamics.

---

## Features

- **Pure Go Standard Library**: Zero external dependencies. Fast binary stream parsing and vector rendering using Go 1.22+ standard library primitives.
- **Complete v6 Block Hierarchy**: Parses top-level blocks including `SceneLineItemBlock` (pen and highlighter strokes), `TreeNodeBlock` (layers), `SceneGroupItemBlock`, `MigrationInfoBlock`, `SceneTreeBlock`, `AuthorIdsBlock`, `PageInfoBlock`, and text blocks.
- **Paper Pro 24-bit Color Support**: Extracts true 24-bit RGB colors from Paper Pro highlighter strokes encoded in little-endian BGRA extra-value data buffers.
- **Physics-Based Pen Dynamics**: Faithfully reproduces stylus stroke geometry, width variation, opacity curves, and color gradients for:
  - Ballpoint Pen (pressure sensitivity and speed dampening)
  - Fineliner
  - Marker (tilt-dependent thickness)
  - Pencil (pressure-sensitive opacity and tilt shading)
  - Mechanical Pencil
  - Paintbrush / Brush (pressure-sensitive width and dynamic color mixing)
  - Calligraphy Pen (tilt and velocity dynamics)
  - Highlighter & Shader
  - Eraser & Area Eraser
- **Painter's Order SVG Layering**: Highlighters are automatically rendered *underneath* ink strokes in `<g id="highlighters">` with semi-transparency and square caps, ensuring handwritten text and drawings remain crisp and legible.
- **Bi-directional Binary Tooling**: Includes both `DataStream` reader and `DataWriter` encoder for reading, inspecting, and programmatically constructing v6 `.rm` binary streams.

---

## Installation

```sh
go get github.com/alexgorbatchev/go-rmscene
```

---

## Usage

### 1. Parsing a `.rm` File

Parse binary scene data directly from a byte slice or any `io.Reader`:

```go
package main

import (
	"fmt"
	"os"

	"github.com/alexgorbatchev/go-rmscene"
)

func main() {
	data, err := os.ReadFile("page.rm")
	if err != nil {
		panic(err)
	}

	// Parse scene from byte buffer (validates 43-byte magic header)
	scene, err := rmscene.Parse(data)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Total blocks: %d\n", len(scene.Blocks))

	// Convenient accessors for pen and highlighter strokes
	penLines := scene.Lines()
	highlighters := scene.Highlighters()

	fmt.Printf("Pen strokes: %d\n", len(penLines))
	fmt.Printf("Highlighter strokes: %d\n", len(highlighters))

	// Inspect stroke coordinates and stylus telemetry
	for i, stroke := range penLines {
		fmt.Printf("Stroke #%d: tool=%s color=%s points=%d\n",
			i, stroke.Tool, stroke.Color, len(stroke.Points))
		for _, pt := range stroke.Points {
			// X, Y in tablet units (226 DPI); speed, direction, width, pressure
			_ = pt.X
			_ = pt.Y
			_ = pt.Pressure
		}
	}
}
```

Or parse from a streaming reader:

```go
file, err := os.Open("page.rm")
if err != nil {
	panic(err)
}
defer file.Close()

scene, err := rmscene.ParseReader(file)
if err != nil {
	panic(err)
}
```

---

### 2. Extracting Paper Pro 24-bit BGRA Highlighter Colors

On reMarkable Paper Pro, highlighters can produce rich 24-bit custom colors (such as Cyan, Magenta, Yellow, Green, Pink). The tablet encodes these colors inside the stroke's `ExtraValueData` using tag index 8 with type `Byte4` (encoded in LEB128 as `[0x84, 0x01]`) followed by little-endian BGRA bytes.

`go-rmscene` provides dedicated helpers to extract these colors:

```go
package main

import (
	"fmt"

	"github.com/alexgorbatchev/go-rmscene"
)

func inspectHighlighterColors(scene *rmscene.Scene) {
	for _, block := range scene.Blocks {
		lineBlock, ok := block.(*rmscene.SceneLineItemBlock)
		if !ok || lineBlock.Item.Value == nil {
			continue
		}

		line := lineBlock.Item.Value
		if !line.Tool.IsHighlighter() {
			continue
		}

		// Extract 24-bit RGB color from Paper Pro extra_value_data
		// Falls back to standard yellow (255, 235, 59) if absent or malformed
		color := rmscene.ExtractHighlighterColor(lineBlock.ExtraValueData)

		fmt.Printf("Highlighter Stroke (Item ID: %v):\n", lineBlock.Item.ItemID)
		fmt.Printf("  RGB:   %s\n", color.String()) // "rgb(139,208,229)"
		fmt.Printf("  Hex:   %s\n", color.Hex())    // "#8bd0e5"
		fmt.Printf("  R=%d, G=%d, B=%d\n", color.R, color.G, color.B)
	}
}
```

You can also specify a custom fallback color:

```go
customFallback := rmscene.Color{R: 251, G: 247, B: 25}
color := rmscene.ExtractHighlighterColorWithFallback(lineBlock.ExtraValueData, customFallback)
```

---

### 3. Rendering to Layered SVG

Render a parsed scene or raw `.rm` bytes into an SVG string. In painter's order, highlighters are layered *first* in `<g id="highlighters">` underneath the ink, while pen strokes are rendered on top in `<g id="pen_strokes">`:

```go
package main

import (
	"os"

	"github.com/alexgorbatchev/go-rmscene"
)

func main() {
	rmBytes, err := os.ReadFile("drawing.rm")
	if err != nil {
		panic(err)
	}

	// 1. One-shot rendering from raw .rm bytes
	svg, err := rmscene.RenderRMToSVG(rmBytes)
	if err != nil {
		panic(err)
	}
	os.WriteFile("output.svg", []byte(svg), 0644)

	// 2. Or render from a parsed Scene with custom options
	scene, err := rmscene.Parse(rmBytes)
	if err != nil {
		panic(err)
	}

	customSVG := scene.ToSVG(
		// Standard A5 portrait dimensions in points (default: 447.874 x 595.275)
		rmscene.WithDimensions(447.874, 595.275),
		// Custom viewport viewBox
		rmscene.WithViewBox("-223.937 0.0 447.874 595.275"),
		// Custom highlighter opacity (default: 0.45)
		rmscene.WithHighlighterOpacity(0.50),
		// Custom highlighter stroke width (default: 10.0)
		rmscene.WithHighlighterWidth(12.0),
	)
	os.WriteFile("custom.svg", []byte(customSVG), 0644)
}
```

#### Why Painter's Order Matters

When writing or drawing on physical paper, highlighting over ink or writing over a highlight maintains legibility. However, when rendering vector paths to an SVG canvas:
- If highlighters were drawn in chronological order after pen strokes, semi-transparent highlighter ribbons would cover dark ballpoint, fineliner, or pencil lines, muddying text and lineart.
- By placing all highlighter polylines into an initial `<g id="highlighters">` group with `stroke-opacity="0.45"` and `stroke-linecap="square"`, and ink strokes in a subsequent `<g id="pen_strokes">` group, pen lines remain sharply rendered on top.

---

### 4. Constructing `.rm` Files Programmatically

`go-rmscene` includes a full binary `DataWriter` supporting the v6 format:

```go
package main

import (
	"bytes"
	"os"

	"github.com/alexgorbatchev/go-rmscene"
)

func main() {
	var buf bytes.Buffer
	w := rmscene.NewDataWriter(&buf)

	// Write 43-byte magic header
	if err := w.WriteHeader(); err != nil {
		panic(err)
	}

	// Add a ballpoint pen stroke
	block := &rmscene.SceneLineItemBlock{
		ParentID: rmscene.CrdtId{Part1: 1, Part2: 1},
		Item: rmscene.Item[*rmscene.Line]{
			ItemID: rmscene.CrdtId{Part1: 1, Part2: 10},
			Value: &rmscene.Line{
				Tool:           rmscene.PenToolBallpoint1,
				Color:          rmscene.PenColorBlack,
				ThicknessScale: 1.0,
				Points: []rmscene.Point{
					{X: 100, Y: 100, Speed: 100, Direction: 45, Width: 10, Pressure: 180},
					{X: 110, Y: 115, Speed: 110, Direction: 48, Width: 10, Pressure: 190},
					{X: 125, Y: 130, Speed: 120, Direction: 50, Width: 10, Pressure: 200},
				},
			},
		},
	}

	if err := w.WriteSceneLineItemBlock(block, 2); err != nil {
		panic(err)
	}

	os.WriteFile("generated.rm", buf.Bytes(), 0644)
}
```

---

## Technical Specifications

| Property | Value |
|---|---|
| Supported Format | reMarkable lines format version 6 (`reMarkable .lines file, version=6          `) |
| Supported OS | reMarkable OS 3.x and above |
| Hardware Targets | reMarkable 1, reMarkable 2, reMarkable Paper Pro |
| Native Resolution | 226 DPI screen coordinates |
| SVG Coordinate Scale | 72 / 226 (~0.318584 pt per tablet screen unit) |
| Standard Dimensions | 447.874 pt x 595.275 pt (A5 portrait) |
| Color Palettes | 14-color standard palette + 24-bit Paper Pro BGRA extra value data |
| Pen Tool Models | 11 distinct tool models with pressure, speed, tilt, and opacity simulation |

---

## Upstream Attribution & Acknowledgments

This library is built upon the foundational research and reverse engineering of the reMarkable lines binary format conducted by the open-source community:

- **Rick Lupton** ([@ricklupton](https://github.com/ricklupton)):
  Author of [ricklupton/rmscene](https://github.com/ricklupton/rmscene) and [ricklupton/rmc](https://github.com/ricklupton/rmc). Rick's meticulous reverse engineering and reference Python implementation of the v6 CRDT block structure, varuint encoding, and stroke models made this Go implementation possible.
- **ddvk** ([@ddvk](https://github.com/ddvk)):
  Author of [ddvk/reader](https://github.com/ddvk/reader) and countless seminal tools for the reMarkable platform. ddvk's pioneering work deciphering earlier lines formats, pen telemetry curves, and tablet internals paved the way for modern format reverse engineering.
- **The reHackable Community** ([github.com/reHackable](https://github.com/reHackable)):
  The collective open-source wiki, format specifications, and reverse-engineering community that document and maintain the reMarkable hardware and software ecosystem.

---

## License

MIT License &mdash; see [LICENSE](LICENSE) for details.

Copyright (c) 2026 Alex Gorbatchev and upstream contributors.
