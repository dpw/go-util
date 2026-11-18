package errorsx

import (
	"bytes"
	"io"
)

// A Writer that precedes each line with a prefix to indent it.
type indentWriter struct {
	io.Writer
	prefix []byte

	// This flag records whether the writer is inside a line.  It
	// is false at the start of a line.
	midLine bool
}

func (iw *indentWriter) Write(p []byte) (int, error) {
	res := 0
	for len(p) != 0 {
		// If we are at the start of a line, write the
		// prefix before we write anything else.
		if !iw.midLine {
			if _, err := iw.Writer.Write(iw.prefix); err != nil {
				return res, err
			}
		}

		iw.midLine = true
		i := bytes.IndexByte(p, '\n')
		if i < 0 {
			// There's no newline remaining, so write out
			// the remaining bytes and we are done.
			n, err := iw.Writer.Write(p)
			return res + n, err
		}

		// Write up to and including the newline.
		n, err := iw.Writer.Write(p[:i+1])
		res += n
		if err != nil {
			return res, err
		}

		// Move on to the start of the next line.
		iw.midLine = false
		p = p[i+1:]
	}

	return res, nil
}

// Write a newline unless we are already at the start of a line.
func (iw *indentWriter) newline() {
	if iw.midLine {
		iw.Writer.Write(newline)
		iw.midLine = false
	}
}

var newline = []byte{'\n'}
