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

package errdefs

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"
)

func TestErrorEquivalence(t *testing.T) {
	var e1 error = ErrAborted
	var e2 error = ErrUnknown
	if e1 == e2 {
		t.Fatal("should not equal the same error")
	}
	if errors.Is(e1, e2) {
		t.Fatal("errors.Is should not return true")
	}

	var e3 error = errAborted{}
	if e1 != e3 {
		t.Fatal("new instance should be equivalent")
	}
	if !errors.Is(e1, e3) {
		t.Fatal("errors.Is should be true")
	}
	if !errors.Is(e3, e1) {
		t.Fatal("errors.Is should be true")
	}
	var aborted errAborted
	if !errors.As(e1, &aborted) {
		t.Fatal("errors.As should be true")
	}

	var e4 = ErrAborted.WithMessage("custom message")
	if e1 == e4 {
		t.Fatal("should not equal the same error")
	}

	if !errors.Is(e4, e1) {
		t.Fatal("errors.Is should be true, e1 is in the tree of e4")
	}

	if errors.Is(e1, e4) {
		t.Fatal("errors.Is should be false, e1 is not a custom message")
	}

	if !errors.As(e4, &aborted) {
		t.Fatal("errors.As should be true")
	}

	var custom customMessage
	if !errors.As(e4, &custom) {
		t.Fatal("errors.As should be true")
	}
	if custom.msg != "custom message" {
		t.Fatalf("unexpected custom message: %q", custom.msg)
	}
	if custom.err != e1 {
		t.Fatalf("unexpected custom message error: %v", custom.err)
	}
}

func TestWithMessage(t *testing.T) {
	testErrors := []error{ErrUnknown,
		ErrInvalidArgument,
		ErrNotFound,
		ErrAlreadyExists,
		ErrPermissionDenied,
		ErrResourceExhausted,
		ErrFailedPrecondition,
		ErrConflict,
		ErrNotModified,
		ErrAborted,
		ErrOutOfRange,
		ErrNotImplemented,
		ErrInternal,
		ErrUnavailable,
		ErrDataLoss,
		ErrUnauthenticated,
	}
	for _, err := range testErrors {
		e1 := err
		t.Run(err.Error(), func(t *testing.T) {
			wm, ok := e1.(interface{ WithMessage(string) error })
			if !ok {
				t.Fatal("WithMessage not supported")
			}
			e2 := wm.WithMessage("custom message")

			if e1 == e2 {
				t.Fatal("should not equal the same error")
			}

			if !errors.Is(e2, e1) {
				t.Fatal("errors.Is should return true")
			}

			if errors.Is(e1, e2) {
				t.Fatal("errors.Is should be false, e1 is not a custom message")
			}

			var raw = reflect.New(reflect.TypeOf(e1)).Interface()
			if !errors.As(e2, raw) {
				t.Fatal("errors.As should be true")
			}

			var custom customMessage
			if !errors.As(e2, &custom) {
				t.Fatal("errors.As should be true")
			}
			if custom.msg != "custom message" {
				t.Fatalf("unexpected custom message: %q", custom.msg)
			}
			if custom.err != e1 {
				t.Fatalf("unexpected custom message error: %v", custom.err)
			}

		})
	}
}

func TestInterfaceMatch(t *testing.T) {
	testCases := []struct {
		err   error
		check func(error) bool
	}{
		{ErrUnknown, isInterface[unknown]},
		{ErrInvalidArgument, isInterface[invalidParameter]},
		{ErrNotFound, isInterface[notFound]},
		{ErrAlreadyExists, isInterface[alreadyExists]},
		{ErrPermissionDenied, isInterface[forbidden]},
		{ErrResourceExhausted, isInterface[resourceExhausted]},
		{ErrFailedPrecondition, isInterface[failedPrecondition]},
		{ErrConflict, isInterface[conflict]},
		{ErrNotModified, isInterface[notModified]},
		{ErrAborted, isInterface[aborted]},
		{ErrOutOfRange, isInterface[outOfRange]},
		{ErrNotImplemented, isInterface[notImplemented]},
		{ErrInternal, isInterface[system]},
		{ErrUnavailable, isInterface[unavailable]},
		{ErrDataLoss, isInterface[dataLoss]},
		{ErrUnauthenticated, isInterface[unauthorized]},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(fmt.Sprintf("%T", tc.err), func(t *testing.T) {
			if !tc.check(tc.err) {
				t.Fatal("Error does not match interface")
			}
		})
	}
}

