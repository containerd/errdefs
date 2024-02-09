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
	"testing"
)

func TestInvalidArgument(t *testing.T) {
	for _, match := range []error{
		ErrInvalidArgument,
		&errInvalidArgument{},
		&customInvalidArgument{},
		&wrappedInvalidArgument{errors.New("invalid parameter")},
	} {
		if !IsInvalidArgument(match) {
			t.Errorf("error did not match invalid argument: %#v", match)
		}
	}
	for _, nonMatch := range []error{
		ErrUnknown,
		context.Canceled,
		errors.New("invalid argument"),
	} {
		if IsInvalidArgument(nonMatch) {
			t.Errorf("error unexpectedly matched invalid argument: %#v", nonMatch)
		}
	}
}

type customInvalidArgument struct{}

func (*customInvalidArgument) Error() string {
	return "my own invalid argument"
}

func (*customInvalidArgument) InvalidParameter() {}

type wrappedInvalidArgument struct{ error }

func (*wrappedInvalidArgument) InvalidParameter() {}
