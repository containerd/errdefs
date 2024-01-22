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
	"fmt"
	"strconv"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ToGRPC will attempt to map the backend containerd error into a grpc error,
// using the original error message as a description.
//
// Further information may be extracted from certain errors depending on their
// type.
//
// If the error is unmapped, the original error will be returned to be handled
// by the regular grpc error handling stack.
func ToGRPC(err error) error {
	if err == nil {
		return nil
	}

	if isGRPCError(err) {
		// error has already been mapped to grpc
		return err
	}

	switch {
	case IsInvalidArgument(err):
		return status.Error(codes.InvalidArgument, err.Error())
	case IsNotFound(err):
		return status.Error(codes.NotFound, err.Error())
	case IsAlreadyExists(err):
		return status.Error(codes.AlreadyExists, err.Error())
	case IsFailedPrecondition(err) || IsConflict(err) || IsNotModified(err):
		return status.Error(codes.FailedPrecondition, err.Error())
	case IsUnavailable(err):
		return status.Error(codes.Unavailable, err.Error())
	case IsNotImplemented(err):
		return status.Error(codes.Unimplemented, err.Error())
	case IsCanceled(err):
		return status.Error(codes.Canceled, err.Error())
	case IsDeadlineExceeded(err):
		return status.Error(codes.DeadlineExceeded, err.Error())
	case IsUnauthorized(err):
		return status.Error(codes.Unauthenticated, err.Error())
	case IsPermissionDenied(err):
		return status.Error(codes.PermissionDenied, err.Error())
	case IsInternal(err):
		return status.Error(codes.Internal, err.Error())
	case IsDataLoss(err):
		return status.Error(codes.DataLoss, err.Error())
	case IsAborted(err):
		return status.Error(codes.Aborted, err.Error())
	case IsOutOfRange(err):
		return status.Error(codes.OutOfRange, err.Error())
	case IsResourceExhausted(err):
		return status.Error(codes.ResourceExhausted, err.Error())
	case IsUnknown(err):
		return status.Error(codes.Unknown, err.Error())
	}

	return err
}

// ToGRPCf maps the error to grpc error codes, assembling the formatting string
// and combining it with the target error string.
//
// This is equivalent to errdefs.ToGRPC(fmt.Errorf("%s: %w", fmt.Sprintf(format, args...), err))
func ToGRPCf(err error, format string, args ...interface{}) error {
	return ToGRPC(fmt.Errorf("%s: %w", fmt.Sprintf(format, args...), err))
}

// FromGRPC returns the underlying error from a grpc service based on the grpc error code
func FromGRPC(err error) error {
	if err == nil {
		return nil
	}

	desc := errDesc(err)

	var cls error // divide these into error classes, becomes the cause

	switch code(err) {
	case codes.InvalidArgument:
		cls = ErrInvalidArgument
	case codes.AlreadyExists:
		cls = ErrAlreadyExists
	case codes.NotFound:
		cls = ErrNotFound
	case codes.Unavailable:
		cls = ErrUnavailable
	case codes.FailedPrecondition:
		if desc == ErrConflict.Error() || strings.HasSuffix(desc, ": "+ErrConflict.Error()) {
			cls = ErrConflict
		} else if desc == ErrNotModified.Error() || strings.HasSuffix(desc, ": "+ErrNotModified.Error()) {
			cls = ErrNotModified
		} else {
			cls = ErrFailedPrecondition
		}
	case codes.Unimplemented:
		cls = ErrNotImplemented
	case codes.Canceled:
		cls = context.Canceled
	case codes.DeadlineExceeded:
		cls = context.DeadlineExceeded
	case codes.Aborted:
		cls = ErrAborted
	case codes.Unauthenticated:
		cls = ErrUnauthenticated
	case codes.PermissionDenied:
		cls = ErrPermissionDenied
	case codes.Internal:
		cls = ErrInternal
	case codes.DataLoss:
		cls = ErrDataLoss
	case codes.OutOfRange:
		cls = ErrOutOfRange
	case codes.ResourceExhausted:
		cls = ErrResourceExhausted
	default:
		if idx := strings.LastIndex(desc, unexpectedStatusPrefix); idx > 0 {
			if status, err := strconv.Atoi(desc[idx+len(unexpectedStatusPrefix):]); err == nil && status >= 200 && status < 600 {
				cls = errUnexpectedStatus{status}
			}
		}
		if cls == nil {
			cls = ErrUnknown
		}
	}

	msg := rebaseMessage(cls, desc)
	if msg != "" {
		err = fmt.Errorf("%s: %w", msg, cls)
	} else {
		err = cls
	}

	return err
}

// rebaseMessage removes the repeats for an error at the end of an error
// string. This will happen when taking an error over grpc then remapping it.
//
// Effectively, we just remove the string of cls from the end of err if it
// appears there.
func rebaseMessage(cls error, desc string) string {
	clss := cls.Error()
	if desc == clss {
		return ""
	}

	return strings.TrimSuffix(desc, ": "+clss)
}

func isGRPCError(err error) bool {
	_, ok := status.FromError(err)
	return ok
}

func code(err error) codes.Code {
	if s, ok := status.FromError(err); ok {
		return s.Code()
	}
	return codes.Unknown
}

func errDesc(err error) string {
	if s, ok := status.FromError(err); ok {
		return s.Message()
	}
	return err.Error()
}
