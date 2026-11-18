package errorsx_test

import (
	"errors"
	"fmt"
	"os"

	"github.com/dpw/go-util/errorsx"
)

// The examples below print errors with "%+v", which includes stack
// traces.  The exact output depends on file paths and line numbers,
// so the examples do not specify it.

func ExampleErrorf() {
	err := errorsx.Errorf("invalid port %d", 99999)

	// Prints the message, followed by the stack trace captured by
	// Errorf.
	fmt.Printf("%+v\n", err)
}

func ExampleWrap() {
	_, err := os.Open("/nonexistent")

	// Wrap adds context to the error from os.Open, and captures a
	// stack trace.
	wrapped := errorsx.Wrap(err, "loading configuration")

	// Prints "loading configuration: open /nonexistent: no such
	// file or directory".
	fmt.Println(wrapped)

	// Prints "loading configuration" and the stack trace, followed
	// by the os.Open error chain.
	fmt.Printf("%+v\n", wrapped)
}

func ExampleWrapf() {
	path := "/nonexistent"
	_, err := os.Open(path)
	wrapped := errorsx.Wrapf(err, "loading configuration from %s", path)

	// Prints "loading configuration from /nonexistent" and the
	// stack trace, followed by the os.Open error chain.
	fmt.Printf("%+v\n", wrapped)
}

func ExampleWithMessage() {
	base := errorsx.Errorf("connection refused")

	// WithMessage adds context without capturing another stack
	// trace, which is useful when the cause already has one that
	// covers the same call path.
	err := errorsx.WithMessage(base, "fetching user")

	// Prints "fetching user", then "connection refused" and the
	// stack trace captured by Errorf.
	fmt.Printf("%+v\n", err)
}

func ExampleWithMessagef() {
	base := errorsx.Errorf("connection refused")
	err := errorsx.WithMessagef(base, "fetching user %d", 42)

	// Prints "fetching user 42", then "connection refused" and the
	// stack trace captured by Errorf.
	fmt.Printf("%+v\n", err)
}

var errNotFound = errors.New("not found")

func lookup(key string) errorsx.Error {
	if key == "missing" {
		return errorsx.Wrapf(errNotFound, "looking up %q", key)
	}
	return nil
}

func ExampleJoin() {
	// Collect the errors from several operations.  The nil
	// results of successful operations are discarded.
	var errs []errorsx.Error
	for _, key := range []string{"present", "missing", "another"} {
		errs = append(errs, lookup(key))
	}
	errs = append(errs, errorsx.Errorf("giving up after %d keys", 3))
	err := errorsx.Join(errs...)

	// The joined error matches each of the errors it wraps.
	if errors.Is(err, errNotFound) {
		// Prints "multiple errors:", followed by each of the
		// joined errors with its stack trace, indented.
		fmt.Printf("%+v\n", err)
	}
}
