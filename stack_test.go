package serror

import (
	"errors"
	"log/slog"
	"strings"
	"testing"
)

func TestCaptureStack(t *testing.T) {
	t.Parallel()

	pcs := captureStack(1)

	if len(pcs) == 0 {
		t.Fatal("captureStack returned no program counters")
	}

	frames := resolveFrames(pcs)
	if len(frames) == 0 {
		t.Fatal("resolveFrames returned no frames")
	}

	f := frames[0]
	if f.File == "" {
		t.Error("first frame file is empty")
	}

	if f.Line == 0 {
		t.Error("first frame line is 0")
	}

	if f.Function == "" {
		t.Error("first frame function is empty")
	}
}

//go:noinline
func newInHelper() error {
	return New("boom")
}

//go:noinline
func wrapStdlibInHelper() error {
	return Wrap(errors.New("inner"), "wrapped")
}

//go:noinline
func wrapSerrorInHelper(err error) error {
	return Wrap(err, "wrapped")
}

func TestNewCapturesCallSite(t *testing.T) {
	t.Parallel()

	err := newInHelper()

	var s *Error
	if !errors.As(err, &s) {
		t.Fatal("As did not match *serror.Error")
	}

	frames := resolveFrames(s.stackPcs)
	if len(frames) == 0 {
		t.Fatal("no frames captured")
	}

	if frames[0].Function != "github.com/ptman/serror.newInHelper" {
		t.Errorf("first frame = %q, want %q", frames[0].Function, "github.com/ptman/serror.newInHelper")
	}
}

func TestWrapCapturesCallSite(t *testing.T) {
	t.Parallel()

	err := wrapStdlibInHelper()

	var s *Error
	if !errors.As(err, &s) {
		t.Fatal("As did not match *serror.Error")
	}

	frames := resolveFrames(s.stackPcs)
	if len(frames) == 0 {
		t.Fatal("no frames captured")
	}

	if frames[0].Function != "github.com/ptman/serror.wrapStdlibInHelper" {
		t.Errorf("first frame = %q, want %q", frames[0].Function, "github.com/ptman/serror.wrapStdlibInHelper")
	}
}

func TestWrapPreservesOriginalCallSite(t *testing.T) {
	t.Parallel()

	original := newInHelper()
	wrapped := wrapSerrorInHelper(original)

	var s *Error
	if !errors.As(wrapped, &s) {
		t.Fatal("As did not match *serror.Error")
	}

	frames := resolveFrames(s.stackPcs)
	if len(frames) == 0 {
		t.Fatal("no frames captured")
	}

	if frames[0].Function != "github.com/ptman/serror.newInHelper" {
		t.Errorf("first frame = %q, want %q", frames[0].Function, "github.com/ptman/serror.newInHelper")
	}
}

func TestCaptureStackSkip(t *testing.T) {
	t.Parallel()

	// Skip enough frames to go past captureStack and this test function.
	pcs := captureStack(2)

	if len(pcs) == 0 {
		t.Error("captureStack with skip=2 returned no program counters")
	}
}

func TestResolveFrames(t *testing.T) {
	t.Parallel()

	pcs := captureStack(1)
	frames := resolveFrames(pcs)

	if len(frames) != len(pcs) {
		t.Errorf("frames length %d, want %d", len(frames), len(pcs))
	}
}

func TestResolveFramesEmpty(t *testing.T) {
	t.Parallel()

	frames := resolveFrames(nil)

	if frames != nil {
		t.Errorf("resolveFrames(nil) = %v, want nil", frames)
	}

	empty := make([]uintptr, 0)
	frames = resolveFrames(empty)

	if frames != nil {
		t.Errorf("resolveFrames(empty) = %v, want nil", frames)
	}
}

func TestFormatStack(t *testing.T) {
	t.Parallel()

	pcs := captureStack(1)
	output := formatStack(pcs)

	if output == "" {
		t.Fatal("formatStack returned empty string")
	}

	if !strings.Contains(output, "\n\t") {
		t.Errorf("formatStack missing tab indentation: %q", output)
	}

	if !strings.Contains(output, ".go:") {
		t.Errorf("formatStack missing file:line: %q", output)
	}
}

func TestFormatStackEmpty(t *testing.T) {
	t.Parallel()

	output := formatStack(nil)

	if output != "" {
		t.Errorf("formatStack(nil) = %q, want empty string", output)
	}
}

func TestCaptureStackEmpty(t *testing.T) {
	t.Parallel()

	// Use a very large skip value to exhaust the call stack.
	pcs := captureStack(1000)

	if pcs != nil {
		t.Errorf("captureStack(1000) pcs = %v, want nil", pcs)
	}
}

func TestStackGroupValuesEmpty(t *testing.T) {
	t.Parallel()

	result := stackGroupValues(nil)

	if result != nil {
		t.Errorf("stackGroupValues(nil) = %v, want nil", result)
	}

	empty := make([]uintptr, 0)
	result = stackGroupValues(empty)

	if result != nil {
		t.Errorf("stackGroupValues(empty) = %v, want nil", result)
	}
}

func TestStackGroupValues(t *testing.T) {
	t.Parallel()

	pcs := captureStack(1)
	result := stackGroupValues(pcs)

	if len(result) != 1 {
		t.Fatalf("stackGroupValues returned %d attrs, want 1", len(result))
	}

	if result[0].Key != "frames" {
		t.Errorf("attr key = %q, want %q", result[0].Key, "frames")
	}
}

func TestAttrToAny(t *testing.T) {
	t.Parallel()

	attrs := []slog.Attr{
		slog.String("k", "v"),
		slog.Int("n", 42),
	}
	result := attrToAny(attrs)

	if len(result) != 2 {
		t.Fatalf("attrToAny returned %d items, want 2", len(result))
	}
}

func TestAttrToAnyEmpty(t *testing.T) {
	t.Parallel()

	result := attrToAny(nil)

	if len(result) != 0 {
		t.Errorf("attrToAny(nil) = %v, want empty slice", result)
	}
}
