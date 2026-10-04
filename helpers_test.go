package serror

import (
	"runtime"
	"strings"
)

// Frame represents a single entry in a stack trace.
type Frame struct {
	File     string
	Line     int
	Function string
}

// stringState adapts strings.Builder to fmt.State so that functions which
// write directly to fmt.State can still produce a plain string.
type stringState struct {
	b strings.Builder
}

func (st *stringState) Write(p []byte) (int, error) {
	//nolint:wrapcheck // strings.Builder.Write never returns an error; this is a fmt.State adapter.
	return st.b.Write(p)
}

func (st *stringState) Width() (int, bool)     { return 0, false }
func (st *stringState) Precision() (int, bool) { return 0, false }
func (st *stringState) Flag(int) bool          { return false }

// resolveFrames converts raw program counters into human-readable frames.
func resolveFrames(pcs []uintptr) []Frame {
	if len(pcs) == 0 {
		return nil
	}

	iter := runtime.CallersFrames(pcs)
	frames := make([]Frame, 0, len(pcs))

	for {
		frame, more := iter.Next()
		frames = append(frames, Frame{
			File:     frame.File,
			Line:     frame.Line,
			Function: frame.Function,
		})

		if !more {
			break
		}
	}

	return frames
}

func formatMap(m map[string]any) string {
	var st stringState
	writeMap(&st, m)

	return st.b.String()
}

func formatFrames(pcs []uintptr) string {
	var st stringState
	writeFrames(&st, pcs)

	return st.b.String()
}

func formatStack(pcs []uintptr) string {
	var st stringState
	writeStack(&st, pcs)

	return st.b.String()
}
