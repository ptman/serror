package serror

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"
)

const ErrCustom Sentinel = "custom error"

func TestNew(t *testing.T) {
	t.Parallel()

	err := New("something went wrong")
	if err == nil {
		t.Fatal("New returned nil")
	}

	if err.Error() != "something went wrong" {
		t.Errorf("Error() = %q, want %q", err.Error(), "something went wrong")
	}

	var s *Error
	if !errors.As(err, &s) {
		t.Fatal("As did not match *serror.Error")
	}

	if len(s.StackTrace()) == 0 {
		t.Error("New did not capture stack")
	}
}

func TestNilError(t *testing.T) {
	t.Parallel()

	err := New("")
	if err == nil {
		t.Fatal("New with empty string returned nil")
	}

	if err.Error() != "" {
		t.Errorf("Error() = %q, want empty string", err.Error())
	}
}

func TestWrapNil(t *testing.T) {
	t.Parallel()

	if Wrap(nil, "msg") != nil {
		t.Fatal("Wrap(nil, ...) should return nil")
	}

	if Wrap(nil, "", "k", "v") != nil {
		t.Fatal("Wrap(nil, ..., pairs) should return nil")
	}
}

func TestWrapStdlibError(t *testing.T) {
	t.Parallel()

	cause := errors.New("original")
	err := Wrap(cause, "wrapped", "key", "value")

	if err.Error() != "wrapped" {
		t.Errorf("Error() = %q, want %q", err.Error(), "wrapped")
	}

	var s *Error
	if !errors.As(err, &s) {
		t.Fatal("As did not match *serror.Error")
	}

	if len(s.StackTrace()) == 0 {
		t.Error("Wrap of stdlib error did not capture stack")
	}

	if s.cause != cause {
		t.Error("Wrap did not preserve cause")
	}

	if s.attrs["key"] != "value" {
		t.Errorf("attrs[\"key\"] = %v, want %v", s.attrs["key"], "value")
	}
}

func TestWrapSerrorError(t *testing.T) {
	t.Parallel()

	cause := New("original")
	err := Wrap(cause, "wrapped", "key", "value")

	if err.Error() != "wrapped" {
		t.Errorf("Error() = %q, want %q", err.Error(), "wrapped")
	}

	// Wrap must return a distinct *Error instance.
	if err == cause {
		t.Error("Wrap(serror.Error, ...) should return a new instance")
	}

	var s *Error
	if !errors.As(err, &s) {
		t.Fatal("As did not match *serror.Error")
	}

	if s.cause != cause {
		t.Error("Wrap should chain the original serror.Error as cause")
	}

	if s.attrs["key"] != "value" {
		t.Errorf("attrs[\"key\"] = %v, want %v", s.attrs["key"], "value")
	}

	// Original should be unchanged.
	if cause.Error() != "original" {
		t.Errorf("original error was mutated: %q", cause.Error())
	}
}

func TestWrapPreservesStack(t *testing.T) {
	t.Parallel()

	original := New("original")
	wrapped := Wrap(original, "wrapped")

	var origS, wrapS *Error
	errors.As(original, &origS)
	errors.As(wrapped, &wrapS)

	origST := origS.StackTrace()
	wrapST := wrapS.StackTrace()

	if len(wrapST) != len(origST) {
		t.Errorf("stack length %d, want %d", len(wrapST), len(origST))
	}

	for i := range origST {
		if wrapST[i] != origST[i] {
			t.Fatalf("stack pc %d differs", i)
		}
	}
}

