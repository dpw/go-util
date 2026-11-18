// Package errorsx provides error wrapping with stack traces and rich
// formatting, building on the standard [errors] and [fmt] packages.
// This is similar to functionality provided by
// [github.com/pkg/errors], which has been deprecated since [the
// extensions to the standard errors package in Go 1.13].
//
// The constructors return errors that implement the [Error]
// interface, which extends the standard error interface with
// [fmt.Formatter] support.  [Errorf] creates a new error, capturing a
// stack trace at the point of the call.  [Wrap] and [Wrapf] wrap an
// existing error with additional context, also capturing a stack
// trace, while [WithMessage] and [WithMessagef] wrap without
// capturing a new stack trace.  All of the wrapping functions return
// nil when passed a nil error, so they can be applied unconditionally
// to a result that may or may not be an error.  [Join] combines
// several errors into one, like [errors.Join].
//
// The wrapped errors integrate with the standard library: they
// implement Unwrap, so [errors.Is], [errors.As], and [errors.Unwrap]
// traverse the wrapped chain as usual, and errors joined with
// [errors.Join] are handled too.
//
// The errors support several fmt formatting verbs:
//
//   - %s and %v print the message for the error chain, in the
//     conventional "outer: inner" form.
//   - %+v prints a multi-line representation of the whole error chain,
//     interleaving each message with the stack trace captured at that
//     point (for errors that carry one).
//   - %#v prints a Go-syntax representation of the error value, as usual.
//
// There are two approaches to using this package in a codebase:
//
// First, the gradual way.  You add calls to [Wrap] etc. to enrich
// errors with stack traces where it seems desirable, and continue to
// use the standard [error] type for error values.  This works, but it
// is not thorough.  It makes it easy to forget to [Wrap] a plain
// error value returned from the standard library or some other
// package.
//
// The second way is to use the [Error] type everywhere you would
// normally use the standard [error] type.  Then the compiler rejects
// any attempt to return a plain error value without first converting
// it with [Wrap] or one of the other functions, so errors from the
// standard library and other packages can't slip through unenriched.
//
// [the extensions to the standard errors package in Go 1.13]: https://go.dev/blog/go1.13-errors
package errorsx

import (
	"bytes"
	"fmt"
	"runtime"
	"slices"
	"strings"
)

// Error is an interface that extends the standard error interface
// with [fmt.Formatter] support for rich error formatting.  For errors
// that implement this interface, %+v will produce a multiline
// representation of the full error chain, including stack traces.
// That multiline representation includes intervening newlines, but
// should not end in a newline.
type Error interface {
	error
	fmt.Formatter
}

type wrappedError struct {
	cause      error
	msg        string
	stackTrace []uintptr
}

// Errorf creates a new error with a formatted message and captures
// the stack trace at the point of invocation.  The format string and
// arguments follow the same conventions as [fmt.Sprintf].
func Errorf(format string, a ...any) Error {
	return wrappedError{
		msg:        fmt.Sprintf(format, a...),
		stackTrace: captureStackTrace(),
	}
}

// Wrap wraps an existing error with a message providing additional
// context and captures the stack trace at the point of invocation.
// If err is nil, Wrap returns nil.
func Wrap(err error, msg string) Error {
	if err == nil {
		return nil
	}
	return wrappedError{
		cause:      err,
		msg:        msg,
		stackTrace: captureStackTrace(),
	}
}

// Wrapf wraps an existing error with a formatted message and captures
// the stack trace at the point of invocation.  The format string and
// arguments follow the same conventions as [fmt.Sprintf].  If err is
// nil, Wrapf returns nil.
func Wrapf(err error, format string, a ...any) Error {
	if err == nil {
		return nil
	}
	return wrappedError{
		cause:      err,
		msg:        fmt.Sprintf(format, a...),
		stackTrace: captureStackTrace(),
	}
}

// WithMessage wraps an existing error with a message providing
// additional context, but does not capture a stack trace.  If err is
// nil, WithMessage returns nil.
func WithMessage(err error, msg string) Error {
	if err == nil {
		return nil
	}
	return wrappedError{
		cause: err,
		msg:   msg,
	}
}