// TestIsHelpers verifies that all IsXxx helpers:
//
//  1. Match the canonical errdefs sentinel (e.g., ErrNotFound).
//  2. Work through standard %w wrapping.
//  3. Recognize Moby-style interface-based errors (custom types implementing
//     methods such as NotFound(), Unauthorized(), etc., without coupling to
//     containerd/errdefs).
//  4. Work through errors.Join (multi-error unwrapping).
func TestIsHelpers(t *testing.T) {
	errOther := errors.New("errdefs test: other error")

	tests := []struct {
		doc       string
		is        func(error) bool
		sentinel  error
		customErr func() error
	}{
		{"IsCanceled", IsCanceled, context.Canceled, newCanceledErr},
		{"IsDeadlineExceeded", IsDeadlineExceeded, context.DeadlineExceeded, newDeadlineExceededErr},

		{"IsUnknown", IsUnknown, ErrUnknown, newUnknownErr},
		{"IsInvalidArgument", IsInvalidArgument, ErrInvalidArgument, newInvalidArgumentErr},
		{"IsNotFound", IsNotFound, ErrNotFound, newNotFoundErr},
		{"IsAlreadyExists", IsAlreadyExists, ErrAlreadyExists, newAlreadyExistsErr},
		{"IsPermissionDenied", IsPermissionDenied, ErrPermissionDenied, newPermissionDeniedErr},
		{"IsResourceExhausted", IsResourceExhausted, ErrResourceExhausted, newResourceExhaustedErr},
		{"IsFailedPrecondition", IsFailedPrecondition, ErrFailedPrecondition, newFailedPreconditionErr},
		{"IsConflict", IsConflict, ErrConflict, newConflictErr},
		{"IsNotModified", IsNotModified, ErrNotModified, newNotModifiedErr},
		{"IsAborted", IsAborted, ErrAborted, newAbortedErr},
		{"IsOutOfRange", IsOutOfRange, ErrOutOfRange, newOutOfRangeErr},
		{"IsNotImplemented", IsNotImplemented, ErrNotImplemented, newNotImplementedErr},
		{"IsInternal", IsInternal, ErrInternal, newInternalErr},
		{"IsUnavailable", IsUnavailable, ErrUnavailable, newUnavailableErr},
		{"IsDataLoss", IsDataLoss, ErrDataLoss, newDataLossErr},
		{"IsUnauthorized", IsUnauthorized, ErrUnauthenticated, newUnauthorizedErr},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.doc, func(t *testing.T) {
			customErr := tc.customErr()

			if tc.is(nil) {
				t.Error("expected false for nil")
			}

			// Sentinel
			if !tc.is(tc.sentinel) {
				t.Errorf("expected true for sentinel (%T)", tc.sentinel)
			}
			if !tc.is(fmt.Errorf("wrap: %w", tc.sentinel)) {
				t.Errorf("expected true for wrapped sentinel (%T)", tc.sentinel)
			}

			// Moby-style interface-based implementation
			if !tc.is(customErr) {
				t.Errorf("expected true for custom err (%T)", customErr)
			}
			if !tc.is(fmt.Errorf("wrap: %w", customErr)) {
				t.Errorf("expected true for wrapped custom err (%T)", customErr)
			}
			if !tc.is(errors.Join(errOther, customErr)) {
				t.Errorf("expected true for joined custom err (%T)", customErr)
			}

			// WithMessage (only for errdefs sentinels that implement it)
			if wm, ok := any(tc.sentinel).(interface{ WithMessage(string) error }); ok {
				if !tc.is(wm.WithMessage("custom msg")) {
					t.Errorf("expected true for WithMessage (%T)", tc.sentinel)
				}
			}

			// Negative control
			if tc.is(errOther) {
				t.Errorf("expected false for unrelated error")
			}
			if tc.is(errors.New(tc.sentinel.Error())) {
				t.Errorf("expected false for message-only match")
			}
		})
	}
}

