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

// Package errgrpc provides utility functions for translating errors to
// and from a gRPC context.
//
// The functions ToGRPC and ToNative can be used to map server-side and
// client-side errors to the correct types.
package errgrpc

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/containerd/typeurl/v2"
	spb "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/containerd/errdefs"
	"github.com/containerd/errdefs/internal/cause"
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
	if err == nil || isGRPCError(err) {
		return err
	}

	p := &spb.Status{
		Code:    int32(errorCode(err)),
		Message: err.Error(),
	}
	withDetails(p, err)
	return status.FromProto(p).Err()
}

func withDetails(p *spb.Status, err error) {
	any, _ := anypb.New(p)
	if any == nil {
		// If we fail to marshal the details, then use a generic
		// error by setting this as the empty struct.
		any, _ = anypb.New(&emptypb.Empty{})
	}

	if any == nil {
		// Extra protection just in case the above fails for
		// some reason.
		return
	}

	// First detail is a serialization of the current error.
	p.Details = append(p.Details, &anypb.Any{
		TypeUrl: any.GetTypeUrl(),
		Value:   any.GetValue(),
	})

	// Any remaining details are wrapped errors. We check
	// both versions of Unwrap to get this correct.
	var errs []error
	switch err := err.(type) {
	case interface{ Unwrap() error }:
		if unwrapped := err.Unwrap(); unwrapped != nil {
			errs = []error{unwrapped}
		}
	case interface{ Unwrap() []error }:
		errs = err.Unwrap()
	}

	for _, err := range errs {
		detail := &spb.Status{
			// Code doesn't matter. We don't use it beyond the top level.
			// Set to unknown just in case it leaks somehow.
			Code:    int32(codes.Unknown),
			Message: err.Error(),
		}
		withDetails(detail, err)

		if any, err := anypb.New(detail); err == nil {
			p.Details = append(p.Details, any)
		}
	}
}

func errorCode(err error) codes.Code {
	switch err := errdefs.Resolve(err); {
	case errdefs.IsInvalidArgument(err):
		return codes.InvalidArgument
	case errdefs.IsNotFound(err):
		return codes.NotFound
	case errdefs.IsAlreadyExists(err):
		return codes.AlreadyExists
	case errdefs.IsFailedPrecondition(err):
		fallthrough
	case errdefs.IsConflict(err):
		fallthrough
	case errdefs.IsNotModified(err):
		return codes.FailedPrecondition
	case errdefs.IsUnavailable(err):
		return codes.Unavailable
	case errdefs.IsNotImplemented(err):
		return codes.Unimplemented
	case errdefs.IsCanceled(err):
		return codes.Canceled
	case errdefs.IsDeadlineExceeded(err):
		return codes.DeadlineExceeded
	case errdefs.IsUnauthorized(err):
		return codes.Unauthenticated
	case errdefs.IsPermissionDenied(err):
		return codes.PermissionDenied
	case errdefs.IsInternal(err):
		return codes.Internal
	case errdefs.IsDataLoss(err):
		return codes.DataLoss
	case errdefs.IsAborted(err):
		return codes.Aborted
	case errdefs.IsOutOfRange(err):
		return codes.OutOfRange
	case errdefs.IsResourceExhausted(err):
		return codes.ResourceExhausted
	default:
		return codes.Unknown
	}
}

// ToGRPCf maps the error to grpc error codes, assembling the formatting string
// and combining it with the target error string.
//
// This is equivalent to grpc.ToGRPC(fmt.Errorf("%s: %w", fmt.Sprintf(format, args...), err))
func ToGRPCf(err error, format string, args ...interface{}) error {
	return ToGRPC(fmt.Errorf("%s: %w", fmt.Sprintf(format, args...), err))
}

func FromGRPC(err error) error {
	if err == nil {
		return nil
	}

	st, ok := status.FromError(err)
	if !ok {
		return err
	}

	p := st.Proto()
	return fromGRPCProto(p)
}

