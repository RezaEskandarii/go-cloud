// Copyright 2018 The Go Cloud Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package retry provides retry logic.
package retry

import (
	"context"
	"fmt"
	"time"

	gax "github.com/googleapis/gax-go"
)

// Call calls the supplied function f repeatedly, using the isRetryable function and
// the provided backoff parameters to control the repetition.
//
// When f returns nil, Call immediately returns nil.
//
// When f returns an error for which isRetryable returns false, Call immediately
// returns that error.
//
// When f returns an error for which isRetryable returns true, Call sleeps for the
// duration provided by isRetryable, or the provided backoff value if that duration
// is zero, and invokes f again.
//
// When the provided context is done, Retry returns with an error that
// includes both ctx.Error() and the last error returned by f, if any.
func Call(ctx context.Context, bo gax.Backoff, isRetryable func(error) (bool, time.Duration), f func() error) error {
	return call(ctx, bo, isRetryable, f, gax.Sleep)
}

// Split out for testing.
func call(ctx context.Context, bo gax.Backoff, isRetryable func(error) (bool, time.Duration), f func() error,
	sleep func(context.Context, time.Duration) error) error {
	// Do nothing if context is done on entry.
	if err := ctx.Err(); err != nil {
		return err
	}
	for {
		err := f()
		if err == nil {
			return nil
		}
		retry, dur := isRetryable(err)
		if !retry {
			return err
		}
		if dur == 0 {
			dur = bo.Pause()
		}
		if cerr := sleep(ctx, dur); cerr != nil {
			return &ContextError{CtxErr: cerr, FuncErr: err}
		}
	}
}

// A ContextError contains both a context error (either context.Canceled or
// context.DeadlineExceeded), and the last error from the function being retried.
type ContextError struct {
	CtxErr  error // The error obtained from ctx.Err()
	FuncErr error // The error obtained from the function being retried
}

func (e *ContextError) Error() string {
	return fmt.Sprintf("%v; last error: %v", e.CtxErr, e.FuncErr)
}