func newCanceledErr() error { return &customCanceled{} }

type customCanceled struct{}

func (*customCanceled) Error() string { return "custom canceled" }
func (*customCanceled) Cancelled()    {}

func newDeadlineExceededErr() error { return &customDeadlineExceeded{} }

type customDeadlineExceeded struct{}

func (*customDeadlineExceeded) Error() string     { return "custom deadline" }
func (*customDeadlineExceeded) DeadlineExceeded() {}

func newUnknownErr() error { return &customUnknown{} }

type customUnknown struct{}

func (*customUnknown) Error() string { return "custom unknown" }
func (*customUnknown) Unknown()      {}

func newInvalidArgumentErr() error { return &customInvalidArgument{} }

type customInvalidArgument struct{}

func (*customInvalidArgument) Error() string     { return "custom invalid argument" }
func (*customInvalidArgument) InvalidParameter() {}

func newNotFoundErr() error { return &customNotFound{} }

type customNotFound struct{}

func (*customNotFound) Error() string { return "custom not found" }
func (*customNotFound) NotFound()     {}

func newAlreadyExistsErr() error { return &customAlreadyExists{} }

type customAlreadyExists struct{}

func (*customAlreadyExists) Error() string  { return "custom already exists" }
func (*customAlreadyExists) AlreadyExists() {}

func newPermissionDeniedErr() error { return &customPermissionDenied{} }

type customPermissionDenied struct{}

func (*customPermissionDenied) Error() string { return "custom permission denied" }
func (*customPermissionDenied) Forbidden()    {}

func newResourceExhaustedErr() error { return &customResourceExhausted{} }

type customResourceExhausted struct{}

func (*customResourceExhausted) Error() string      { return "custom resource exhausted" }
func (*customResourceExhausted) ResourceExhausted() {}

func newFailedPreconditionErr() error { return &customFailedPrecondition{} }

type customFailedPrecondition struct{}

func (*customFailedPrecondition) Error() string       { return "custom failed precondition" }
func (*customFailedPrecondition) FailedPrecondition() {}

func newConflictErr() error { return &customConflict{} }

type customConflict struct{}

func (*customConflict) Error() string { return "custom conflict" }
func (*customConflict) Conflict()     {}

func newNotModifiedErr() error { return &customNotModified{} }

type customNotModified struct{}

func (*customNotModified) Error() string { return "custom not modified" }
func (*customNotModified) NotModified()  {}

func newAbortedErr() error { return &customAborted{} }

type customAborted struct{}

func (*customAborted) Error() string { return "custom aborted" }
func (*customAborted) Aborted()      {}

func newOutOfRangeErr() error { return &customOutOfRange{} }

type customOutOfRange struct{}

func (*customOutOfRange) Error() string { return "custom out of range" }
func (*customOutOfRange) OutOfRange()   {}

func newNotImplementedErr() error { return &customNotImplemented{} }

type customNotImplemented struct{}

func (*customNotImplemented) Error() string   { return "custom not implemented" }
func (*customNotImplemented) NotImplemented() {}

func newInternalErr() error { return &customInternal{} }

type customInternal struct{}

func (*customInternal) Error() string { return "custom internal" }
func (*customInternal) System()       {}

func newUnavailableErr() error { return &customUnavailable{} }

type customUnavailable struct{}

func (*customUnavailable) Error() string { return "custom unavailable" }
func (*customUnavailable) Unavailable()  {}

func newDataLossErr() error { return &customDataLoss{} }

type customDataLoss struct{}

func (*customDataLoss) Error() string { return "custom data loss" }
func (*customDataLoss) DataLoss()     {}

func newUnauthorizedErr() error { return &customUnauthorized{} }

type customUnauthorized struct{}

func (*customUnauthorized) Error() string { return "custom unauthorized" }
func (*customUnauthorized) Unauthorized() {}
