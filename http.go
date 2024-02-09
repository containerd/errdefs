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
	"errors"
	"net/http"
)

// FromHTTP returns the error best matching the HTTP status code
func FromHTTP(statusCode int) error {
	switch statusCode {
	case http.StatusNotFound:
		return ErrNotFound
	case http.StatusBadRequest:
		return ErrInvalidArgument
	case http.StatusConflict:
		return ErrConflict
	case http.StatusPreconditionFailed:
		return ErrFailedPrecondition
	case http.StatusUnauthorized:
		return ErrUnauthenticated
	case http.StatusForbidden:
		return ErrPermissionDenied
	case http.StatusNotModified:
		return ErrNotModified
	case http.StatusTooManyRequests:
		return ErrResourceExhausted
	case http.StatusInternalServerError:
		return ErrInternal
	case http.StatusNotImplemented:
		return ErrNotImplemented
	case http.StatusServiceUnavailable:
		return ErrUnavailable
	default:
		return errUnexpectedStatus{statusCode}
	}
}

// ToHTTP returns the best status code for the given error
func ToHTTP(err error) int {
	switch {
	case IsNotFound(err):
		return http.StatusNotFound
	case IsInvalidArgument(err):
		return http.StatusBadRequest
	case IsConflict(err):
		return http.StatusConflict
	case IsNotModified(err):
		return http.StatusNotModified
	case IsFailedPrecondition(err):
		return http.StatusPreconditionFailed
	case IsUnauthorized(err):
		return http.StatusUnauthorized
	case IsPermissionDenied(err):
		return http.StatusForbidden
	case IsResourceExhausted(err):
		return http.StatusTooManyRequests
	case IsInternal(err):
		return http.StatusInternalServerError
	case IsNotImplemented(err):
		return http.StatusNotImplemented
	case IsUnavailable(err):
		return http.StatusServiceUnavailable
	case IsUnknown(err):
		var unexpected errUnexpectedStatus
		if errors.As(err, &unexpected) && unexpected.status >= 200 && unexpected.status < 600 {
			return unexpected.status
		}
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}
