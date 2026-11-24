// Package task provides general functionality for working with
// tasks: activities that may take some time to execute, with their
// lifetimes controlled by a [context.Context], and which may succeed
// or fail.
package task

import (
	"context"

	"github.com/dpw/go-util/errorsx"
)

// A Task is a function representing an activity that may take some
// time to execute.  It should terminate promptly once its context is
// cancelled.  It returns nil to indicate success, or an error to
// indicate failure.
//
// A Task returns an [errorsx.Error] rather than a plain error, so
// that failures carry stack traces.  A function returning a plain
// error can be adapted by wrapping its result with [errorsx.Wrap].
type Task func(context.Context) errorsx.Error
