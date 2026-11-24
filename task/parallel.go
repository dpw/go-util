package task

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/dpw/go-util/errorsx"
)

// Parallel runs multiple Tasks in parallel.  When any of the tasks
// terminates, the other tasks are cancelled (by cancelling the
// context they receive).  Once all tasks have terminated, the value
// returned is that from the first task to return (which may be nil).
//
// This is useful to express strict parallelism, when a group of tasks
// collaborate to achieve some purpose.
//
// Each task runs on its own goroutine.  The tasks are cancelled
// through a context derived from ctx; ctx itself is never cancelled.
// With no tasks, Parallel returns nil immediately, even if ctx has
// already been cancelled.  A task that terminates by calling
// [runtime.Goexit] is treated as having returned an error.  A panic
// in a task is not recovered, so it crashes the process as usual,
// rather than being returned to the caller.
func Parallel(ctx context.Context, tasks ...Task) errorsx.Error {
	return parallel(ctx, true, tasks)
}

// Wait runs multiple Tasks in parallel, and waits for all of them to
// terminate.  A task returning nil has no effect on the other tasks.
// But when any task returns an error, the other tasks are cancelled
// (by cancelling the context they receive).  Once all tasks have
// terminated, the value returned is the error from the first task to
// fail, or nil if all tasks succeeded.
//
// This is useful when a group of tasks are independent, but the
// failure of any one of them means the work of the others is no
// longer needed.
//
// Each task runs on its own goroutine.  The tasks are cancelled
// through a context derived from ctx; ctx itself is never cancelled.
// With no tasks, Wait returns nil immediately, even if ctx has
// already been cancelled.  A task that terminates by calling
// [runtime.Goexit] is treated as having returned an error.  A panic
// in a task is not recovered, so it crashes the process as usual,
// rather than being returned to the caller.
func Wait(ctx context.Context, tasks ...Task) errorsx.Error {
	return parallel(ctx, false, tasks)
}

func parallel(ctx context.Context, strict bool, tasks []Task) (finalErr errorsx.Error) {
	if len(tasks) == 0 {
		// No tasks to run, indicate immediate success.
		return nil
	}

	// There is deliberately no special case for a single task.
	// If we just ran a single task directly, without a goroutine,
	// it would lead to some inconsistencies in behaviour between
	// the single task and multi-task cases (e.g. around
	// runtime.Goexit).

	pctx, cancel := context.WithCancel(ctx)
	var finalErrSet atomic.Bool
	var wg sync.WaitGroup

	// Start each task.
	for _, task := range tasks {
		wg.Go(func() {
			var normalTermination bool
			var err errorsx.Error

			defer func() {
				if !normalTermination {
					// We can get here for two reasons:
					//
					// - The task yielded a panic.
					// That panic will be
					// propagated out of the
					// goroutine and cause the
					// process to exit, so it
					// doesn't much matter what we
					// set finalErr to.
					//
					// - runtime.Goexit was
					// called, terminating the
					// goroutine.  We need to
					// turn this into some kind of
					// error.
					err = errorsx.Errorf("goroutine was terminated by runtime.Goexit")
				}

				// Record the result of the first task
				// to complete.
				if (strict || err != nil) && !finalErrSet.Swap(true) {
					finalErr = err
					cancel()
				}
			}()

			err = task(pctx)
			normalTermination = true
		})
	}

	// Wait for all tasks to terminate.  Only the first result was
	// recorded; the rest are ignored, even if the first result is
	// nil and later results are non-nil, because the cancellation
	// might lead to spurious errors.
	wg.Wait()

	// Ensure that the cancellation function is always called.
	cancel()
	return finalErr
}
