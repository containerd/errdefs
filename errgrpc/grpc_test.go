/*
   Copyright The containerd Authors.

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package errgrpc

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/containerd/errdefs"
	"github.com/containerd/errdefs/errhttp"
	"github.com/containerd/errdefs/internal/cause"
)

func TestGRPCNilInput(t *testing.T) {
	if err := ToGRPC(nil); err != nil {
		t.Fatalf("Expected nil error, got %v", err)
	}
	if err := ToNative(nil); err != nil {
		t.Fatalf("Expected nil error, got %v", err)
	}
}

func TestGRPCRoundTrip(t *testing.T) {
	errShouldLeaveAlone := errors.New("unknown to package")

	for _, testcase := range []struct {
		input error
		cause error
		str   string
	}{
		{
			input: errdefs.ErrInvalidArgument,
			cause: errdefs.ErrInvalidArgument,
		},
		{
			input: errdefs.ErrAlreadyExists,
			cause: errdefs.ErrAlreadyExists,
		},
		{
			input: errdefs.ErrNotFound,
			cause: errdefs.ErrNotFound,
		},
		{
			input: errdefs.ErrUnavailable,
			cause: errdefs.ErrUnavailable,
		},
		{
			input: errdefs.ErrNotImplemented,
			cause: errdefs.ErrNotImplemented,
		},
		{
			input: errdefs.ErrUnauthenticated,
			cause: errdefs.ErrUnauthenticated,
		},
		{
			input: errdefs.ErrPermissionDenied,
			cause: errdefs.ErrPermissionDenied,
		},
		{
			input: errdefs.ErrInternal,
			cause: errdefs.ErrInternal,
		},
		{
			input: errdefs.ErrDataLoss,
			cause: errdefs.ErrDataLoss,
		},
		{
			input: errdefs.ErrAborted,
			cause: errdefs.ErrAborted,
		},
		{
			input: errdefs.ErrOutOfRange,
			cause: errdefs.ErrOutOfRange,
		},
		{
			input: errdefs.ErrResourceExhausted,
			cause: errdefs.ErrResourceExhausted,
		},
		{
			input: errdefs.ErrUnknown,
			cause: errdefs.ErrUnknown,
		},
		//nolint:dupword
		{
			input: fmt.Errorf("test test test: %w", errdefs.ErrFailedPrecondition),
			cause: errdefs.ErrFailedPrecondition,
			str:   "test test test: failed precondition",
		},
		{
			input: status.Errorf(codes.Unavailable, "should be not available"),
			cause: errdefs.ErrUnavailable,
			str:   "should be not available: unavailable",
		},
		{
			input: errShouldLeaveAlone,
			cause: errdefs.ErrUnknown,
			str:   errShouldLeaveAlone.Error() + ": " + errdefs.ErrUnknown.Error(),
		},
		{
			input: context.Canceled,
			cause: context.Canceled,
			str:   "context canceled",
		},
		{
			input: fmt.Errorf("this is a test cancel: %w", context.Canceled),
			cause: context.Canceled,
			str:   "this is a test cancel: context canceled",
		},
		{
			input: context.DeadlineExceeded,
			cause: context.DeadlineExceeded,
			str:   "context deadline exceeded",
		},
		{
			input: fmt.Errorf("this is a test deadline exceeded: %w", context.DeadlineExceeded),
			cause: context.DeadlineExceeded,
			str:   "this is a test deadline exceeded: context deadline exceeded",
		},
		{
			input: fmt.Errorf("something conflicted: %w", errdefs.ErrConflict),
			cause: errdefs.ErrConflict,
			str:   "something conflicted: conflict",
		},
		{
			input: fmt.Errorf("everything is the same: %w", errdefs.ErrNotModified),
			cause: errdefs.ErrNotModified,
			str:   "everything is the same: not modified",
		},
		{
			input: fmt.Errorf("odd HTTP response: %w", errhttp.ToNative(418)),
			cause: cause.ErrUnexpectedStatus{Status: 418},
			str:   "odd HTTP response: unexpected status 418",
		},
	} {
		t.Run(testcase.input.Error(), func(t *testing.T) {
			t.Logf("input: %v", testcase.input)
			gerr := ToGRPC(testcase.input)
			t.Logf("grpc: %v", gerr)
			ferr := ToNative(gerr)
			t.Logf("recovered: %v", ferr)

			if !errors.Is(ferr, testcase.cause) {
				t.Fatalf("unexpected cause: !errors.Is(%v, %v)", ferr, testcase.cause)
			}

			expected := testcase.str
			if expected == "" {
				expected = testcase.cause.Error()
			}
			if ferr.Error() != expected {
				t.Fatalf("unexpected string: %q != %q", ferr.Error(), expected)
			}
		})
	}
}

func TestGRPC(t *testing.T) {
	err := fmt.Errorf("my error: %w", errdefs.ErrAborted)
	gerr := ToGRPC(err)
	st, _ := status.FromError(gerr)
	p := st.Proto()
	fmt.Printf("%d\n", len(p.Details))
	ferr := FromGRPC(gerr)

	for unvisited := []error{ferr}; len(unvisited) > 0; {
		cur := unvisited[0]
		unvisited = unvisited[1:]
		fmt.Printf("%s %T\n", cur.Error(), cur)
		switch cur := cur.(type) {
		case interface{ Unwrap() error }:
			if v := cur.Unwrap(); v != nil {
				unvisited = append(unvisited, v)
			}
		case interface{ Unwrap() []error }:
			unvisited = append(unvisited, cur.Unwrap()...)
		}
	}
}
