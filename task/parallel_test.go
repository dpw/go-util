package task

import (
	"context"
	"errors"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alecthomas/assert/v2"

	"github.com/dpw/go-util/errorsx"
)

// testTimeout bounds how long withTimeout waits, so that a bug causes
// a test failure rather than a hang.
const testTimeout = 5 * time.Second

// withTimeout runs body on a new goroutine and waits for it to
// terminate, failing the test if that takes longer than testTimeout.
// Termination by runtime.Goexit counts as terminating.
//
// body must not call t.FailNow (or the assert functions, which use
// it), because that is only allowed on the test's own goroutine.
// Instead, body should store any results in variables that the test
// checks after withTimeout returns.  If body times out, the test
// stops immediately, so those variables are never read while body
// might still be writing to them.
func withTimeout(t *testing.T, body func()) {
	t.Helper()
	done := make(chan struct{})
	go func() {
		defer close(done)
		body()
	}()
	select {
	case <-done:
	case <-time.After(testTimeout):
		t.Fatalf("timed out after %v", testTimeout)
	}
}

// parallelOrWaitFunc is the signature shared by Parallel and Wait.
type parallelOrWaitFunc func(context.Context, ...Task) errorsx.Error

// forParallelAndWait runs test as a subtest for each of Parallel and
// Wait.  It is used for behaviour that the two have in common.
func forParallelAndWait(t *testing.T, test func(t *testing.T, run parallelOrWaitFunc)) {
	t.Run("Parallel", func(t *testing.T) { test(t, Parallel) })
	t.Run("Wait", func(t *testing.T) { test(t, Wait) })
}

// A task that immediately succeeds.
func succeed(context.Context) errorsx.Error { return nil }

// blockUntilDone is a Task that blocks until its context is
// cancelled, then returns the context's error, wrapped.
func blockUntilDone(ctx context.Context) errorsx.Error {
	<-ctx.Done()
	return errorsx.Wrap(ctx.Err(), "cancelled")
}

// Behaviour common to Parallel and Wait.

func TestParallelAndWaitNoTasks(t *testing.T) {
	forParallelAndWait(t, func(t *testing.T, run parallelOrWaitFunc) {
		var err error
		withTimeout(t, func() { err = run(context.Background()) })
		assert.NoError(t, err)
	})
}

func TestParallelAndWaitNoTasksWithCancelledContext(t *testing.T) {
	// With no tasks, the result is success regardless of the context.
	forParallelAndWait(t, func(t *testing.T, run parallelOrWaitFunc) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		var err error
		withTimeout(t, func() { err = run(ctx) })
		assert.NoError(t, err)
	})
}

func TestParallelAndWaitAllTasksSucceed(t *testing.T) {
	forParallelAndWait(t, func(t *testing.T, run parallelOrWaitFunc) {
		const n = 10
		var calls atomic.Int32
		tasks := make([]Task, n)
		for i := range tasks {
			tasks[i] = func(context.Context) errorsx.Error {
				calls.Add(1)
				return nil
			}
		}

		var err error
		withTimeout(t, func() {
			err = run(context.Background(), tasks...)
		})
		assert.NoError(t, err)

		// All tasks are waited for, so all have run by now.
		assert.Equal(t, int32(n), calls.Load())
	})
}

func TestParallelAndWaitTasksRunConcurrently(t *testing.T) {
	// Each task waits until all tasks have started.  This can only
	// succeed if they really do run concurrently.
	forParallelAndWait(t, func(t *testing.T, run parallelOrWaitFunc) {
		const n = 8
		var barrier sync.WaitGroup
		barrier.Add(n)
		tasks := make([]Task, n)
		for i := range tasks {
			tasks[i] = func(context.Context) errorsx.Error {
				barrier.Done()
				barrier.Wait()
				return nil
			}
		}
		var err error
		withTimeout(t, func() {
			err = run(context.Background(), tasks...)
		})
		assert.NoError(t, err)
	})
}

