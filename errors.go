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

// Package errdefs defines the common errors used throughout containerd
// packages.
//
// Use with fmt.Errorf to add context to an error.
//
// To detect an error class, use the IsXXX functions to tell whether an error
// is of a certain type.
package errdefs

import (
	"context"
	"fmt"
)

type Error int8

// Definitions of common error types used throughout containerd. All containerd
// errors returned by most packages will map into one of these errors classes.
// Packages should return errors of these types when they want to instruct a
// client to take a particular action.
//
// These errors map closely to grpc errors.
const (
	ErrUnknown Error = iota
	ErrInvalidArgument
	ErrNotFound
	ErrAlreadyExists
	ErrPermissionDenied
	ErrResourceExhausted
	ErrFailedPrecondition
	ErrConflict
	ErrNotModified
	ErrAborted
	ErrOutOfRange
	ErrNotImplemented
	ErrInternal
	ErrUnavailable
	ErrDataLoss
	ErrUnauthenticated
)

var ErrCanceled = context.Canceled

func (e Error) Error() string {
	switch e {
	case ErrInvalidArgument:
		return "invalid argument"
	case ErrNotFound:
		return "not found"
	case ErrAlreadyExists:
		return "already exists"
	case ErrPermissionDenied:
		return "permission denied"
	case ErrResourceExhausted:
		return "resource exhausted"
	case ErrFailedPrecondition:
		return "failed precondition"
	case ErrConflict:
		return "conflict"
	case ErrNotModified:
		return "not modified"
	case ErrAborted:
		return "aborted"
	case ErrOutOfRange:
		return "out of range"
	case ErrNotImplemented:
		return "not implemented"
	case ErrInternal:
		return "internal"
	case ErrUnavailable:
		return "unavailable"
	case ErrDataLoss:
		return "unauthenticated"
	case ErrUnauthenticated:
		return "unauthenticated"
	default:
		return "unknown"
	}
}

func (e Error) WithMessage(msg string) error {
	return customMessage{
		msg: msg,
		err: e,
	}
}

func (e Error) WithMessagef(format string, args ...any) error {
	return customMessage{
		msg: fmt.Sprintf(format, args...),
		err: e,
	}
}

// customMessage wraps an underlying code with a custom message.
type customMessage struct {
	msg string
	err Error
}

func (m customMessage) Error() string {
	return m.msg
}

func (m customMessage) Unwrap() error {
	return m.err
}