// WithMessagef wraps an existing error with a formatted message, but
// does not capture a stack trace.  The format string and arguments
// follow the same conventions as [fmt.Sprintf].  If err is nil,
// WithMessagef returns nil.
func WithMessagef(err error, format string, a ...any) Error {
	if err == nil {
		return nil
	}
	return wrappedError{
		cause: err,
		msg:   fmt.Sprintf(format, a...),
	}
}

func captureStackTrace() []uintptr {
	// Limit the length of stack traces included in errors to the
	// innermost 32 frames, to avoid ridiculously large errors.
	var pcs [32]uintptr

	// The skip count of 3 discards the frames for runtime.Callers,
	// captureStackTrace itself, and the exported errorsx function that
	// called it (Errorf, Wrap, etc.).
	n := runtime.Callers(3, pcs[:])
	return pcs[:n]
}

func (w wrappedError) Unwrap() error {
	return w.cause
}

func (w wrappedError) Error() string {
	if w.cause == nil {
		return w.msg
	}
	return w.msg + ": " + w.cause.Error()
}

func (w wrappedError) Format(f fmt.State, verb rune) {
	if verb == 'v' {
		switch {
		case f.Flag('#'):
			// The '#' flag takes precedence over the '+'
			// flag.
			formatHashV(f, w)
			return

		case f.Flag('+'):
			// The "%+v" case.  Print the error chain in a
			// multi-line format, starting with the
			// outermost error message.
			fst := toFmtState(f)
			fst.newline()
			fst.Write([]byte(w.msg))

			if len(w.stackTrace) != 0 {
				frames := runtime.CallersFrames(w.stackTrace)
				for {
					frame, more := frames.Next()
					fst.newline()
					fmt.Fprintf(fst, "%s\n\t%s:%d",
						frame.Function,
						frame.File, frame.Line)
					if !more {
						break
					}
				}
			}

			if w.cause != nil {
				toErrorTree(w.cause).formatPlusV(fst)
			}

			return
		}
	}

	fmt.Fprintf(f, fmt.FormatString(f, verb), w.Error())
}

// If a type implements fmt.Formatter, the fmt package will use it for
// all formatting, including the "%#v" format string.  It's usually
// desirable for that to produce the conventional go-syntax
// representation.  But the fmt package does not provide a direct way
// to generate that representation.  So we have to generate it
// indirectly.
func formatHashV(f fmt.State, val any) {
	type wrapper struct{ x any }
	s := fmt.Sprintf("%#v", wrapper{x: val})
	s = strings.TrimPrefix(s, "errorsx.wrapper{x:")
	s = strings.TrimSuffix(s, "}")
	f.Write([]byte(s))
}

// Join returns an error that wraps the given errors, like
// [errors.Join] but taking and returning [Error] values.  Any nil
// errors are discarded, and Join returns nil if every error is nil.
// The error message consists of the messages of the non-nil errors,
// separated by newlines.  Like [errors.Join], and unlike [Wrap], Join
// does not capture a stack trace.
//
// The returned error has an Unwrap method returning []error, and
// with %+v it prints each of the wrapped errors in full, including
// their stack traces.
func Join(errs ...Error) Error {
	n := 0
	for _, err := range errs {
		if err != nil {
			n++
		}
	}
	if n == 0 {
		return nil
	}

	e := joinedError{errs: make([]error, 0, n)}
	for _, err := range errs {
		if err != nil {
			e.errs = append(e.errs, err)
		}
	}
	return e
}

type joinedError struct {
	errs []error
}

func (j joinedError) Unwrap() []error {
	return j.errs
}

func (j joinedError) Error() string {
	var b strings.Builder
	for i, err := range j.errs {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(err.Error())
	}
	return b.String()
}

func (j joinedError) Format(f fmt.State, verb rune) {
	if verb == 'v' {
		switch {
		case f.Flag('#'):
			// The '#' flag takes precedence over the '+'
			// flag.
			formatHashV(f, j)
			return

		case f.Flag('+'):
			causes := make([]errorTree, len(j.errs))
			for i, e := range j.errs {
				causes[i] = toErrorTree(e)
			}
			formatCauses(toFmtState(f), causes)
			return
		}
	}

	fmt.Fprintf(f, fmt.FormatString(f, verb), j.Error())
}

