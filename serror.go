// Package serror provides a structured error type with attributes and stack traces,
// compatible with the Sentry Go SDK and standard library errors.
package serror

import (
	"fmt"
	"log/slog"
	"maps"
	"runtime"
	"sort"
)

// Sentinel is a string type that implements the error interface,
// allowing constant errors to be defined without boilerplate:
//
//	const ErrNotFound Sentinel = "not found"
//
//	sentry.CaptureException(serror.Wrap(ErrNotFound, "failed to fetch resource"))
//	if errors.Is(err, ErrNotFound) { ... }
type Sentinel string //nolint:errname

func (s Sentinel) Error() string {
	return string(s)
}

// Error is a wrapped error with message, attributes, and stack trace.
// Stack frames are resolved lazily from stackPcs when formatted or logged.
type Error struct {
	msg      string
	cause    error
	attrs    map[string]any
	stackPcs []uintptr
}

// StackTrace is a slice of program counters for compatibility with
// pkg/errors and Sentry Go SDK.
type StackTrace []uintptr

// StackTrace implements the pkg/errors and Sentry Go SDK interface.
// Sentry uses reflection to look for a StackTrace() method and extracts
// program counters from the returned value.
func (e *Error) StackTrace() StackTrace {
	return e.stackPcs
}

// attrsFromPairs builds a map from alternating key-value pairs.
// pairs are key-value tuples: key1, value1, key2, value2, ...
// Keys must be strings; non-string keys and orphaned values are ignored.
func attrsFromPairs(pairs []any) map[string]any {
	//nolint:mnd // pairs contain two elements per attribute.
	attrs := make(map[string]any, len(pairs)/2)
	addPairs(attrs, pairs)

	return attrs
}

// addPairs appends alternating key-value pairs to attrs. Keys must be strings;
// non-string keys and orphaned values are ignored.
func addPairs(attrs map[string]any, pairs []any) {
	for i := 0; i+1 < len(pairs); i += 2 {
		if k, ok := pairs[i].(string); ok {
			attrs[k] = pairs[i+1]
		}
	}
}

// New creates a new error with the given message and optional key-value
// attributes. Attribute keys must be strings.
func New(msg string, pairs ...any) error {
	attrs := attrsFromPairs(pairs)

	stackPcs := captureStack(1)

	//nolint:exhaustruct // cause is intentionally nil for a fresh error.
	return &Error{
		msg:      msg,
		attrs:    attrs,
		stackPcs: stackPcs,
	}
}

// Wrap wraps an existing error with a new message and optional key-value
// attributes. Attribute keys must be strings. If err is nil, Wrap returns nil.
// If err is already an serror.Error, its stack trace is preserved and the
// original error is chained as the cause. Otherwise a new stack trace is captured.
func Wrap(err error, msg string, pairs ...any) error {
	if err == nil {
		return nil
	}

	attrs := attrsFromPairs(pairs)

	if e, ok := err.(*Error); ok {
		mergedAttrs := make(map[string]any, len(e.attrs)+len(attrs))
		maps.Copy(mergedAttrs, e.attrs)
		maps.Copy(mergedAttrs, attrs)

		return &Error{
			msg:      msg,
			cause:    e,
			attrs:    mergedAttrs,
			stackPcs: e.stackPcs,
		}
	}

	stackPcs := captureStack(1)

	return &Error{
		msg:      msg,
		cause:    err,
		attrs:    attrs,
		stackPcs: stackPcs,
	}
}

// WithAttrs returns a copy of the error with additional key-value attributes.
// pairs are key-value tuples: key1, value1, key2, value2, ...
// Attribute keys must be strings.
func (e *Error) WithAttrs(pairs ...any) error {
	cpy := *e
	cpy.attrs = make(map[string]any, len(e.attrs)+len(pairs)/2)
	maps.Copy(cpy.attrs, e.attrs)
	addPairs(cpy.attrs, pairs)

	return &cpy
}

// Attrs returns a copy of the error's attributes map.
func (e *Error) Attrs() map[string]any {
	if e.attrs == nil {
		return nil
	}

	cpy := make(map[string]any, len(e.attrs))
	maps.Copy(cpy, e.attrs)

	return cpy
}

// Error implements the error interface.
func (e *Error) Error() string {
	return e.msg
}

// bareError is an Error without the fmt.Formatter method. Formatting it
// reproduces fmt's standard output for an error type without a custom
// formatter, which Format uses for verbs it does not handle explicitly.
type bareError Error

func (e *bareError) Error() string {
	return e.msg
}

// Format implements the fmt.Formatter interface.
// %v and %s produce the message only.
// %+v produces the message followed by the formatted stack trace.
// %#v produces a Go-syntax representation with message, attrs, and stack.
// %q produces the quoted message.
// All other verbs produce the standard fmt output for the value, as if the
// error did not implement fmt.Formatter.
func (e *Error) Format(s fmt.State, verb rune) {
	switch verb {
	case 'v':
		switch {
		case s.Flag('#'):
			e.formatGoSyntax(s)
		case s.Flag('+'):
			e.formatDetailed(s)
		default:
			write(s, e.msg)
		}
	case 's':
		write(s, e.msg)
	case 'q':
		writef(s, "%q", e.msg)
	default:
		bare := bareError(*e)
		writef(s, "%"+string(verb), &bare)
	}
}

// Unwrap returns the wrapped underlying error for compatibility with
// errors.Is and errors.As.
func (e *Error) Unwrap() error {
	return e.cause
}

