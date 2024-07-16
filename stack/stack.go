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

package stack

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"sync/atomic"
	"unsafe"

	"github.com/containerd/errdefs"
	"github.com/containerd/typeurl/v2"
)

func init() {
	typeurl.Register((*Error)(nil), "github.com/containerd/errdefs", "stack+json")
}

var (
	// Version is version of running process
	Version string = "dev"

	// Revision is the specific revision of the running process
	Revision string = "dirty"
)

type Error struct {
	decoded *Trace

	callers []uintptr
	helpers []uintptr
}

type Trace struct {
	Version  string   `json:"version,omitempty"`
	Revision string   `json:"revision,omitempty"`
	Cmdline  []string `json:"cmdline,omitempty"`
	Frames   []Frame  `json:"frames,omitempty"`
	Pid      int32    `json:"pid,omitempty"`
}

// Frame is a single frame of the trace representing a line of code
type Frame struct {
	Name string `json:"Name,omitempty"`
	File string `json:"File,omitempty"`
	Line int32  `json:"Line,omitempty"`
}

func (f Frame) WriteTo(w io.Writer) {
	fmt.Fprintf(w, "%s\n\t%s:%d\n", f.Name, f.File, f.Line)
}

// Callers returns a stack.Error with a customized amount
// of skipped frames.
func Callers(skip int) *Error {
	// This function calls two other functions so set the
	// default minimum skip to 2.
	return callers(skip + 2)
}

// callers returns the current stack, skipping over the number of frames mentioned
// Frames with skip=0:
//
//	frame[0] runtime.Callers
//	frame[1] <this function> github.com/containerd/errdefs/stack.callers
//	frame[2] <caller> (Use skip=2 to have this be first frame)
func callers(skip int) *Error {
	const depth = 32
	var pcs [depth]uintptr
	n := runtime.Callers(skip, pcs[:])
	return &Error{
		callers: pcs[0:n],
	}
}

func (e *Error) getDecoded() *Trace {
	if e.decoded == nil {
		unsafeDecoded := (*unsafe.Pointer)(unsafe.Pointer(&e.decoded))

		var helpers map[string]struct{}
		if len(e.helpers) > 0 {
			helpers = make(map[string]struct{})
			frames := runtime.CallersFrames(e.helpers)
			for {
				frame, more := frames.Next()
				helpers[frame.Function] = struct{}{}
				if !more {
					break
				}
			}
		}

		f := make([]Frame, 0, len(e.callers))
		if len(e.callers) > 0 {
			frames := runtime.CallersFrames(e.callers)
			for {
				frame, more := frames.Next()
				if _, ok := helpers[frame.Function]; !ok {
					f = append(f, Frame{
						Name: frame.Function,
						File: frame.File,
						Line: int32(frame.Line),
					})
				}
				if !more {
					break
				}
			}
		}

		t := Trace{
			Version:  Version,
			Revision: Revision,
			Cmdline:  os.Args,
			Frames:   f,
			Pid:      int32(os.Getpid()),
		}

		atomic.StorePointer(unsafeDecoded, unsafe.Pointer(&t))
	}

	return e.decoded
}

// Error implements the error interface. This method is rarely
// called because this is a collapsible error and the New/Join
// function will remove this error from non-verbose output.
func (e *Error) Error() string {
	return fmt.Sprintf("%+v", e.getDecoded())
}

func (e *Error) Format(st fmt.State, verb rune) {
	if verb == 'v' && st.Flag('+') {
		t := e.getDecoded()
		fmt.Fprintf(st, "%d %s %s\n", t.Pid, t.Version, strings.Join(t.Cmdline, " "))
		for _, f := range t.Frames {
			f.WriteTo(st)
		}
		fmt.Fprintln(st)
	}
}

func (e *Error) MarshalJSON() ([]byte, error) {
	return json.Marshal(e.getDecoded())
}

func (e *Error) UnmarshalJSON(b []byte) error {
	unsafeDecoded := (*unsafe.Pointer)(unsafe.Pointer(&e.decoded))
	var t Error

	if err := json.Unmarshal(b, &t); err != nil {
		return err
	}

	atomic.StorePointer(unsafeDecoded, unsafe.Pointer(&t))

	return nil
}

func (e *Error) CollapseError() {}

// ErrStack is a convenience method for calling Callers
// with the correct skip value for non-helper functions
// directly calling this package.
func ErrStack() error {
	// Skip the call to Callers and ErrStack.
	return Callers(2)
}

// Errorf creates a new error with the given format and
// arguments and adds a stack trace if one isn't already
// included.
func Errorf(format string, args ...any) error {
	err := errdefs.New(format, args...)
	if !hasStack(err) {
		err = errdefs.Join(err, Callers(2))
	}
	return err
}

// Join joins the errors and adds a stack trace if one
// isn't already present.
func Join(errs ...error) error {
	err := errdefs.Join(errs...)
	if !hasStack(err) {
		err = errdefs.Join(err, Callers(2))
	}
	return err
}

func hasStack(err error) bool {
	se := &Error{}
	return errors.As(err, &se)
}
