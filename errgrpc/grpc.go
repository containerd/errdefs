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
	"reflect"
	"strconv"
	"strings"

	spb "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/protoadapt"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/containerd/typeurl/v2"

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
	if err == nil {
		return nil
	}

	if _, ok := status.FromError(err); ok {
		// error has already been mapped to grpc
		return err
	}

	s, extra := findStatus(err, "")
	if s != nil {
		var details []protoadapt.MessageV1

		for _, e := range extra {
			// Do not double encode proto messages, otherwise use Any
			if pm, ok := e.(protoadapt.MessageV1); ok {
				details = append(details, pm)
			} else if pm, ok := e.(proto.Message); ok {
				details = append(details, protoadapt.MessageV1Of(pm))
			} else {

				if reflect.TypeOf(e).Kind() == reflect.Ptr {
					a, aerr := typeurl.MarshalAny(e)
					if aerr == nil {
						details = append(details, &anypb.Any{
							TypeUrl: a.GetTypeUrl(),
							Value:   a.GetValue(),
						})
						continue
					}
				}
				if gs, ok := status.FromError(ToGRPC(e)); ok {
					details = append(details, gs.Proto())
				}
				// TODO: Else include unknown extra error type
			}
		}

		if len(details) > 0 {
			if ds, _ := s.WithDetails(details...); ds != nil {
				s = ds
			}
		}
		err = s.Err()
	}

	return err
}

// findStatus finds the first error which matches a GRPC status and returns
// any extra
func findStatus(err error, msg string) (*status.Status, []error) {
	switch uerr := err.(type) {
	case interface{ Unwrap() error }:
		if msg == "" {
			// preserve wrap message
			msg = err.Error()
		}
		return findStatus(uerr.Unwrap(), msg)
	case interface{ Unwrap() []error }:
		var (
			extra  []error
			status *status.Status
		)
		for _, e := range uerr.Unwrap() {
			// NOTE: Multi errors do not preserve message when created fmt.Errorf,
			// Document this and suggest use of errors.Join with individual messages
			// for each error.
			s, errs := findStatus(e, msg)
			if s != nil && status == nil {
				status = s
				extra = append(extra, errs...)
			} else {
				extra = append(extra, e)
			}
		}
		return status, extra
	}
	if msg == "" {
		msg = err.Error()
	}

	return getStatus(err, msg), nil
}

func getStatus(err error, msg string) *status.Status {
	switch {
	case errdefs.IsInvalidArgument(err):
		return status.New(codes.InvalidArgument, msg)
	case errdefs.IsNotFound(err):
		return status.New(codes.NotFound, msg)
	case errdefs.IsAlreadyExists(err):
		return status.New(codes.AlreadyExists, msg)
	case errdefs.IsFailedPrecondition(err) || errdefs.IsConflict(err) || errdefs.IsNotModified(err):
		return status.New(codes.FailedPrecondition, msg)
	case errdefs.IsUnavailable(err):
		return status.New(codes.Unavailable, msg)
	case errdefs.IsNotImplemented(err):
		return status.New(codes.Unimplemented, msg)
	case errdefs.IsCanceled(err):
		return status.New(codes.Canceled, msg)
	case errdefs.IsDeadlineExceeded(err):
		return status.New(codes.DeadlineExceeded, msg)
	case errdefs.IsUnauthorized(err):
		return status.New(codes.Unauthenticated, msg)
	case errdefs.IsPermissionDenied(err):
		return status.New(codes.PermissionDenied, msg)
	case errdefs.IsInternal(err):
		return status.New(codes.Internal, msg)
	case errdefs.IsDataLoss(err):
		return status.New(codes.DataLoss, msg)
	case errdefs.IsAborted(err):
		return status.New(codes.Aborted, msg)
	case errdefs.IsOutOfRange(err):
		return status.New(codes.OutOfRange, msg)
	case errdefs.IsResourceExhausted(err):
		return status.New(codes.ResourceExhausted, msg)
	case errdefs.IsUnknown(err):
		return status.New(codes.Unknown, msg)
	}
	return nil
}

// ToGRPCf maps the error to grpc error codes, assembling the formatting string
// and combining it with the target error string.
//
// This is equivalent to grpc.ToGRPC(fmt.Errorf("%s: %w", fmt.Sprintf(format, args...), err))
func ToGRPCf(err error, format string, args ...interface{}) error {
	return ToGRPC(fmt.Errorf("%s: %w", fmt.Sprintf(format, args...), err))
}

// ToNative returns the underlying error from a grpc service based on the grpc error code
func ToNative(err error) error {
	if err == nil {
		return nil
	}

	s, isGRPC := status.FromError(err)

	var (
		desc string
		code codes.Code
	)

	if isGRPC {
		desc = s.Message()
		code = s.Code()

	} else {
		desc = err.Error()
		code = codes.Unknown
	}

	var cls error // divide these into error classes, becomes the cause

	switch code {
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

	if isGRPC {
		errs := []error{err}
		for _, a := range s.Details() {
			if s, ok := a.(*spb.Status); ok {
				errs = append(errs, ToNative(status.ErrorProto(s)))
			} else if derr, ok := a.(error); ok {
				errs = append(errs, derr)
			} else if dany, ok := a.(typeurl.Any); ok {
				i, uerr := typeurl.UnmarshalAny(dany)
				if uerr == nil {
					if derr, ok = i.(error); ok {
						errs = append(errs, derr)
					}
					// TODO: Wrap unknown type in error for visibility
				}
				// TODO: Wrap unregistered type in error for visibility
			}
			// TODO: Wrap unknown type in error for visibility

		}
		if len(errs) > 1 {
			err = errors.Join(errs...)
		}
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