// LogValue implements the slog.LogValuer interface.
func (e *Error) LogValue() slog.Value {
	var values []slog.Attr

	values = append(values, slog.String("message", e.msg))

	if len(e.stackPcs) > 0 {
		values = append(values, slog.Group("stack", attrToAny(stackGroupValues(e.stackPcs))...))
	}

	if e.cause != nil {
		if lv, ok := e.cause.(slog.LogValuer); ok {
			values = append(values, slog.Any("cause", lv.LogValue()))
		} else {
			values = append(values, slog.String("cause", e.cause.Error()))
		}
	}

	forEachSortedAttr(e.attrs, func(k string, v any) {
		values = append(values, slog.Any(k, v))
	})

	return slog.GroupValue(values...)
}

func (e *Error) formatGoSyntax(s fmt.State) {
	write(s, "serror.Error{")
	writef(s, "msg: %q", e.msg)

	if len(e.attrs) > 0 {
		write(s, ", attrs: ")
		writeMap(s, e.attrs)
	}

	if len(e.stackPcs) > 0 {
		write(s, ", stack: ")
		writeFrames(s, e.stackPcs)
	}

	if e.cause != nil {
		write(s, ", cause: ")

		if causeErr, ok := e.cause.(*Error); ok {
			write(s, causeErr)
		} else {
			writef(s, "%q", e.cause.Error())
		}
	}

	write(s, "}")
}

func (e *Error) formatDetailed(s fmt.State) {
	write(s, e.msg)

	if len(e.attrs) > 0 {
		write(s, " {")

		first := true

		forEachSortedAttr(e.attrs, func(k string, v any) {
			if !first {
				write(s, ", ")
			}

			writef(s, "%s=%v", k, v)

			first = false
		})

		write(s, "}")
	}

	if len(e.stackPcs) > 0 {
		write(s, "\n")
		writeStack(s, e.stackPcs)
	}

	if e.cause != nil {
		write(s, "\ncaused by: ")

		if causeErr, ok := e.cause.(*Error); ok {
			causeErr.formatDetailed(s)
		} else {
			write(s, e.cause.Error())
		}
	}
}

// write and writef are small wrappers around fmt.Fprint/fprintf that absorb
// write errors. The fmt.Formatter contract treats these as unrecoverable.
func write(s fmt.State, v ...any) {
	//nolint:errcheck // fmt.Formatter writes to io.Writer; errors are unrecoverable.
	fmt.Fprint(s, v...)
}

func writef(s fmt.State, format string, v ...any) {
	//nolint:errcheck // fmt.Formatter writes to io.Writer; errors are unrecoverable.
	fmt.Fprintf(s, format, v...)
}

func stackGroupValues(pcs []uintptr) []slog.Attr {
	if len(pcs) == 0 {
		return nil
	}

	iter := runtime.CallersFrames(pcs)
	entries := make([]slog.Value, 0, len(pcs))

	for {
		frame, more := iter.Next()

		entries = append(entries, slog.GroupValue(
			slog.String("file", frame.File),
			slog.Int("line", frame.Line),
			slog.String("function", frame.Function),
		))

		if !more {
			break
		}
	}

	return []slog.Attr{slog.Any("frames", entries)}
}

func attrToAny(attrs []slog.Attr) []any {
	out := make([]any, len(attrs))
	for i, a := range attrs {
		out[i] = a
	}

	return out
}

// sortedAttrKeys returns the keys of attrs in alphabetical order.
// Callers must handle the empty-map case themselves.
func sortedAttrKeys(attrs map[string]any) []string {
	keys := make([]string, 0, len(attrs))
	for k := range attrs {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	return keys
}

// forEachSortedAttr calls fn for each key-value pair in attrs in alphabetical
// order. It avoids allocations for the common cases of 0, 1, or 2 attributes.
func forEachSortedAttr(attrs map[string]any, fn func(k string, v any)) {
	switch len(attrs) {
	case 0:
		return
	case 1:
		for k, v := range attrs {
			fn(k, v)
		}
	//nolint:mnd // two attributes are handled as a special fast path.
	case 2:
		var k1, k2 string

		var v1, v2 any

		i := 0
		for k, v := range attrs {
			if i == 0 {
				k1, v1 = k, v
			} else {
				k2, v2 = k, v
			}

			i++
		}

		if k1 <= k2 {
			fn(k1, v1)
			fn(k2, v2)
		} else {
			fn(k2, v2)
			fn(k1, v1)
		}
	default:
		for _, k := range sortedAttrKeys(attrs) {
			fn(k, attrs[k])
		}
	}
}

func writeMap(s fmt.State, m map[string]any) {
	if m == nil {
		write(s, "nil")

		return
	}

	write(s, "map[string]any{")

	first := true

	forEachSortedAttr(m, func(k string, v any) {
		if !first {
			write(s, ", ")
		}

		writef(s, "%q: %v", k, v)

		first = false
	})

	write(s, "}")
}

func writeFrames(s fmt.State, pcs []uintptr) {
	const frameType = "struct{File string; Line int; Function string}"

	if len(pcs) == 0 {
		writef(s, "[]%s{}", frameType)

		return
	}

	writef(s, "[]%s{", frameType)

	iter := runtime.CallersFrames(pcs)

	for i := 0; ; i++ {
		frame, more := iter.Next()

		if i > 0 {
			write(s, ", ")
		}

		writef(s, "{File: %q, Line: %d, Function: %q}", frame.File, frame.Line, frame.Function)

		if !more {
			break
		}
	}

	write(s, "}")
}
