package output

import (
	"encoding/json"
	"io"

	"github.com/codeflash-ai/gotrace/pkg/tracer"
)

type jsonFrame struct {
	Name       string       `json:"name"`
	DurationMs float64      `json:"duration_ms"`
	Children   []*jsonFrame `json:"children,omitempty"`
}

func RenderJSON(w io.Writer, roots []*tracer.Frame) error {
	jframes := make([]*jsonFrame, len(roots))
	for i, root := range roots {
		jframes[i] = toJSON(root)
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(jframes)
}

func toJSON(frame *tracer.Frame) *jsonFrame {
	jf := &jsonFrame{
		Name:       frame.Name,
		DurationMs: float64(frame.Duration.Microseconds()) / 1000.0,
	}
	if len(frame.Children) > 0 {
		jf.Children = make([]*jsonFrame, len(frame.Children))
		for i, child := range frame.Children {
			jf.Children[i] = toJSON(child)
		}
	}
	return jf
}
