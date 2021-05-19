package pattern

import (
	"bytes"
	"errors"
)

var ErrNoCapture = errors.New("at least one capture is required")

type Matcher interface {
	Matches(in []byte) [][]byte
}

type matcher struct {
	e expr

	captures [][]byte
}

func New(in string) (Matcher, error) {
	e, err := parseExpr(in)
	if err != nil {
		return nil, err
	}
	if !e.hasCapture() {
		return nil, ErrNoCapture
	}
	// todo validate that two consecutive capture are literal never exists.
	return &matcher{
		e:        e,
		captures: make([][]byte, 0, e.captureCount()),
	}, nil
}

// Matches matches the given line with the provided pattern.
// Matches invalidates the previous returned captures array.
func (m *matcher) Matches(in []byte) [][]byte {
	if len(in) == 0 {
		return nil
	}
	if len(m.e) == 0 {
		return nil
	}
	captures := m.captures[:0]
	expr := m.e
	if ls, ok := expr[0].(literals); ok {
		i := bytes.Index(in, ls)
		if i != 0 {
			return nil
		}
		in = in[len(ls):]
		expr = expr[1:]
	}
	if len(expr) == 0 {
		return nil
	}
	// from now we have capture - literals - capture ... (literals)?
	for len(expr) != 0 {
		if len(expr) == 1 { // we're ending on a capture.
			if !(expr[0].(capture)).isUnamed() {
				captures = append(captures, in)
			}
			return captures
		}
		cap := expr[0].(capture)
		ls := expr[1].(literals)
		expr = expr[2:]
		i := bytes.Index(in, ls)
		if i == -1 {
			captures = append(captures, in)
			return captures
		}

		if cap.isUnamed() {
			in = in[len(ls)+i:]
			continue
		}
		captures = append(captures, in[:i])
		in = in[len(ls)+i:]
	}

	return captures
}
