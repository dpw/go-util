package errorsx

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/alecthomas/assert/v2"
)

type printedError struct {
	message           string
	printfPlusVRegexp string
	printfHashVRegexp string
}

func requirePrintedError(t *testing.T, err error, expect printedError) {
	assert.Equal(t, expect.message, err.Error())
	assert.Equal(t, expect.message, fmt.Sprintf("%s", err))
	assert.Equal(t, expect.message, fmt.Sprintf("%v", err))

	// Allow the cumbersome regexp for a stack trace to be
	// abbreviated as "<starting function name>".
	re := strings.ReplaceAll(strings.ReplaceAll(expect.printfPlusVRegexp, `<`, `github.com/dpw/go-util/errorsx\.`), `>`, `\n\t+.*/errorsx_test\.go:\d+(?:\n\t*\S+\n\t+[^:]+:\d+)+`)
	assertRegexp(t, re, fmt.Sprintf("%+v", err))
	assertRegexp(t, expect.printfHashVRegexp, fmt.Sprintf("%#v", err))
}

func assertRegexp(t *testing.T, pattern, s string) {
	t.Helper()
	assert.True(t, regexp.MustCompile(pattern).MatchString(s),
		"pattern %q did not match %q", pattern, s)
}

func TestErrorf(t *testing.T) {
	requirePrintedError(t, Errorf("foo %s %d", "bar", 42), printedError{
		message:           "foo bar 42",
		printfPlusVRegexp: `^foo bar 42\n<TestErrorf>`,
		printfHashVRegexp: `^errorsx.wrappedError{cause:.*, msg:"foo bar 42", stackTrace:.*}$`,
	})
}

func TestWrap(t *testing.T) {
	baseErr := Errorf("base error")
	wrapped := Wrap(baseErr, "wrapped")
	requirePrintedError(t, wrapped, printedError{
		message:           "wrapped: base error",
		printfPlusVRegexp: `^wrapped\n<TestWrap>\nbase error\n<TestWrap>$`,
		printfHashVRegexp: `^errorsx\.wrappedError\{cause:.*, msg:"wrapped", stackTrace:.*}$`,
	})
	assert.Equal[error](t, baseErr, errors.Unwrap(wrapped))

	doubleWrapped := Wrap(wrapped, "outer")
	requirePrintedError(t, doubleWrapped, printedError{
		message:           "outer: wrapped: base error",
		printfPlusVRegexp: `^outer\n<TestWrap>\nwrapped\n<TestWrap>\nbase error\n<TestWrap>$`,
		printfHashVRegexp: `^errorsx\.wrappedError\{cause:errorsx\.wrappedError\{cause:.*, msg:"wrapped", stackTrace:.*}, msg:"outer", stackTrace:.*\}$`,
	})
	assert.Equal[error](t, baseErr, errors.Unwrap(errors.Unwrap(doubleWrapped)))

	// Wrapping nil should return nil
	assert.Zero(t, Wrap(nil, "something failed"))
}

func TestWrapStandardError(t *testing.T) {
	baseErr := errors.New("base error")
	wrapped := Wrap(baseErr, "wrapped")
	requirePrintedError(t, wrapped, printedError{
		message:           "wrapped: base error",
		printfPlusVRegexp: `^wrapped\n<TestWrapStandardError>\nbase error$`,
		printfHashVRegexp: `^errorsx\.wrappedError\{cause:.*, msg:"wrapped", stackTrace:.*}$`,
	})
	assert.Equal(t, baseErr, errors.Unwrap(wrapped))
}

// Tests the trimming of a single cause's message from the end of a
// standard error's message, using errors from fmt.Errorf with %w.
func TestWrapSingleCauseSuffix(t *testing.T) {
	cause := errors.New("fix")
	for _, tc := range []struct {
		name      string
		err       error
		plusVBody string
	}{{
		name:      "colon separator",
		err:       fmt.Errorf("prefix: %w", cause),
		plusVBody: `prefix\nfix`,
	}, {
		name:      "space separator",
		err:       fmt.Errorf("prefix %w", cause),
		plusVBody: `prefix\nfix`,
	}, {
		// Without a separator, the cause message is not
		// trimmed, even though it is a suffix.
		name:      "no separator",
		err:       fmt.Errorf("prefix%w", cause),
		plusVBody: `prefixfix\nfix`,
	}, {
		// The cause message is not a suffix.
		name:      "not a suffix",
		err:       fmt.Errorf("%w happened", cause),
		plusVBody: `fix happened\nfix`,
	}} {
		// Wrap outside the subtest, so that the stack trace
		// starts with TestWrapSingleCauseSuffix.
		err := Wrap(tc.err, "wrapped")
		t.Run(tc.name, func(t *testing.T) {
			requirePrintedError(t, err, printedError{
				message:           "wrapped: " + tc.err.Error(),
				printfPlusVRegexp: `^wrapped\n<TestWrapSingleCauseSuffix>\n` + tc.plusVBody + `$`,
				printfHashVRegexp: `^errorsx\.wrappedError\{cause:.*, msg:"wrapped", stackTrace:.*}$`,
			})
		})
	}
}