func TestWrapSerrorErrorMergesAttrs(t *testing.T) {
	t.Parallel()

	cause := New("original", "original_key", "original_value").(*Error)
	err := Wrap(cause, "wrapped", "new_key", "new_value").(*Error)

	if err.attrs["original_key"] != "original_value" {
		t.Errorf("original attr lost: %v", err.attrs["original_key"])
	}

	if err.attrs["new_key"] != "new_value" {
		t.Errorf("new attr missing: %v", err.attrs["new_key"])
	}

	// New attrs should override original attrs with the same key.
	override := Wrap(cause, "wrapped", "original_key", "overridden").(*Error)
	if override.attrs["original_key"] != "overridden" {
		t.Errorf("attr override failed: %v", override.attrs["original_key"])
	}

	// Original attrs should be unchanged.
	if cause.attrs["original_key"] != "original_value" {
		t.Errorf("original error attrs were mutated: %v", cause.attrs["original_key"])
	}
}

func TestWithAttrs(t *testing.T) {
	t.Parallel()

	original := New("msg")

	withOne := original.(*Error).WithAttrs("k1", "v1")
	if len(withOne.(*Error).attrs) != 1 {
		t.Errorf("attrs length = %d, want 1", len(withOne.(*Error).attrs))
	}

	if withOne.(*Error).attrs["k1"] != "v1" {
		t.Errorf("attrs[\"k1\"] = %v, want %v", withOne.(*Error).attrs["k1"], "v1")
	}

	// Original should be unchanged.
	if len(original.(*Error).attrs) != 0 {
		t.Error("WithAttrs mutated original")
	}

	withMore := withOne.(*Error).WithAttrs("k2", 42, "k3", true)
	if len(withMore.(*Error).attrs) != 3 {
		t.Errorf("attrs length = %d, want 3", len(withMore.(*Error).attrs))
	}

	if withMore.(*Error).attrs["k2"] != 42 {
		t.Errorf("attrs[\"k2\"] = %v, want %v", withMore.(*Error).attrs["k2"], 42)
	}

	if withMore.(*Error).attrs["k3"] != true {
		t.Errorf("attrs[\"k3\"] = %v, want %v", withMore.(*Error).attrs["k3"], true)
	}
}

func TestNewWithAttrs(t *testing.T) {
	t.Parallel()

	err := New("msg", "k1", "v1", "k2", 42).(*Error)
	if err.Error() != "msg" {
		t.Errorf("Error() = %q, want %q", err.Error(), "msg")
	}

	if len(err.attrs) != 2 {
		t.Errorf("attrs length = %d, want 2", len(err.attrs))
	}

	if err.attrs["k1"] != "v1" {
		t.Errorf("attrs[\"k1\"] = %v, want %v", err.attrs["k1"], "v1")
	}

	if err.attrs["k2"] != 42 {
		t.Errorf("attrs[\"k2\"] = %v, want %v", err.attrs["k2"], 42)
	}
}

func TestAttrs(t *testing.T) {
	t.Parallel()

	err := New("msg").(*Error).WithAttrs("k", "v").(*Error)

	attrs := err.Attrs()
	if attrs["k"] != "v" {
		t.Errorf("Attrs()[\"k\"] = %v, want %v", attrs["k"], "v")
	}

	// Modifying the returned map should not affect the error.
	attrs["k"] = "modified"

	if err.Attrs()["k"] != "v" {
		t.Error("Attrs returned a mutable map")
	}
}

func TestErrorReturnsMessageOnly(t *testing.T) {
	t.Parallel()

	err := New("message", "k", "v").(*Error)
	if strings.Contains(err.Error(), "{") {
		t.Errorf("Error() = %q should not contain attrs", err.Error())
	}

	if strings.Contains(err.Error(), "\n") {
		t.Errorf("Error() = %q should not contain stack", err.Error())
	}
}

func TestUnwrap(t *testing.T) {
	t.Parallel()

	cause := errors.New("original")
	err := Wrap(cause, "wrapped")

	unwrapped := errors.Unwrap(err)
	if unwrapped != cause {
		t.Error("Unwrap did not return the cause")
	}
}

func TestIs(t *testing.T) {
	t.Parallel()

	cause := errors.New("original")
	err := Wrap(cause, "wrapped")

	if !errors.Is(err, cause) {
		t.Error("errors.Is(err, cause) should be true")
	}
}

