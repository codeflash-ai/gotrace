package output

import (
	"fmt"
	"io"
	"strings"

	"github.com/codeflash-ai/gotrace/pkg/tracer"
)

func RenderCollapsedStacks(w io.Writer, roots []*tracer.Frame) error {
	for _, root := range roots {
		emitStacks(w, root, nil)
	}
	return nil
}

func emitStacks(w io.Writer, frame *tracer.Frame, ancestors []string) {
	stack := append(ancestors, frame.Name)

	if len(frame.Children) == 0 {
		fmt.Fprintf(w, "%s %d\n", strings.Join(stack, ";"), frame.Duration.Microseconds())
		return
	}

	var childDuration int64
	for _, child := range frame.Children {
		emitStacks(w, child, stack)
		childDuration += child.Duration.Microseconds()
	}

	selfTime := frame.Duration.Microseconds() - childDuration
	if selfTime > 0 {
		fmt.Fprintf(w, "%s %d\n", strings.Join(stack, ";"), selfTime)
	}
}
