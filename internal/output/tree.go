package output

import (
	"fmt"
	"io"
	"time"

	"github.com/codeflash-ai/gotrace/pkg/tracer"
)

func shortName(name string) string {
	parts := splitPkgPath(name)
	if len(parts) <= 2 {
		return name
	}
	return parts[len(parts)-2] + "." + parts[len(parts)-1]
}

func splitPkgPath(name string) []string {
	var parts []string
	for name != "" {
		i := 0
		for i < len(name) && name[i] != '.' && name[i] != '/' {
			i++
		}
		if i > 0 {
			parts = append(parts, name[:i])
		}
		if i < len(name) {
			name = name[i+1:]
		} else {
			break
		}
	}
	return parts
}

func RenderTree(w io.Writer, roots []*tracer.Frame) error {
	var totalDuration time.Duration
	for _, root := range roots {
		if root.Duration > totalDuration {
			totalDuration = root.Duration
		}
	}

	if totalDuration == 0 {
		for _, root := range roots {
			computeDuration(root)
			if root.Duration > totalDuration {
				totalDuration = root.Duration
			}
		}
	}

	fmt.Fprintf(w, "TOTAL: %s\n\n", formatDuration(totalDuration))

	for _, root := range roots {
		renderNode(w, root, totalDuration, "", true, true)
	}
	return nil
}

func renderNode(w io.Writer, frame *tracer.Frame, total time.Duration, prefix string, last bool, isRoot bool) {
	connector := "├── "
	if last {
		connector = "└── "
	}
	if isRoot {
		connector = ""
	}

	pct := float64(0)
	if total > 0 {
		pct = float64(frame.Duration) / float64(total) * 100
	}

	fmt.Fprintf(w, "%s%s%-40s %10s %5.1f%%\n",
		prefix, connector,
		shortName(frame.Name),
		formatDuration(frame.Duration),
		pct,
	)

	var childPrefix string
	if isRoot {
		childPrefix = ""
	} else if last {
		childPrefix = prefix + "    "
	} else {
		childPrefix = prefix + "│   "
	}

	for i, child := range frame.Children {
		renderNode(w, child, total, childPrefix, i == len(frame.Children)-1, false)
	}
}

func computeDuration(frame *tracer.Frame) {
	if frame.Duration > 0 {
		return
	}
	var maxEnd time.Duration
	for _, child := range frame.Children {
		computeDuration(child)
		if child.End > maxEnd {
			maxEnd = child.End
		}
	}
	if maxEnd > frame.Start {
		frame.Duration = maxEnd - frame.Start
		frame.End = maxEnd
	}
}

func formatDuration(d time.Duration) string {
	switch {
	case d >= time.Second:
		return fmt.Sprintf("%.2fs", d.Seconds())
	case d >= time.Millisecond:
		return fmt.Sprintf("%dms", d.Milliseconds())
	case d >= time.Microsecond:
		return fmt.Sprintf("%dus", d.Microseconds())
	default:
		return fmt.Sprintf("%dns", d.Nanoseconds())
	}
}