func TestIsSentinel(t *testing.T) {
	t.Parallel()

	err := Wrap(ErrCustom, "wrapped")

	if !errors.Is(err, ErrCustom) {
		t.Error("errors.Is(err, sentinel) should be true")
	}
}

func TestAs(t *testing.T) {
	t.Parallel()

	err := New("msg")

	var s *Error
	if !errors.As(err, &s) {
		t.Fatal("As did not match *serror.Error")
	}

	if s.Error() != "msg" {
		t.Errorf("As matched wrong error: %q", s.Error())
	}
}

func TestAsNested(t *testing.T) {
	t.Parallel()

	cause := errors.New("inner")
	err := Wrap(cause, "outer")

	var s *Error
	if !errors.As(err, &s) {
		t.Fatal("As did not match nested *serror.Error")
	}

	if s.Error() != "outer" {
		t.Errorf("As matched wrong level: %q, want %q", s.Error(), "outer")
	}
}

func TestLogValue(t *testing.T) { //nolint:cyclop
	t.Parallel()

	cause := errors.New("inner")
	err := Wrap(cause, "outer", "key", "value", "count", 42)

	lv := err.(slog.LogValuer)
	v := lv.LogValue()

	if v.Kind() != slog.KindGroup {
		t.Fatalf("LogValue() kind = %v, want Group", v.Kind())
	}

	attrs := v.Group()
	if attrs == nil {
		t.Fatal("LogValue() group is nil")
	}

	// Check message.
	msg := findAttr(attrs, "message")
	if msg == nil || msg.Value.String() != "outer" {
		t.Errorf("message = %v, want %q", msg, "outer")
	}

	// Check stack group.
	stack := findAttr(attrs, "stack")
	if stack == nil || stack.Value.Kind() != slog.KindGroup {
		t.Fatalf("stack = %v, want Group", stack)
	}

	stackAttrs := stack.Value.Group()

	frames := findAttr(stackAttrs, "frames")
	if frames == nil {
		t.Fatal("stack has no frames")
	}

	frameSlice := frames.Value.Any().([]slog.Value)
	if len(frameSlice) == 0 {
		t.Error("stack frames is empty")
	}

	// Check attrs.
	keyAttr := findAttr(attrs, "key")
	if keyAttr == nil || keyAttr.Value.String() != "value" {
		t.Errorf("key = %v, want %q", keyAttr, "value")
	}

	countAttr := findAttr(attrs, "count")
	if countAttr == nil || countAttr.Value.Int64() != 42 {
		t.Errorf("count = %v, want 42", countAttr)
	}

	// Check cause is a string (stdlib error doesn't implement LogValuer).
	causeAttr := findAttr(attrs, "cause")

	if causeAttr == nil || causeAttr.Value.Kind() != slog.KindString {
		t.Fatalf("cause = %v, want String", causeAttr)
	}

	if causeAttr.Value.String() != "inner" {
		t.Errorf("cause = %q, want %q", causeAttr.Value.String(), "inner")
	}
}

func TestLogValueWithStdlibCause(t *testing.T) {
	t.Parallel()

	err := Wrap(errors.New("std"), "wrapped", "k", "v")
	v := err.(slog.LogValuer).LogValue()

	causeAttr := findAttr(v.Group(), "cause")
	if causeAttr == nil || causeAttr.Value.Kind() != slog.KindString {
		t.Fatalf("cause = %v, want String", causeAttr)
	}

	if causeAttr.Value.String() != "std" {
		t.Errorf("cause = %q, want %q", causeAttr.Value.String(), "std")
	}
}

func TestLogValueNoCause(t *testing.T) {
	t.Parallel()

	err := New("msg", "k", "v").(*Error)
	v := err.LogValue()

	causeAttr := findAttr(v.Group(), "cause")
	if causeAttr != nil {
		t.Errorf("cause = %v, want nil", causeAttr)
	}
}