type errorTree struct {
	// The error message, from calling the Error() method.
	msg []byte

	// If the error is a fmt.Formatter instance, it can take care
	// of formatting itself and its causes.
	formatter fmt.Formatter

	// Otherwise, this holds the causes.
	causes []errorTree
}

func toErrorTree(err error) errorTree {
	res := errorTree{msg: []byte(err.Error())}

	switch u := err.(type) {
	case fmt.Formatter:
		res.formatter = u

	case interface{ Unwrap() []error }:
		errs := u.Unwrap()
		res.causes = make([]errorTree, 0, len(errs))
		for _, e := range errs {
			// We expect the main source of the
			// multiple-cause case to be errors.Join,
			// which filters out nils from its argument.
			// But there could be an independent
			// implementation which doesn't.
			if e != nil {
				res.causes = append(res.causes, toErrorTree(e))
			}
		}

	case interface{ Unwrap() error }:
		if e := u.Unwrap(); e != nil {
			res.causes = []errorTree{toErrorTree(e)}
		}
	}

	return res
}

func (et errorTree) formatPlusV(fst *fmtState) {
	if et.formatter != nil {
		// The formatter might not start with a newline
		// itself, so ensure we are at the start of a line.
		fst.newline()
		et.formatter.Format(fst, 'v')
		return
	}

	// If this error has a useful message, print it.
	if msg := et.trimmedMessage(); len(msg) != 0 {
		fst.newline()
		fst.Write(msg)
	}

	formatCauses(fst, et.causes)
}

func formatCauses(fst *fmtState, causes []errorTree) {
	switch len(causes) {
	case 0:
		// Nothing to do
	case 1:
		// Simply write the single cause.
		causes[0].formatPlusV(fst)
	default:
		// Write the causes with indentation,
		// interspersed with the separating messages.
		sep := joinedIntroLine
		for _, cause := range causes {
			fst.newline()
			fst.Write(sep)
			fst.prefix = append(fst.prefix, '\t')
			cause.formatPlusV(fst)
			fst.prefix = fst.prefix[:len(fst.prefix)-1]
			sep = joinedNextLine
		}
	}
}

var joinedIntroLine = []byte("multiple errors:\n")
var joinedNextLine = []byte("and:\n")

// Determine the error message to be printed for the error at the root
// of the errorTree.
//
// This may be the same as the result of the error's Error()
// method. But if that error includes a cause, and the message repeats
// the message of the cause as a suffix, it is trimmed off.  In the
// case where the message is identical to the cause message, a
// zero-length slice is returned.
func (et errorTree) trimmedMessage() []byte {
	if len(et.causes) == 0 {
		return et.msg
	}

	causeMsg := slices.Clip(et.causes[0].msg)
	for _, cause := range et.causes[1:] {
		// In the multiple-cause case, recapitulate the
		// message produced by errors.Join.
		causeMsg = append(causeMsg, '\n')
		causeMsg = append(causeMsg, cause.msg...)
	}

	// Look for messages of the form: "error: chained error", and
	// remove the cause part.
	msg := et.msg
	if bytes.HasSuffix(msg, causeMsg) {
		if len(msg) == len(causeMsg) {
			msg = nil
		} else {
			// There should be some kind of separator
			// before the cause message.
			l := len(msg) - len(causeMsg)
			trimmed := bytes.TrimRight(msg[:l], " \t\n:")
			if len(trimmed) != l {
				msg = trimmed
			}
		}
	}

	return msg
}

// A fmt.State associated with an indentWriter.
type fmtState struct {
	fmt.State
	indentWriter
}

// Convert a fmt.State to a fmtState.  If the fmt.State is not
// already a fmtState, then wrap its Writer with an indentWriter.
func toFmtState(f fmt.State) *fmtState {
	fst, ok := f.(*fmtState)
	if !ok {
		w := indentWriter{Writer: f}
		fst = &fmtState{
			State:        f,
			indentWriter: w,
		}
	}
	return fst
}

func (fst *fmtState) Write(b []byte) (int, error) {
	return fst.indentWriter.Write(b)
}
