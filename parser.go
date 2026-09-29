package rmscene

import (
	"bytes"
	"errors"
	"io"
)

// Scene represents a parsed reMarkable .rm v6 scene file.
type Scene struct {
	Blocks []Block
}

// Lines returns all pen strokes in the scene (excluding highlighters).
func (s *Scene) Lines() []*Line {
	var lines []*Line
	for _, b := range s.Blocks {
		if lb, ok := b.(*SceneLineItemBlock); ok && lb.Item.Value != nil {
			if !lb.Item.Value.Tool.IsHighlighter() {
				lines = append(lines, lb.Item.Value)
			}
		}
	}
	return lines
}

// Highlighters returns all highlighter strokes in the scene.
func (s *Scene) Highlighters() []*Line {
	var hl []*Line
	for _, b := range s.Blocks {
		if lb, ok := b.(*SceneLineItemBlock); ok && lb.Item.Value != nil {
			if lb.Item.Value.Tool.IsHighlighter() {
				hl = append(hl, lb.Item.Value)
			}
		}
	}
	return hl
}

// ToSVG exports the scene to an SVG string.
func (s *Scene) ToSVG(opts ...SVGOption) string {
	return RenderToSVG(s.Blocks, opts...)
}

// Parse parses a reMarkable v6 .rm binary file from bytes.
func Parse(data []byte) (*Scene, error) {
	if err := ValidateHeaderBytes(data); err != nil {
		return nil, err
	}
	r := bytes.NewReader(data[HeaderLength:])
	return parseStream(NewDataStream(r))
}

// ParseReader parses a reMarkable v6 .rm binary stream from an io.Reader.
func ParseReader(r io.Reader) (*Scene, error) {
	if err := ValidateHeader(r); err != nil {
		return nil, err
	}
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return parseStream(NewDataStream(bytes.NewReader(data)))
}

func parseStream(s *DataStream) (*Scene, error) {
	var blocks []Block
	for {
		block, err := ReadBlock(s)
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}
		blocks = append(blocks, block)
	}
	return &Scene{Blocks: blocks}, nil
}

// RenderRMToSVG parses raw .rm file bytes and exports an SVG string directly.
func RenderRMToSVG(rmBytes []byte, opts ...SVGOption) (string, error) {
	if len(rmBytes) == 0 {
		return RenderToSVG(nil, opts...), nil
	}
	scene, err := Parse(rmBytes)
	if err != nil {
		return "", err
	}
	return scene.ToSVG(opts...), nil
}