func TestFormatVerb(t *testing.T) {
	t.Parallel()

	err := New("test error", "k", "v").(*Error)

	// %v should be same as Error()
	if fmt.Sprintf("%v", err) != "test error" {
		t.Errorf("%v = %q, want %q", "%v", fmt.Sprintf("%v", err), "test error")
	}

	// %s should be same as Error()
	if fmt.Sprintf("%s", err) != "test error" {
		t.Errorf("%s = %q, want %q", "%s", fmt.Sprintf("%s", err), "test error")
	}

	// %+v should include stack
	out := fmt.Sprintf("%+v", err)
	if !strings.Contains(out, "test error") {
		t.Errorf("%+v missing message: %q", "%+v", out)
	}
	// pkg/errors style: function name on one line, file:line indented below
	if !strings.Contains(out, "\t") {
		t.Errorf("%+v missing tab indentation: %q", "%+v", out)
	}
	// %+v should include attrs
	if !strings.Contains(out, "k=v") {
		t.Errorf("%+v missing attrs: %q", "%+v", out)
	}

	// %q should quote the message
	if fmt.Sprintf("%q", err) != `"test error"` {
		t.Errorf("%q = %q, want %q", "%q", fmt.Sprintf("%q", err), `"test error"`)
	}
}

func TestFormatVerbFallthrough(t *testing.T) {
	t.Parallel()

	err := New("test error", "k", "v").(*Error)

	// Unhandled verbs must produce exactly what fmt produces for a type
	// without a custom fmt.Formatter, i.e. bareError with the same fields.
	for _, verb := range []rune{'d', 'o', 'x', 'X', 'b', 'e', 'E', 'f', 'F', 'g', 'G', 'c', 't', 'U'} {
		bare := bareError(*err)

		want := fmt.Sprintf("%"+string(verb), &bare)
		if got := fmt.Sprintf("%"+string(verb), err); got != want {
			t.Errorf("%%%c = %q, want %q (standard fmt output)", verb, got, want)
		}
	}

	// The standard fallback is fmt's per-field diagnostics, not the message
	// or empty output.
	const wantPrefix = "&{%!d(string=test error) <nil> map[%!d(string=k):%!d(string=v)] ["
	if got := fmt.Sprintf("%d", err); !strings.HasPrefix(got, wantPrefix) {
		t.Errorf("%%d = %q, want standard fmt diagnostics", got)
	}
}

func TestFormatVerbPlusVMultipleAttrs(t *testing.T) {
	t.Parallel()

	err := New("test error", "k1", "v1", "k2", "v2").(*Error)
	out := fmt.Sprintf("%+v", err)

	if !strings.Contains(out, "k1=v1") {
		t.Errorf("%+v missing first attr: %q", "%+v", out)
	}

	if !strings.Contains(out, "k2=v2") {
		t.Errorf("%+v missing second attr: %q", "%+v", out)
	}

	if !strings.Contains(out, ", ") {
		t.Errorf("%+v missing comma separator between attrs: %q", "%+v", out)
	}
}

func TestFormatVerbPlusVIncludesCause(t *testing.T) {
	t.Parallel()

	cause := errors.New("inner cause")
	err := Wrap(cause, "outer error", "k", "v").(*Error)
	out := fmt.Sprintf("%+v", err)

	if !strings.Contains(out, "outer error") {
		t.Errorf("%+v missing outer message: %q", "%+v", out)
	}

	if !strings.Contains(out, "caused by: inner cause") {
		t.Errorf("%+v missing cause chain: %q", "%+v", out)
	}

	if !strings.Contains(out, "k=v") {
		t.Errorf("%+v missing attrs: %q", "%+v", out)
	}
}

func TestFormatVerbPlusVIncludesSerrorCause(t *testing.T) {
	t.Parallel()

	inner := New("inner error", "inner_k", "inner_v")
	err := Wrap(inner, "outer error", "outer_k", "outer_v").(*Error)
	out := fmt.Sprintf("%+v", err)

	if !strings.Contains(out, "outer error") {
		t.Errorf("%+v missing outer message: %q", "%+v", out)
	}

	if !strings.Contains(out, "caused by: inner error") {
		t.Errorf("%+v missing nested cause message: %q", "%+v", out)
	}

	if !strings.Contains(out, "inner_k=inner_v") {
		t.Errorf("%+v missing nested cause attrs: %q", "%+v", out)
	}
}