func TestParallelAndWaitReturnsFirstError(t *testing.T) {
	forParallelAndWait(t, func(t *testing.T, run parallelOrWaitFunc) {
		want := errors.New("first failure")
		var err error
		withTimeout(t, func() {
			err = run(context.Background(),
				blockUntilDone,
				func(context.Context) errorsx.Error { return errorsx.Wrap(want, "task failed") },
				blockUntilDone,
			)
		})
		assert.IsError(t, err, want)
	})
}

func TestParallelAndWaitFirstErrorWinsOverLaterErrors(t *testing.T) {
	forParallelAndWait(t, func(t *testing.T, run parallelOrWaitFunc) {
		first := errors.New("first")
		second := errors.New("second")
		var err error
		withTimeout(t, func() {
			err = run(context.Background(),
				func(context.Context) errorsx.Error { return errorsx.Wrap(first, "task failed") },
				func(ctx context.Context) errorsx.Error {
					// Can only finish after being
					// cancelled, which only happens
					// once the first task has
					// returned.
					<-ctx.Done()
					return errorsx.Wrap(second, "task failed")
				},
			)
		})
		assert.IsError(t, err, first)
		assert.NotIsError(t, err, second)
	})
}

func TestParallelAndWaitCancelsOtherTasksOnError(t *testing.T) {
	// A task failing cancels the rest, via their own contexts:
	// the parent context is left alone.
	forParallelAndWait(t, func(t *testing.T, run parallelOrWaitFunc) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		want := errors.New("fail")
		var ctxErr error
		var err error
		withTimeout(t, func() {
			err = run(ctx,
				func(ctx context.Context) errorsx.Error {
					<-ctx.Done()
					ctxErr = ctx.Err()
					return nil
				},
				func(context.Context) errorsx.Error { return errorsx.Wrap(want, "task failed") },
			)
		})
		assert.IsError(t, err, want)
		// The cancelled task has been waited for, so ctxErr
		// is set.
		assert.IsError(t, ctxErr, context.Canceled)
		assert.NoError(t, ctx.Err())
	})
}

func TestParallelAndWaitPropagatesParentCancellation(t *testing.T) {
	// Cancelling the parent while the tasks are running cancels
	// them.
	forParallelAndWait(t, func(t *testing.T, run parallelOrWaitFunc) {
		ctx, cancel := context.WithCancel(context.Background())
		var err error
		withTimeout(t, func() {
			err = run(ctx,
				blockUntilDone,
				func(ctx context.Context) errorsx.Error {
					cancel()
					<-ctx.Done()
					return errorsx.Wrap(ctx.Err(), "cancelled")
				},
			)
		})
		assert.IsError(t, err, context.Canceled)
	})
}

func TestParallelAndWaitWithAlreadyCancelledContext(t *testing.T) {
	forParallelAndWait(t, func(t *testing.T, run parallelOrWaitFunc) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		var err error
		withTimeout(t, func() {
			err = run(ctx, blockUntilDone, blockUntilDone)
		})
		assert.IsError(t, err, context.Canceled)
	})
}

func TestParallelAndWaitWaitsForSlowCleanup(t *testing.T) {
	// Tasks that take a while to wind down after cancellation are
	// waited for, so their cleanup is complete on return.
	forParallelAndWait(t, func(t *testing.T, run parallelOrWaitFunc) {
		var cleanedUp atomic.Int32
		slowCleanup := func(ctx context.Context) errorsx.Error {
			<-ctx.Done()
			time.Sleep(20 * time.Millisecond)
			cleanedUp.Add(1)
			return nil
		}
		var err error
		withTimeout(t, func() {
			err = run(context.Background(),
				slowCleanup,
				func(context.Context) errorsx.Error { return errorsx.Errorf("fail") },
				slowCleanup,
			)
		})
		assert.Error(t, err)
		assert.Equal(t, int32(2), cleanedUp.Load())
	})
}

