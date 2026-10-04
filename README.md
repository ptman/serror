# serror

Errors with stack traces and attributes.

## Overview

`serror` provides an error type that carries key-value attributes and a stack
trace. It implements the standard `error` interface and is compatible with
`errors.Is`/`errors.As` and the Sentry Go SDK.

## API

```go
import "github.com/ptman/serror"

// Create a new error with optional attributes.
err := serror.New("failed to process", "user", userID, "attempt", 3)

// Wrap an existing error, preserving its stack trace.
err := serror.Wrap(innerErr, "context: upload")

// Add attributes to an existing error.
err := originalErr.(*serror.Error).WithAttrs("retry", true)

// Access attributes.
attrs := err.(*serror.Error).Attrs()
```

## Formatting

| Verb | Output |
|------|--------|
| `%v` | message only |
| `%s` | message only |
| `%q` | quoted message |
| `%+v` | message, attributes, stack trace |
| `%#v` | Go-syntax representation |
| other | standard fmt output for the value, e.g. `%!d(string=...)` per field |

## Logging

`serror.Error` implements `slog.LogValuer`. When passed to `slog`, it produces
structured output with a `message`, a `stack` group, a `cause` when the error
wraps another, and the error's attributes.

## Sentinel

Define compile-time constant errors without boilerplate:

```go
import (
	"errors"

	"github.com/ptman/serror"
)

const ErrNotFound serror.Sentinel = "not found"

err := serror.Wrap(ErrNotFound, "failed to fetch resource")
if errors.Is(err, ErrNotFound) { ... }
```
