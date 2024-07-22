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
)

// IsCanceled returns true if the error is due to `context.Canceled`.
func IsCanceled(err error) bool {
	return errors.Is(err, context.Canceled) || isInterface[cancelled](err)
}

// IsUnknown returns true if the error is due to an unknown error,
// unhandled condition or unexpected response.
func IsUnknown(err error) bool {
	return errors.Is(err, ErrUnknown) || isInterface[unknown](err)
}

// IsInvalidArgument returns true if the error is due to an invalid argument
func IsInvalidArgument(err error) bool {
	return errors.Is(err, ErrInvalidArgument) || isInterface[invalidParameter](err)
}

// IsDeadlineExceeded returns true if the error is due to
// `context.DeadlineExceeded`.
func IsDeadlineExceeded(err error) bool {
	return errors.Is(err, context.DeadlineExceeded) || isInterface[deadlineExceeded](err)
}

// IsNotFound returns true if the error is due to a missing object
func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound) || isInterface[notFound](err)
}

// IsAlreadyExists returns true if the error is due to an already existing
// metadata item
func IsAlreadyExists(err error) bool {
	return errors.Is(err, ErrAlreadyExists)
}

// IsPermissionDenied returns true if the error is due to permission denied
// or forbidden (403) response
func IsPermissionDenied(err error) bool {
	return errors.Is(err, ErrPermissionDenied) || isInterface[forbidden](err)
}

// IsResourceExhausted returns true if the error is due to
// a lack of resources or too many attempts.
func IsResourceExhausted(err error) bool {
	return errors.Is(err, ErrResourceExhausted)
}

// IsFailedPrecondition returns true if an operation could not proceed due to
// the lack of a particular condition
func IsFailedPrecondition(err error) bool {
	return errors.Is(err, ErrFailedPrecondition)
}

// IsConflict returns true if an operation could not proceed due to
// a conflict.
func IsConflict(err error) bool {
	return errors.Is(err, ErrConflict) || isInterface[conflict](err)
}

// IsNotModified returns true if an operation could not proceed due
// to an object not modified from a previous state.
func IsNotModified(err error) bool {
	return errors.Is(err, ErrNotModified) || isInterface[notModified](err)
}

// IsAborted returns true if an operation was aborted.
func IsAborted(err error) bool {
	return errors.Is(err, ErrAborted)
}

// IsOutOfRange returns true if an operation could not proceed due
// to data being out of the expected range.
func IsOutOfRange(err error) bool {
	return errors.Is(err, ErrOutOfRange)
}

// IsNotImplemented returns true if the error is due to not being implemented
func IsNotImplemented(err error) bool {
	return errors.Is(err, ErrNotImplemented) || isInterface[notImplemented](err)
}

// IsInternal returns true if the error returns to an internal or system error
func IsInternal(err error) bool {
	return errors.Is(err, ErrInternal) || isInterface[system](err)
}

// IsUnavailable returns true if the error is due to a resource being unavailable
func IsUnavailable(err error) bool {
	return errors.Is(err, ErrUnavailable) || isInterface[unavailable](err)
}

// IsDataLoss returns true if data during an operation was lost or corrupted
func IsDataLoss(err error) bool {
	return errors.Is(err, ErrDataLoss) || isInterface[dataLoss](err)
}

// IsUnauthorized returns true if the error indicates that the user was
// unauthenticated or unauthorized.
func IsUnauthorized(err error) bool {
	// Intentional change. Old name was Unauthorized, but the grpc error
	// code is named Unauthenticated. The name is changing to Unauthenticated
	// but the old function name was IsUnauthorized.
	return errors.Is(err, ErrUnauthenticated) || isInterface[unauthorized](err)
}

// cancelled maps to Moby's "ErrCancelled"
type cancelled interface {
	Cancelled()
}

// unknown maps to Moby's "ErrUnknown"
type unknown interface {
	Unknown()
}

// invalidParameter maps to Moby's "ErrInvalidParameter"
type invalidParameter interface {
	InvalidParameter()
}

// deadlineExceed maps to Moby's "ErrDeadline"
type deadlineExceeded interface {
	DeadlineExceeded()
}

// notFound maps to Moby's "ErrNotFound"
type notFound interface {
	NotFound()
}

// forbidden maps to Moby's "ErrForbidden"
type forbidden interface {
	Forbidden()
}

// conflict maps to Moby's "ErrConflict"
type conflict interface {
	Conflict()
}

// notModified maps to Moby's "ErrNotModified"
type notModified interface {
	NotModified()
}

// notImplemented maps to Moby's "ErrNotImplemented"
type notImplemented interface {
	NotImplemented()
}

// system maps to Moby's "ErrSystem"
type system interface {
	System()
}

// unavailable maps to Moby's "ErrUnavailable"
type unavailable interface {
	Unavailable()
}

// dataLoss maps to Moby's "ErrDataLoss"
type dataLoss interface {
	DataLoss()
}

// unauthorized maps to Moby's "ErrUnauthorized"
type unauthorized interface {
	Unauthorized()
}

func isInterface[T any](err error) bool {
	for {
		switch x := err.(type) {
		case T:
			return true
		case interface{ Unwrap() error }:
			err = x.Unwrap()
			if err == nil {
				return false
			}
		case interface{ Unwrap() []error }:
			for _, err := range x.Unwrap() {
				if isInterface[T](err) {
					return true
				}
			}
			return false
		default:
			return false
		}
	}
}