func TestParallelAndWaitGoexitReturnsError(t *testing.T) {
	// A task terminated by runtime.Goexit is reported as an error,
	// and cancels the other tasks like any other failure.  (If the
	// blocking task were not cancelled, the call would not
	// return.)
	forParallelAndWait(t, func(t *testing.T, run parallelOrWaitFunc) {
		var err error
		withTimeout(t, func() {
			err = run(context.Background(),
				func(context.Context) errorsx.Error {
					runtime.Goexit()
					return nil
				},
				blockUntilDone,
			)
		})
		assert.EqualError(t, err, "goroutine was terminated by runtime.Goexit")
	})
}

func TestParallelAndWaitGoexitAfterFirstResultIsIgnored(t *testing.T) {
	// A task that calls Goexit after another task has already
	// failed must neither change the result nor block forever.
	forParallelAndWait(t, func(t *testing.T, run parallelOrWaitFunc) {
		want := errors.New("fail")
		var err error
		withTimeout(t, func() {
			err = run(context.Background(),
				func(ctx context.Context) errorsx.Error {
					<-ctx.Done()
					runtime.Goexit()
					return nil
				},
				func(context.Context) errorsx.Error { return errorsx.Wrap(want, "task failed") },
			)
		})
		assert.IsError(t, err, want)
	})
}

func TestParallelAndWaitNested(t *testing.T) {
	forParallelAndWait(t, func(t *testing.T, run parallelOrWaitFunc) {
		want := errors.New("inner failure")
		var err error
		withTimeout(t, func() {
			err = run(context.Background(),
				func(ctx context.Context) errorsx.Error {
					return run(ctx,
						blockUntilDone,
						func(context.Context) errorsx.Error { return errorsx.Wrap(want, "task failed") },
					)
				},
				blockUntilDone,
			)
		})
		assert.IsError(t, err, want)
	})
}

func TestParallelAndWaitNestedOuterCancellationReachesInner(t *testing.T) {
	forParallelAndWait(t, func(t *testing.T, run parallelOrWaitFunc) {
		want := errors.New("outer failure")
		var err error
		withTimeout(t, func() {
			err = run(context.Background(),
				func(ctx context.Context) errorsx.Error {
					return run(ctx, blockUntilDone, blockUntilDone)
				},
				func(context.Context) errorsx.Error { return errorsx.Wrap(want, "task failed") },
			)
		})
		assert.IsError(t, err, want)
	})
}

// Behaviour specific to Parallel: any task terminating, even
// successfully, determines the result and cancels the other tasks.

func TestParallelCancelsOtherTasksOnSuccess(t *testing.T) {
	var err error
	withTimeout(t, func() {
		err = Parallel(context.Background(), blockUntilDone, succeed)
	})
	assert.NoError(t, err)
}

func TestParallelFirstResultWinsOverLaterErrors(t *testing.T) {
	// The first task to finish succeeds.  The others return an
	// error once cancelled.  Those later errors must be ignored.
	lateErr := errors.New("late error")
	late := func(ctx context.Context) errorsx.Error {
		<-ctx.Done()
		return errorsx.Wrap(lateErr, "task failed")
	}
	var err error
	withTimeout(t, func() {
		err = Parallel(context.Background(), late, succeed, late)
	})
	assert.NoError(t, err)
}

// Behaviour specific to Wait: a task succeeding has no effect on the
// other tasks or on the result.

func TestWaitSuccessDoesNotCancelOtherTasks(t *testing.T) {
	// One task succeeds immediately.  The other gives that time to
	// take effect, then fails, unless it was cancelled first.  Its
	// error must be the result.
	want := errors.New("fail after success")
	var err error
	withTimeout(t, func() {
		err = Wait(context.Background(),
			succeed,
			func(ctx context.Context) errorsx.Error {
				select {
				case <-ctx.Done():
					return errorsx.Wrap(ctx.Err(), "cancelled")
				case <-time.After(50 * time.Millisecond):
					return errorsx.Wrap(want, "task failed")
				}
			},
		)
	})
	assert.IsError(t, err, want)
}
