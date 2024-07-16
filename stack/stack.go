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
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"sync/atomic"
	"unsafe"

	"github.com/containerd/typeurl/v2"
)

func init() {
	typeurl.Register((*Trace)(nil), "github.com/containerd/errdefs", "stack+json")
}

var (
	// Version is version of running process
	Version string = "dev"

	// Revision is the specific revision of the running process
	Revision string = "dirty"
)

func Callers(skip int) *Trace {
	return callers(skip + 2)
}

type Trace struct {
	decoded *trace

	callers []uintptr
	helpers []uintptr
}

type trace struct {
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

func (f Frame) Print(w io.Writer) {
	fmt.Fprintf(w, "%s\n\t%s:%d\n", f.Name, f.File, f.Line)
}

// callers returns the current stack, skipping over the number of frames mentioned
// Frames with skip=0:
//
//	frame[0] runtime.Callers
//	frame[1] <this function> github.com/containerd/errdefs/stack.callers
//	frame[2] <caller> (Use skip=2 to have this be first frame)
func callers(skip int) *Trace {
	const depth = 32
	var pcs [depth]uintptr
	n := runtime.Callers(skip, pcs[:])
	return &Trace{
		callers: pcs[0:n],
	}
}

func (s *Trace) getDecoded() *trace {
	if s.decoded == nil {
		unsafeDecoded := (*unsafe.Pointer)(unsafe.Pointer(&s.decoded))

		var helpers map[string]struct{}
		if len(s.helpers) > 0 {
			helpers = make(map[string]struct{})
			frames := runtime.CallersFrames(s.helpers)
			for {
				frame, more := frames.Next()
				helpers[frame.Function] = struct{}{}
				if !more {
					break
				}
			}
		}

		f := make([]Frame, 0, len(s.callers))
		if len(s.callers) > 0 {
			frames := runtime.CallersFrames(s.callers)
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

		t := trace{
			Version:  Version,
			Revision: Revision,
			Cmdline:  os.Args,
			Frames:   f,
			Pid:      int32(os.Getpid()),
		}

		atomic.StorePointer(unsafeDecoded, unsafe.Pointer(&t))
	}

	return s.decoded
}

func (s *Trace) Print(w io.Writer) {
	t := s.getDecoded()
	fmt.Fprintf(w, "%d %s %s\n", t.Pid, t.Version, strings.Join(t.Cmdline, " "))
	for _, f := range t.Frames {
		f.Print(w)
	}
	fmt.Fprintln(w)
}

func (s *Trace) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.getDecoded())
}

func (s *Trace) UnmarshalJSON(b []byte) error {
	unsafeDecoded := (*unsafe.Pointer)(unsafe.Pointer(&s.decoded))
	var t Trace

	if err := json.Unmarshal(b, &t); err != nil {
		return err
	}

	atomic.StorePointer(unsafeDecoded, unsafe.Pointer(&t))

	return nil
}