func TestFormatDeterministicAttrs(t *testing.T) {
	t.Parallel()

	err := New("test error", "z", 1, "a", 2, "m", 3).(*Error)

	out1 := fmt.Sprintf("%+v", err)
	out2 := fmt.Sprintf("%+v", err)

	if out1 != out2 {
		t.Errorf("%+v output is non-deterministic:\n%q\n%q", "%+v", out1, out2)
	}

	expected := "test error {a=2, m=3, z=1}"
	if !strings.Contains(out1, expected) {
		t.Errorf("%+v = %q, want substring %q", "%+v", out1, expected)
	}
}

func TestFormatMapSorted(t *testing.T) {
	t.Parallel()

	m := map[string]any{"z": 1, "a": 2, "m": 3}
	out := formatMap(m)

	expected := `map[string]any{"a": 2, "m": 3, "z": 1}`
	if out != expected {
		t.Errorf("formatMap = %q, want %q", out, expected)
	}
}

func TestFormatVerbNoStack(t *testing.T) {
	t.Parallel()

	err := New("msg").(*Error)
	// New always captures stack, so %+v should always have it
	out := fmt.Sprintf("%+v", err)
	if !strings.Contains(out, "\n") {
		t.Errorf("%+v should contain newline for stack: %q", "%+v", out)
	}
}

func TestFormatVerbGoSyntax(t *testing.T) {
	t.Parallel()

	err := New("msg", "k", "v").(*Error)
	out := fmt.Sprintf("%#v", err)

	if !strings.HasPrefix(out, "serror.Error{") {
		t.Errorf("%#v should start with serror.Error{, got %q", "%#v", out)
	}

	if !strings.Contains(out, "msg: \"msg\"") {
		t.Errorf("%#v missing msg: %q", "%#v", out)
	}

	if !strings.Contains(out, "\"k\"") {
		t.Errorf("%#v missing attrs: %q", "%#v", out)
	}

	if !strings.Contains(out, "stack:") {
		t.Errorf("%#v missing stack: %q", "%#v", out)
	}

	if !strings.Contains(out, "struct{File string; Line int; Function string}") {
		t.Errorf("%#v missing anonymous frame type: %q", "%#v", out)
	}
}

func TestFormatVerbGoSyntaxWithCause(t *testing.T) {
	t.Parallel()

	cause := errors.New("original")
	err := Wrap(cause, "wrapped", "k", "v").(*Error)
	out := fmt.Sprintf("%#v", err)

	if !strings.Contains(out, "cause:") {
		t.Errorf("%#v missing cause: %q", "%#v", out)
	}

	if !strings.Contains(out, "original") {
		t.Errorf("%#v missing cause message: %q", "%#v", out)
	}
}

func TestFormatVerbGoSyntaxNested(t *testing.T) {
	t.Parallel()

	inner := New("inner", "inner_k", "inner_v")
	err := Wrap(inner, "outer", "outer_k", "outer_v").(*Error)
	out := fmt.Sprintf("%#v", err)

	if !strings.Contains(out, "inner") {
		t.Errorf("%#v missing inner message: %q", "%#v", out)
	}

	if !strings.Contains(out, "outer") {
		t.Errorf("%#v missing outer message: %q", "%#v", out)
	}
}

func TestStackTrace(t *testing.T) {
	t.Parallel()

	err := New("test", "k", "v").(*Error)
	st := err.StackTrace()

	if len(st) == 0 {
		t.Fatal("StackTrace() returned no frames")
	}

	// StackTrace should be []uintptr for Sentry/pkg/errors compat.
	for _, pc := range st {
		if pc == 0 {
			t.Error("StackTrace contains zero PC")
		}
	}
}