// multiCauseError is like the errors produced by errors.Join, but with
// its own message.  Unlike errors.Join, it does not filter out nil
// causes.
type multiCauseError struct {
	msg    string
	causes []error
}

func (e multiCauseError) Error() string   { return e.msg }
func (e multiCauseError) Unwrap() []error { return e.causes }

func TestWrapMultiCause(t *testing.T) {
	a, b := errors.New("a"), errors.New("b")
	for _, tc := range []struct {
		name      string
		err       multiCauseError
		plusVBody string
	}{{
		// A message unrelated to the causes is printed in
		// full.
		name:      "unrelated",
		err:       multiCauseError{"custom failure", []error{a, b}},
		plusVBody: `custom failure\nmultiple errors:\n\ta\nand:\n\tb`,
	}, {
		// A message that combines the causes differently
		// from errors.Join is printed in full.
		name:      "differently combined",
		err:       multiCauseError{"custom: a; b", []error{a, b}},
		plusVBody: `custom: a; b\nmultiple errors:\n\ta\nand:\n\tb`,
	}, {
		// A message with the joined causes as a suffix has
		// that suffix trimmed off.
		name:      "suffix",
		err:       multiCauseError{"custom: a\nb", []error{a, b}},
		plusVBody: `custom\nmultiple errors:\n\ta\nand:\n\tb`,
	}, {
		// A message that is the same as from errors.Join is
		// not printed at all.
		name:      "same as join",
		err:       multiCauseError{"a\nb", []error{a, b}},
		plusVBody: `multiple errors:\n\ta\nand:\n\tb`,
	}, {
		// Nil causes are ignored, including when recreating
		// the message from errors.Join.
		name:      "nil cause",
		err:       multiCauseError{"custom: a\nb", []error{a, nil, b}},
		plusVBody: `custom\nmultiple errors:\n\ta\nand:\n\tb`,
	}, {
		// If there is only one non-nil cause, it is printed
		// as a simple chain.
		name:      "single non-nil cause",
		err:       multiCauseError{"custom: a", []error{nil, a}},
		plusVBody: `custom\na`,
	}, {
		// The joined causes are only trimmed if there is a
		// separator before them.  Otherwise the message is
		// printed in full.
		name:      "suffix without separator",
		err:       multiCauseError{"banana\nb", []error{a, b}},
		plusVBody: `banana\nb\nmultiple errors:\n\ta\nand:\n\tb`,
	}, {
		name:      "suffix after colon",
		err:       multiCauseError{"custom:a\nb", []error{a, b}},
		plusVBody: `custom\nmultiple errors:\n\ta\nand:\n\tb`,
	}, {
		name:      "suffix after newline",
		err:       multiCauseError{"custom\na\nb", []error{a, b}},
		plusVBody: `custom\nmultiple errors:\n\ta\nand:\n\tb`,
	}} {
		// Wrap outside the subtest, so that the stack trace
		// starts with TestWrapMultiCause.
		err := Wrap(tc.err, "wrapped")
		t.Run(tc.name, func(t *testing.T) {
			requirePrintedError(t, err, printedError{
				message:           "wrapped: " + tc.err.msg,
				printfPlusVRegexp: `^wrapped\n<TestWrapMultiCause>\n` + tc.plusVBody + `$`,
				printfHashVRegexp: `^errorsx\.wrappedError\{cause:errorsx\.multiCauseError\{.*\}, msg:"wrapped", stackTrace:.*}$`,
			})
		})
	}
}

