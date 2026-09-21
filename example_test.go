package serror

import (
	"errors"
	"fmt"
)

func ExampleNew() {
	err := New("something went wrong")
	fmt.Println(err.Error())

	// Output:
	// something went wrong
}

func ExampleNew_withAttrs() {
	err := New("something went wrong", "user", "alice", "attempt", 3)
	e := err.(*Error)
	fmt.Println(e.Error())

	attrs := e.Attrs()
	fmt.Println(attrs["user"])
	fmt.Println(attrs["attempt"])

	// Output:
	// something went wrong
	// alice
	// 3
}

func ExampleWrap() {
	cause := errors.New("disk failure")
	err := Wrap(cause, "failed to read file")

	fmt.Println(err.Error())
	fmt.Println(errors.Is(err, cause))

	// Output:
	// failed to read file
	// true
}

func ExampleSentinel() {
	const ErrNotFound Sentinel = "not found"

	err := Wrap(ErrNotFound, "failed to fetch resource")

	fmt.Println(err.Error())
	fmt.Println(errors.Is(err, ErrNotFound))

	// Output:
	// failed to fetch resource
	// true
}

func ExampleWrap_serrorError() {
	original := New("original error")
	wrapped := Wrap(original, "wrapped error")

	fmt.Println(wrapped.Error())
	fmt.Println(wrapped == original)
	fmt.Println(errors.Is(wrapped, original))

	// Output:
	// wrapped error
	// false
	// true
}

func ExampleError_Format() {
	err := New("test error", "key", "value")

	// %v and %s produce the message only.
	fmt.Printf("%%v: %v\n", err)
	fmt.Printf("%%s: %s\n", err)

	// %q quotes the message.
	fmt.Printf("%%q: %q\n", err)

	// Output:
	// %v: test error
	// %s: test error
	// %q: "test error"
}

func ExampleError_Unwrap() {
	cause := errors.New("inner error")
	err := Wrap(cause, "outer error")

	unwrapped := errors.Unwrap(err)
	fmt.Println(unwrapped)

	var s *Error
	fmt.Println(errors.As(err, &s))

	// Output:
	// inner error
	// true
}