func TestStackTracePreservedOnWrap(t *testing.T) {
	t.Parallel()

	original := New("original", "k", "v").(*Error)
	wrapped := Wrap(original, "wrapped").(*Error)

	origST := original.StackTrace()
	wrapST := wrapped.StackTrace()

	if len(wrapST) != len(origST) {
		t.Errorf("stack length %d, want %d", len(wrapST), len(origST))
	}
}

func TestStackTraceWithStdlibCause(t *testing.T) {
	t.Parallel()

	cause := errors.New("original")
	err := Wrap(cause, "wrapped").(*Error)

	st := err.StackTrace()
	if len(st) == 0 {
		t.Fatal("StackTrace() returned no frames for wrapped stdlib error")
	}
}

func TestStackTraceNotSharedOnWithAttrs(t *testing.T) {
	t.Parallel()

	err := New("msg")
	cpy := err.(*Error).WithAttrs("k", "v").(*Error)

	// stackPcs is a slice, shallow copied via *e copy.
	// This is fine since we never modify it, but verify the method works.
	st := cpy.StackTrace()
	if len(st) == 0 {
		t.Fatal("StackTrace() on WithAttrs copy returned no frames")
	}
}

func findAttr(attrs []slog.Attr, key string) *slog.Attr {
	for i := range attrs {
		if attrs[i].Key == key {
			return &attrs[i]
		}
	}

	return nil
}

func TestAttrsNil(t *testing.T) {
	t.Parallel()

	err := &Error{msg: "msg"} //nolint:exhaustruct // minimal error for test

	attrs := err.Attrs()
	if attrs != nil {
		t.Errorf("Attrs() = %v, want nil for error without attributes", attrs)
	}
}

func TestFormatVerbGoSyntaxNestedErrorCause(t *testing.T) {
	t.Parallel()

	inner := New("inner")
	err := &Error{ //nolint:exhaustruct // minimal error for test
		msg:   "outer",
		cause: inner,
	}
	out := fmt.Sprintf("%#v", err)

	if !strings.Contains(out, "cause:") {
		t.Errorf("%#v missing cause: %q", "%#v", out)
	}

	if !strings.Contains(out, "inner") {
		t.Errorf("%#v missing inner message in cause: %q", "%#v", out)
	}
}

func TestFormatVerbPlusVNoAttrsNoStack(t *testing.T) {
	t.Parallel()

	err := &Error{msg: "simple"} //nolint:exhaustruct // minimal error for test
	out := fmt.Sprintf("%+v", err)

	if out != "simple" {
		t.Errorf("%+v = %q, want %q", "%+v", out, "simple")
	}
}

func TestFormatVerbVNoFlags(t *testing.T) {
	t.Parallel()

	err := New("msg", "k", "v").(*Error)
	out := fmt.Sprintf("%v", err)

	if out != "msg" {
		t.Errorf("%v = %q, want %q", "%v", out, "msg")
	}
}

func TestLogValueWithLogValuerCause(t *testing.T) {
	t.Parallel()

	inner := New("inner cause")
	err := &Error{ //nolint:exhaustruct // minimal error for test
		msg:   "outer",
		cause: inner,
		attrs: map[string]any{"key": "value"},
	}

	v := err.LogValue()
	attrs := v.Group()

	causeAttr := findAttr(attrs, "cause")

	if causeAttr == nil {
		t.Fatal("missing cause attr")
	}

	if causeAttr.Value.Kind() != slog.KindGroup {
		t.Errorf("cause kind = %v, want Group (LogValuer)", causeAttr.Value.Kind())
	}
}

func TestLogValueWithNilCause(t *testing.T) {
	t.Parallel()

	err := New("no cause", "k", "v").(*Error)
	v := err.LogValue()
	attrs := v.Group()

	causeAttr := findAttr(attrs, "cause")
	if causeAttr != nil {
		t.Errorf("cause = %v, want nil when no cause", causeAttr)
	}
}