func TestJoin(t *testing.T) {
	a := Errorf("a")
	b := WithMessage(sentinel, "b")

	joined := Join(a, nil, b)
	requirePrintedError(t, joined, printedError{
		message:           "a\nb: sentinel",
		printfPlusVRegexp: `^multiple errors:\n\ta\n\t<TestJoin>\nand:\n\tb\n\tsentinel$`,
		printfHashVRegexp: `^errorsx\.joinedError\{errs:\[\]error\{errorsx\.wrappedError\{.*\}, errorsx\.wrappedError\{.*\}\}\}$`,
	})

	// Nils are discarded, as with errors.Join.
	assert.Equal(t, []error{a, b}, joined.(interface{ Unwrap() []error }).Unwrap())
	assert.True(t, errors.Is(joined, sentinel))
	var target customError
	assert.True(t, errors.As(Join(a, WithMessage(customError{code: 42}, "c")), &target))
	assert.Equal(t, 42, target.code)

	// A joined error can be wrapped, and joined.
	requirePrintedError(t, Wrap(joined, "wrapped"), printedError{
		message:           "wrapped: a\nb: sentinel",
		printfPlusVRegexp: `^wrapped\n<TestJoin>\nmultiple errors:\n\ta\n\t<TestJoin>\nand:\n\tb\n\tsentinel$`,
		printfHashVRegexp: `^errorsx\.wrappedError\{cause:errorsx\.joinedError\{.*\}, msg:"wrapped", stackTrace:.*}$`,
	})
	requirePrintedError(t, Join(Wrap(sentinel, "c"), joined), printedError{
		message:           "c: sentinel\na\nb: sentinel",
		printfPlusVRegexp: `^multiple errors:\n\tc\n\t<TestJoin>\n\tsentinel\nand:\n\tmultiple errors:\n\t\ta\n\t\t<TestJoin>\n\tand:\n\t\tb\n\t\tsentinel$`,
		printfHashVRegexp: `^errorsx\.joinedError\{.*\}$`,
	})

	// A single error is printed as a simple chain.
	requirePrintedError(t, Join(nil, b), printedError{
		message:           "b: sentinel",
		printfPlusVRegexp: `^b\nsentinel$`,
		printfHashVRegexp: `^errorsx\.joinedError\{errs:\[\]error\{.*\}\}$`,
	})

	// Joining only nils gives nil.
	assert.Zero(t, Join())
	assert.Zero(t, Join(nil, nil))
}

// formatterError is an error that implements fmt.Formatter, but not
// in the way errorsx errors do: its output does not start with a
// newline.
type formatterError struct{}

func (formatterError) Error() string { return "formatter error" }

func (formatterError) Format(f fmt.State, verb rune) {
	fmt.Fprint(f, "formatter error")
}

func TestWrapFormatter(t *testing.T) {
	requirePrintedError(t, WithMessage(formatterError{}, "wrapped"), printedError{
		message:           "wrapped: formatter error",
		printfPlusVRegexp: `^wrapped\nformatter error$`,
		printfHashVRegexp: `^errorsx\.wrappedError\{cause:errorsx\.formatterError\{\}, msg:"wrapped", stackTrace:\[\]uintptr\(nil\)\}$`,
	})
	requirePrintedError(t, Wrap(formatterError{}, "wrapped"), printedError{
		message:           "wrapped: formatter error",
		printfPlusVRegexp: `^wrapped\n<TestWrapFormatter>\nformatter error$`,
		printfHashVRegexp: `^errorsx\.wrappedError\{cause:errorsx\.formatterError\{\}, msg:"wrapped", stackTrace:.*\}$`,
	})
}

func TestWrapThinWrappers(t *testing.T) {
	expect := printedError{
		message:           "wrapped: base error",
		printfPlusVRegexp: `^wrapped\n<TestWrapThinWrappers>\nbase error\n<TestWrapThinWrappers>$`,
		printfHashVRegexp: `^errorsx\.wrappedError\{cause:errorsx\.thinWrapper\{.*\}, msg:"wrapped", stackTrace:.*}$`,
	}
	requirePrintedError(t, Wrap(thinWrapper{Errorf("base error")}, "wrapped"), expect)
	requirePrintedError(t, Wrap(thinWrapper{thinWrapper{Errorf("base error")}}, "wrapped"), expect)

	expect.printfPlusVRegexp = `^wrapped\n<TestWrapThinWrappers>\nbase error$`
	requirePrintedError(t, Wrap(thinWrapper{fmt.Errorf("base error")}, "wrapped"), expect)
	requirePrintedError(t, Wrap(thinWrapper{thinWrapper{fmt.Errorf("base error")}}, "wrapped"), expect)
}

type thinWrapper struct {
	error
}

func (t thinWrapper) Unwrap() error {
	return t.error
}

func TestWrapf(t *testing.T) {
	baseErr := errors.New("base error")
	requirePrintedError(t, Wrapf(baseErr, "wrapped %s", "context"), printedError{
		message:           "wrapped context: base error",
		printfPlusVRegexp: `^wrapped context\n<TestWrapf>\nbase error$`,
		printfHashVRegexp: `^errorsx\.wrappedError\{cause:.*, msg:"wrapped context", stackTrace:.*}$`,
	})

	// Wrapping nil should return nil
	assert.Zero(t, Wrapf(nil, "%d", 123))
}

