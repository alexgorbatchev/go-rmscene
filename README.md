`go-rmscene` is a high-performance, zero-dependency Go library for parsing reMarkable tablet v6 binary scene files (`.rm` lines format) and rendering them to layered, publication-quality vector SVG documents.

# What It Does

- **Binary stroke decoding**: Parses reMarkable v6 binary `.rm` files into structured block hierarchies and typed stroke models.
- **Paper Pro color decoding**: Decodes 24-bit RGB colors from Paper Pro highlighter and shader strokes encoded in little-endian BGRA buffers.
- **Physics-based stylus dynamics**: Reproduces pressure curves, tilt dampening, velocity variations, and opacity gradients across 11 writing tools.
- **Layer-ordered SVG rendering**: Groups highlighter and shader washes underneath pen ink in painter's order to maintain sharp, legible handwriting.
- **Bi-directional binary tooling**: Includes `DataWriter` encoder and `DataStream` reader for inspecting and generating valid v6 binary streams.

# How It Works

- Validates the 43-byte magic file header (`reMarkable .lines file, version=6`).
- Iterates through variable-length binary blocks, decoding tagged CRDT identifiers and telemetry points.
- Builds an in-memory scene tree separating vector strokes, text items, migration info, and layer metadata.
- Generates layered vector SVGs using stylus physics models and Paper Pro color palettes.

# How it Really Works

- The parser operates directly on byte slices or seekable streams using pure Go standard library primitives, allocating zero external buffers or C bindings.
- In painter's order SVG output, highlighters are placed into `<g id="highlighters">` with semi-transparency and square linecaps before `<g id="pen_strokes">`, ensuring highlighter washes never obscure black handwriting.
- Tool 23 (`SHADER`) is dynamically classified as a variable-opacity wash rather than an opaque yellow stroke, mapping raw alpha bytes (e.g. `77` -> 30%, `115` -> 45%) to CSS stroke opacity.
- Subblocks are protected against buffer underflows and overflows; any stream exceeding its declared length fails immediately with `ErrBlockOverflow`.

# Prerequisites

- [Go](https://go.dev/) 1.26 or newer.

# Installation

```bash
go get github.com/alexgorbatchev/go-rmscene
```

# Quick Start

```go
package main

import (
	"fmt"
	"os"

	"github.com/alexgorbatchev/go-rmscene"
)

func main() {
	rmBytes, err := os.ReadFile("page.rm")
	if err != nil {
		panic(err)
	}

	// Parse binary strokes into Scene structure
	scene, err := rmscene.Parse(rmBytes)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Parsed %d blocks, %d lines\n", len(scene.Blocks), len(scene.Lines()))

	// Render directly to SVG string
	svg, err := rmscene.RenderRMToSVG(rmBytes, rmscene.WithDimensions(447.87, 608.20))
	if err != nil {
		panic(err)
	}

	_ = os.WriteFile("page.svg", []byte(svg), 0644)
}
```

# API

| Export | Signature | Description |
| :--- | :--- | :--- |
| `Parse` | `(data []byte) (*Scene, error)` | Parses v6 binary scene file from a byte slice |
| `ParseReader` | `(r io.Reader) (*Scene, error)` | Parses v6 binary scene stream from an `io.Reader` |
| `RenderRMToSVG` | `(rmBytes []byte, opts ...SVGOption) (string, error)` | Parses and converts `.rm` bytes directly to SVG |
| `RenderToSVG` | `(blocks []Block, opts ...SVGOption) string` | Renders parsed blocks into a layered SVG string |
| `ValidateHeader` | `(r io.Reader) error` | Validates the 43-byte magic header in a stream |
| `NewDataWriter` | `(w io.Writer) *DataWriter` | Creates a binary serializer for v6 `.rm` streams |
| `NewDataStream` | `(r io.ReadSeeker) *DataStream` | Creates a low-level binary tag stream decoder |

# Configuration

| Option | Type | Default | Description |
| :--- | :--- | :--- | :--- |
| `WithDimensions` | `(widthPt, heightPt float64)` | `447.87, 608.20` | Output SVG canvas width and height in points |
| `WithViewBox` | `(viewBox string)` | centered | Custom SVG viewBox string |
| `WithHighlighterOpacity` | `(opacity float64)` | `0.45` | Default opacity for standard highlighter strokes |
| `WithHighlighterWidth` | `(width float64)` | `10.0` | Default stroke width for highlighters |

# License

MIT License (c) 2026 Alex Gorbatchev
