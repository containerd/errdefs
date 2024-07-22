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

import "context"

// Resolve returns the first error found in the error chain which matches an
// error defined in this package or context error. A raw, unwrapped error is
// returned or ErrUnknown if no matching error is found.
//
// This is useful for determining a response code based on the outermost wrapped
// error rather than the original cause. For example, a not found error deep
// in the code may be wrapped as an invalid argument. When determining status
// code from Is* functions, the depth or ordering of the error is not
// considered.
//
// The search order is depth first, a wrapped error returned from any part of
// the chain from `Unwrap() error` will be returned before any joined errors
// as returned by `Unwrap() []error`.
func Resolve(err error) error {
	if err == nil {
		return nil
	}
	err = firstError(err)
	if err == nil {
		err = ErrUnknown
	}
	return err
}

func firstError(err error) error {
	for {
		switch err {
		case context.Canceled, context.DeadlineExceeded:
			return err
		}

		switch e := err.(type) {
		case Error:
			return e
		case unknown:
			return ErrUnknown
		case invalidParameter:
			return ErrInvalidArgument
		case notFound:
			return ErrNotFound
		// Skip ErrAlreadyExists, no interface defined
		case forbidden:
			return ErrPermissionDenied
		// Skip ErrResourceExhasuted, no interface defined
		// Skip ErrFailedPrecondition, no interface defined
		case conflict:
			return ErrConflict
		case notModified:
			return ErrNotModified
		// Skip ErrAborted, no interface defined
		// Skip ErrOutOfRange, no interface defined
		case notImplemented:
			return ErrNotImplemented
		case system:
			return ErrInternal
		case unavailable:
			return ErrUnavailable
		case dataLoss:
			return ErrDataLoss
		case unauthorized:
			return ErrUnauthenticated
		case deadlineExceeded:
			return context.DeadlineExceeded
		case cancelled:
			return context.Canceled
		case interface{ Unwrap() error }:
			err = e.Unwrap()
			if err == nil {
				return nil
			}
		case interface{ Unwrap() []error }:
			for _, ue := range e.Unwrap() {
				if fe := firstError(ue); fe != nil {
					return fe
				}
			}
			return nil
		default:
			return nil
		}
	}
}
