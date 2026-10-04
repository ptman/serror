package serror

import (
	"errors"
	"fmt"
	"log/slog"
	"testing"
)

func BenchmarkNew(b *testing.B) {
	b.ReportAllocs()

	for b.Loop() {
		_ = New("error")
	}
}

func BenchmarkNewWithAttrs(b *testing.B) {
	b.ReportAllocs()

	for b.Loop() {
		_ = New("error", "k1", "v1", "k2", 42)
	}
}

func BenchmarkWrapStdlib(b *testing.B) {
	cause := errors.New("original")

	b.ReportAllocs()

	for b.Loop() {
		_ = Wrap(cause, "wrapped")
	}
}

func BenchmarkWrapSerror(b *testing.B) {
	cause := New("original")

	b.ReportAllocs()

	for b.Loop() {
		_ = Wrap(cause, "wrapped")
	}
}

func BenchmarkWithAttrs(b *testing.B) {
	err := New("error")

	b.ReportAllocs()

	for b.Loop() {
		_ = err.(*Error).WithAttrs("k", "v")
	}
}

func BenchmarkError(b *testing.B) {
	err := New("error", "k", "v")

	b.ReportAllocs()

	for b.Loop() {
		_ = err.Error()
	}
}

func BenchmarkFormatV(b *testing.B) {
	err := New("error", "k", "v")

	b.ReportAllocs()

	for b.Loop() {
		_ = fmt.Sprintf("%v", err)
	}
}

func BenchmarkFormatPlusV(b *testing.B) {
	err := New("error", "k", "v")

	b.ReportAllocs()

	for b.Loop() {
		_ = fmt.Sprintf("%+v", err)
	}
}

func BenchmarkFormatHashV(b *testing.B) {
	err := New("error", "k", "v")

	b.ReportAllocs()

	for b.Loop() {
		_ = fmt.Sprintf("%#v", err)
	}
}

func BenchmarkLogValue(b *testing.B) {
	err := New("error", "k", "v")
	lv := err.(slog.LogValuer)

	b.ReportAllocs()

	for b.Loop() {
		_ = lv.LogValue()
	}
}

func BenchmarkUnwrap(b *testing.B) {
	cause := errors.New("original")
	err := Wrap(cause, "wrapped")

	b.ReportAllocs()

	for b.Loop() {
		_, _ = errors.Unwrap(err), err
	}
}

func BenchmarkStackTrace(b *testing.B) {
	err := New("error").(*Error)

	b.ReportAllocs()

	for b.Loop() {
		_ = err.StackTrace()
	}
}