func TestLogValueNoStackGroup(t *testing.T) {
	t.Parallel()

	err := &Error{msg: "no stack"} //nolint:exhaustruct // minimal error for test
	v := err.LogValue()
	attrs := v.Group()

	stackAttr := findAttr(attrs, "stack")
	if stackAttr != nil {
		t.Errorf("stack = %v, want nil when no frames captured", stackAttr)
	}
}

func TestFormatMapNil(t *testing.T) {
	t.Parallel()

	result := formatMap(nil)

	if result != "nil" {
		t.Errorf("formatMap(nil) = %q, want %q", result, "nil")
	}
}

func TestFormatMapEmpty(t *testing.T) {
	t.Parallel()

	result := formatMap(map[string]any{})

	if result != "map[string]any{}" {
		t.Errorf("formatMap(empty) = %q, want %q", result, "map[string]any{}")
	}
}

func TestFormatFramesEmpty(t *testing.T) {
	t.Parallel()

	want := "[]struct{File string; Line int; Function string}{}"

	result := formatFrames(nil)

	if result != want {
		t.Errorf("formatFrames(nil) = %q, want %q", result, want)
	}

	empty := make([]uintptr, 0)
	result = formatFrames(empty)

	if result != want {
		t.Errorf("formatFrames(empty) = %q, want %q", result, want)
	}
}

func TestErrorMethodEmptyMessage(t *testing.T) {
	t.Parallel()

	err := &Error{msg: ""} //nolint:exhaustruct // minimal error for test

	if err.Error() != "" {
		t.Errorf("Error() = %q, want empty string", err.Error())
	}
}

func TestUnwrapNoCause(t *testing.T) {
	t.Parallel()

	err := New("no cause").(*Error)

	unwrapped := err.Unwrap()
	if unwrapped != nil {
		t.Errorf("Unwrap() = %v, want nil for error without cause", unwrapped)
	}
}

func TestUnwrapWithCause(t *testing.T) {
	t.Parallel()

	cause := errors.New("cause")
	err := Wrap(cause, "wrapped").(*Error)
	unwrapped := err.Unwrap()

	if unwrapped != cause {
		t.Error("Unwrap() did not return the cause")
	}
}

func TestSentinel(t *testing.T) {
	t.Parallel()

	const ErrNotFound Sentinel = "not found"

	var err error = ErrNotFound

	if err.Error() != "not found" {
		t.Errorf("Sentinel.Error() = %q, want %q", err.Error(), "not found")
	}

	if !errors.Is(err, ErrNotFound) {
		t.Error("errors.Is(Sentinel, Sentinel) should be true")
	}
}

func TestSentinelWrapped(t *testing.T) {
	t.Parallel()

	const ErrNotFound Sentinel = "not found"

	wrapped := Wrap(ErrNotFound, "failed to fetch")

	if !errors.Is(wrapped, ErrNotFound) {
		t.Error("errors.Is(wrapped Sentinel, Sentinel) should be true")
	}

	if wrapped.Error() != "failed to fetch" {
		t.Errorf("wrapped.Error() = %q, want %q", wrapped.Error(), "failed to fetch")
	}
}

func TestStringState(t *testing.T) {
	t.Parallel()

	var st stringState

	n, err := st.Write([]byte("hello"))
	if n != 5 || err != nil {
		t.Errorf("Write = %d, %v, want 5, nil", n, err)
	}

	if st.b.String() != "hello" {
		t.Errorf("buffer = %q, want %q", st.b.String(), "hello")
	}

	if w, ok := st.Width(); w != 0 || ok {
		t.Errorf("Width = %d, %v, want 0, false", w, ok)
	}

	if p, ok := st.Precision(); p != 0 || ok {
		t.Errorf("Precision = %d, %v, want 0, false", p, ok)
	}

	if st.Flag('#') {
		t.Error("Flag('#') = true, want false")
	}
}
