package task_test

import (
	"context"
	"fmt"
	"time"

	"github.com/dpw/go-util/errorsx"
	"github.com/dpw/go-util/task"
)

func ExampleParallel() {
	// A server that runs until it is cancelled.
	serve := func(ctx context.Context) errorsx.Error {
		<-ctx.Done()
		fmt.Println("server stopped")
		return errorsx.Wrap(ctx.Err(), "cancelled")
	}

	// A client that exercises the server, then finishes.  When it
	// finishes, Parallel cancels the server.
	client := func(ctx context.Context) errorsx.Error {
		fmt.Println("client done")
		return nil
	}

	// Parallel returns the result of the first task to finish,
	// the client.
	err := task.Parallel(context.Background(), serve, client)
	fmt.Println("result:", err)

	// Output:
	// client done
	// server stopped
	// result: <nil>
}

func ExampleWait() {
	// A quick task that succeeds.  This does not affect the other
	// tasks.
	quick := func(ctx context.Context) errorsx.Error {
		return nil
	}

	// A slower task, which still gets to finish its work.
	slow := func(ctx context.Context) errorsx.Error {
		select {
		case <-time.After(10 * time.Millisecond):
			fmt.Println("slow task finished")
			return nil
		case <-ctx.Done():
			return errorsx.Wrap(ctx.Err(), "cancelled")
		}
	}

	err := task.Wait(context.Background(), quick, slow)
	fmt.Println("result:", err)

	// A failing task cancels the others, and its error is the
	// result.
	failing := func(ctx context.Context) errorsx.Error {
		return errorsx.Errorf("failed")
	}

	blocking := func(ctx context.Context) errorsx.Error {
		<-ctx.Done()
		return errorsx.Wrap(ctx.Err(), "cancelled")
	}

	err = task.Wait(context.Background(), failing, blocking)
	fmt.Println("result:", err)

	// Output:
	// slow task finished
	// result: <nil>
	// result: failed
}