func fromGRPCProto(p *spb.Status) error {
	err := errors.New(p.Message)
	if len(p.Details) == 0 {
		return err
	}

	// First detail has the serialization.
	detail := p.Details[0]
	v, terr := typeurl.UnmarshalAny(detail)
	if terr == nil {
		if verr, ok := v.(error); ok {
			// Successfully unmarshaled the type as an error.
			// Use this instead.
			err = verr
		}
	}

	// If there is more than one detail, attempt to unmarshal
	// each one of them.
	if len(p.Details) > 1 {
		wrapped := make([]error, 0, len(p.Details)-1)
		for _, detail := range p.Details[1:] {
			p, derr := typeurl.UnmarshalAny(detail)
			if derr != nil {
				continue
			}

			switch p := p.(type) {
			case *spb.Status:
				wrapped = append(wrapped, fromGRPCProto(p))
			}
		}
		err = wrap(err, wrapped)
	}
	return err
}

// wrap will wrap errs within the parent error.
// If the parent supports errorWrapper, it will use that.
// Otherwise, it will generate the necessary structs
// to fit the structure.
func wrap(parent error, errs []error) error {
	// If the error supports WrapError, then use that
	// to modify the error to include the wrapped error.
	// Otherwise, we create a proxy type so Unwrap works.
	for len(errs) > 0 {
		// If errorWrapper is implemented, invoke it
		// and set the new error to the returned
		// value. Then return to the start of the loop.
		if err, ok := parent.(errorWrapper); ok {
			parent = err.WrapError(errs[0])
			errs = errs[1:]
			continue
		}

		// Create a default wrapper that conforms to
		// the errors API. If there is only one wrapped
		// error, use a version that supports Unwrap() error
		// since there's more compatibility with that interface.
		if len(errs) == 1 {
			return &wrapError{
				error: parent,
				err:   errs[0],
			}
		}
		return &wrapErrors{
			error: parent,
		}
	}
	return parent
}

type errorWrapper interface {
	// WrapError is used to include a wrapped error in the
	// parent error during unmarshaling. If an error doesn't
	// need direct knowledge of its wrapped error, then it
	// shouldn't implement this method and should instead
	// utilize the generic structure created during unmarshaling.
	//
	// This will return the error. It is ok if the error modifies
	// and returns itself.
	WrapError(err error) error
}

type wrapError struct {
	error
	err error
}

func (e *wrapError) Unwrap() error {
	return e.err
}

type wrapErrors struct {
	error
	errs []error
}

func (e *wrapErrors) Unwrap() []error {
	return e.errs
}

// ToNative returns the underlying error from a grpc service based on the grpc error code
func ToNative(err error) error {
	if err == nil {
		return nil
	}

	desc := errDesc(err)

	var cls error // divide these into error classes, becomes the cause

	switch code(err) {
	case codes.InvalidArgument:
		cls = errdefs.ErrInvalidArgument
	case codes.AlreadyExists:
		cls = errdefs.ErrAlreadyExists
	case codes.NotFound:
		cls = errdefs.ErrNotFound
	case codes.Unavailable:
		cls = errdefs.ErrUnavailable
	case codes.FailedPrecondition:
		if desc == errdefs.ErrConflict.Error() || strings.HasSuffix(desc, ": "+errdefs.ErrConflict.Error()) {
			cls = errdefs.ErrConflict
		} else if desc == errdefs.ErrNotModified.Error() || strings.HasSuffix(desc, ": "+errdefs.ErrNotModified.Error()) {
			cls = errdefs.ErrNotModified
		} else {
			cls = errdefs.ErrFailedPrecondition
		}
	case codes.Unimplemented:
		cls = errdefs.ErrNotImplemented
	case codes.Canceled:
		cls = context.Canceled
	case codes.DeadlineExceeded:
		cls = context.DeadlineExceeded
	case codes.Aborted:
		cls = errdefs.ErrAborted
	case codes.Unauthenticated:
		cls = errdefs.ErrUnauthenticated
	case codes.PermissionDenied:
		cls = errdefs.ErrPermissionDenied
	case codes.Internal:
		cls = errdefs.ErrInternal
	case codes.DataLoss:
		cls = errdefs.ErrDataLoss
	case codes.OutOfRange:
		cls = errdefs.ErrOutOfRange
	case codes.ResourceExhausted:
		cls = errdefs.ErrResourceExhausted
	default:
		if idx := strings.LastIndex(desc, cause.UnexpectedStatusPrefix); idx > 0 {
			if status, err := strconv.Atoi(desc[idx+len(cause.UnexpectedStatusPrefix):]); err == nil && status >= 200 && status < 600 {
				cls = cause.ErrUnexpectedStatus{Status: status}
			}
		}
		if cls == nil {
			cls = errdefs.ErrUnknown
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