func TestWithMessage(t *testing.T) {
	// WithMessage wraps without capturing a stack trace
	baseErr := errors.New("base error")
	wrapped := WithMessage(baseErr, "with message")
	requirePrintedError(t, wrapped, printedError{
		message:           "with message: base error",
		printfPlusVRegexp: `^with message\nbase error$`,
		printfHashVRegexp: `^errorsx\.wrappedError\{cause:.*, msg:"with message", stackTrace:\[\]uintptr\(nil\)}$`,
	})
	assert.Equal(t, baseErr, errors.Unwrap(wrapped))

	// WithMessage on an errorsx error preserves the inner stack trace
	errWithStack := Errorf("inner error")
	wrapped = WithMessage(errWithStack, "outer message")
	requirePrintedError(t, wrapped, printedError{
		message:           "outer message: inner error",
		printfPlusVRegexp: `^outer message\ninner error\n<TestWithMessage>$`,
		printfHashVRegexp: `^errorsx\.wrappedError\{cause:.*, msg:"outer message", stackTrace:\[\]uintptr\(nil\)}$`,
	})

	// Wrapping nil should return nil
	assert.Zero(t, WithMessage(nil, "something"))
}

func TestWithMessagef(t *testing.T) {
	// WithMessagef wraps without capturing a stack trace
	baseErr := errors.New("base error")
	wrapped := WithMessagef(baseErr, "context %s %d", "foo", 42)
	requirePrintedError(t, wrapped, printedError{
		message:           "context foo 42: base error",
		printfPlusVRegexp: `^context foo 42\nbase error$`,
		printfHashVRegexp: `^errorsx\.wrappedError\{cause:.*, msg:"context foo 42", stackTrace:\[\]uintptr\(nil\)}$`,
	})
	assert.Equal(t, baseErr, errors.Unwrap(wrapped))

	// WithMessagef on an errorsx error preserves the inner stack trace
	errWithStack := Errorf("inner error")
	wrapped = WithMessagef(errWithStack, "outer %s", "message")
	requirePrintedError(t, wrapped, printedError{
		message:           "outer message: inner error",
		printfPlusVRegexp: `^outer message\ninner error\n<TestWithMessagef>$`,
		printfHashVRegexp: `^errorsx\.wrappedError\{cause:.*, msg:"outer message", stackTrace:\[\]uintptr\(nil\)}$`,
	})

	// Wrapping nil should return nil
	assert.Zero(t, WithMessagef(nil, "something %d", 123))
}

// sentinel is used to check errors.Is matching through the wrappers.
var sentinel = errors.New("sentinel")

// customError is used to check errors.As matching through the wrappers.
type customError struct {
	code int
}

func (customError) Error() string {
	return "custom error"
}

func TestErrorsIs(t *testing.T) {
	// Each of the wrapping functions should preserve errors.Is
	// matching against the wrapped error.
	wrappers := map[string]func(error) error{
		"Wrap":         func(err error) error { return Wrap(err, "wrapped") },
		"Wrapf":        func(err error) error { return Wrapf(err, "wrapped %d", 1) },
		"WithMessage":  func(err error) error { return WithMessage(err, "wrapped") },
		"WithMessagef": func(err error) error { return WithMessagef(err, "wrapped %d", 1) },
	}

	for name, wrap := range wrappers {
		t.Run(name, func(t *testing.T) {
			wrapped := wrap(sentinel)
			assert.True(t, errors.Is(wrapped, sentinel))
			assert.False(t, errors.Is(wrapped, errors.New("other")))

			// Matching should continue to work through several
			// layers of wrapping.
			assert.True(t, errors.Is(wrap(wrapped), sentinel))
		})
	}

	// errors.Is should find the sentinel through a wrapped
	// errors.Join too.
	joined := Wrap(errors.Join(errors.New("a"), sentinel), "wrapped")
	assert.True(t, errors.Is(joined, sentinel))
}

func TestErrorsAs(t *testing.T) {
	wrapped := Wrap(customError{code: 42}, "wrapped")

	var target customError
	assert.True(t, errors.As(wrapped, &target))
	assert.Equal(t, 42, target.code)

	// A chain with no customError should not match.
	var noMatch customError
	assert.False(t, errors.As(Wrap(errors.New("plain"), "wrapped"), &noMatch))
}
