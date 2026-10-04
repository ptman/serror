package serror

import (
	"fmt"
	"runtime"
)

// captureStack captures the current call stack, skipping the given number of
// initial frames. Returns raw program counters.
func captureStack(skip int) []uintptr {
	//nolint:mnd // 32 is a reasonable default stack depth for most call chains
	pcs := make([]uintptr, 32)
	//nolint:mnd // 2 skips captureStack itself and its caller (New or Wrap)
	n := runtime.Callers(skip+2, pcs)
	if n == 0 {
		return nil
	}

	pcsCopy := make([]uintptr, n)
	copy(pcsCopy, pcs[:n])

	return pcsCopy
}

// writeStack writes the formatted stack trace directly to s without
// allocating an intermediate string.
func writeStack(s fmt.State, pcs []uintptr) {
	if len(pcs) == 0 {
		return
	}

	iter := runtime.CallersFrames(pcs)

	for {
		frame, more := iter.Next()

		writef(s, "%s\n\t%s:%d\n", frame.Function, frame.File, frame.Line)

		if !more {
			break
		}
	}
}
