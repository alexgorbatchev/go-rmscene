# AGENTS.md

Instructions, architectural rules, and test verification standards for agents working on `go-rmscene`.

---

## Repository Purpose

`go-rmscene` (`github.com/alexgorbatchev/go-rmscene`) is a high-performance Go library for parsing, constructing, and rendering reMarkable tablet version 6 binary scene files (`.rm` lines format). It provides:
1. Complete streaming parser and writer for v6 tagged blocks, CRDT items, and LWW structures.
2. Extraction of 24-bit BGRA highlighter colors used by the reMarkable Paper Pro canvas.
3. Physics-based stroke dynamics modeling across 11 stylus tools.
4. Layered vector SVG export implementing strict painter's order (highlighters underneath pen ink).

---

## Architectural Rules & Invariants

### 1. Zero External Dependencies
- The library MUST depend strictly on the Go standard library (`bytes`, `encoding/binary`, `errors`, `fmt`, `io`, `math`, `strings`).
- Do NOT add third-party dependencies or external packages to `go.mod`.

### 2. SVG Painter's Order Invariant
- In vector SVG rendering (`RenderToSVG`), highlighter strokes MUST ALWAYS be rendered before pen ink strokes:
  - Group 1: `<g id="highlighters">` (rendered first, layered at the bottom)
  - Group 2: `<g id="pen_strokes">` (rendered second, layered on top)
- Highlighters use `stroke-opacity="0.45"` (configurable via `WithHighlighterOpacity`), `stroke-linecap="square"`, and `stroke-linejoin="round"`.
- Pen strokes use dynamic segments, round linecaps, and pressure/tilt curves.
- Breaking painter's order violates document legibility and causes highlighter ribbons to obscure text.

### 3. Tag Probing & Stream Rewind
- `DataStream.ReadTag(index, tagType)` MUST rewind stream position to `initialPos` when an index or tag type mismatch occurs.
- Optional block attributes and variable metadata rely on this rewind behavior to probe for optional fields.

### 4. Paper Pro Color Encoding
- Paper Pro highlighter colors are stored in little-endian BGRA format in `SceneLineItemBlock.ExtraValueData`.
- The color tag is index 8 with type `TagByte4` (0x04), serialized in LEB128 as `[0x84, 0x01]`.
- Extraction logic in `color.go` must locate `[0x84, 0x01]`, read Blue (`idx+2`), Green (`idx+3`), Red (`idx+4`), and return `Color{R, G, B}`, falling back to standard highlighter yellow (`DefaultHighlighterColor`: 255, 235, 59) if missing or malformed.

### 5. Forward Compatibility & Unknown Blocks
- Top-level blocks and subblocks not yet explicitly mapped or containing additional trailing fields must retain trailing bytes in `extraData` or `ExtraValueData`.
- The parser must not fail fatally on unknown block types or unmapped tags unless the stream itself is truncated or malformed.

### 6. Coordinate Scaling & DPI
- Tablet digitizer coordinates are in hardware screen units (226 DPI).
- Vector SVG rendering scales dimensions by `ScaleFactor = 72.0 / ScreenDPI` (`72.0 / 226.0` pt per unit).
- Default A5 portrait dimensions: `WidthPt = 447.874`, `HeightPt = 595.275`.

---

## Coding Standards & Style

- **Error Wrapping**: Wrap binary parsing errors using `%w` and appropriate sentinel errors (`ErrInvalidHeader`, `ErrBlockOverflow`, `ErrUnexpectedEOF`).
- **Idiomatic Functional Options**: Exporter configurations use functional options (`WithDimensions`, `WithViewBox`, `WithHighlighterOpacity`, `WithHighlighterWidth`).
- **Memory Efficiency**: Avoid unnecessary byte allocations in tight point iteration and stroke decoding loops.

---

## Testing & Verification Standards

### 1. Test Execution
All tests are run via `just`:
```sh
just test
# Or directly:
go test -v ./...
```

### 2. Red/Green Development Discipline
- For every bug fix or new feature, write a failing unit test first.
- Run `just test` to confirm test failure, apply the fix/feature, and re-run `just test` to verify green status.
- Never write tests that only verify static constant values; verify dynamic parser behavior, roundtrips, and exporter contracts.

### 3. Required Test Coverage Areas
When modifying any parser or exporter code, ensure the following test suites remain green:
- `header_test.go`: 43-byte magic header validation, length checks, invalid magic strings.
- `stream_test.go`: Varuint roundtrips, tag matching/rewinding, CRDT IDs, primitives, LWW values.
- `blocks_test.go`: Version 1 and Version 2 point decoding, tree node blocks, unknown blocks.
- `color_test.go`: Paper Pro BGRA extraction, offsets, fallback handling, color conversions.
- `tools_test.go`: Tool instantiation, ballpoint/pencil dynamics, speed/pressure damping.
- `svg_test.go`: Painter's order `<g id="highlighters">` vs `<g id="pen_strokes">`, Paper Pro colors, custom options.
- `roundtrip_test.go`: End-to-end binary write, parse, line separation, and SVG render.

---

## Git & Release Conventions

- Keep the working directory clean. Build artifacts (`*.test`, `coverage.out`) and OS metadata (`.DS_Store`) must remain ignored by `.gitignore`.
- Ensure all commits have clear, descriptive commit messages describing the changes made.
