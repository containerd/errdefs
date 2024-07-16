package errdefs

import (
	"fmt"
	"strings"

	"github.com/containerd/errdefs/internal/types"
)

type joinError struct {
	errs []error
}

// Join will join the errors together and ensure stack traces
// are appropriately formatted.
func Join(errs ...error) error {
	var e error
	n := 0
	for _, err := range errs {
		if err != nil {
			e = err
			n++
		}
	}

	switch n {
	case 0:
		return nil
	case 1:
		switch e.(type) {
		case *errorValue, *joinError:
			// Don't wrap the types defined by this package
			// as that could interfere with the formatting.
			return e
		}
		return &errorValue{e}
	}

	joined := make([]error, 0, n)
	for _, err := range errs {
		if err != nil {
			joined = append(joined, err)
		}
	}
	return &joinError{errs: joined}
}

func (e *joinError) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%v", e)
	return b.String()
}

func (e *joinError) Format(st fmt.State, verb rune) {
	format := fmt.FormatString(st, verb)
	collapsed := verb == 'v' && st.Flag('+')
	first := true
	for _, err := range e.errs {
		if !collapsed {
			if _, ok := err.(types.CollapsibleError); ok {
				continue
			}
		}
		if !first {
			fmt.Fprintln(st)
		}
		fmt.Fprintf(st, format, err)
		first = false
	}
}

func (e *joinError) Unwrap() []error {
	return e.errs
}
