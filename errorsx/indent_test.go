package errorsx

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/alecthomas/assert/v2"
)

func TestIndentWriter(t *testing.T) {
	cases := []string{
		"a", "\na", "\n\na",
		"ab", "\nab", "a\nb", "\na\nb",
	}

	var buf bytes.Buffer
	for _, c := range cases {
		b := []byte(c)

		t.Logf("%q", b)
		buf.Reset()
		iw := indentWriter{Writer: &buf, prefix: []byte("x")}

		n, err := iw.Write(b)
		assert.NoError(t, err)
		assert.Equal(t, len(b), n)
		assert.Equal(t, "x"+strings.ReplaceAll(c, "\n", "\nx"),
			buf.String())

		for i := range b {
			t.Logf("%q %q", b[:i], b[i:])
			buf.Reset()
			iw := indentWriter{Writer: &buf, prefix: []byte("x")}

			n, err := iw.Write(b[:i])
			assert.NoError(t, err)
			assert.Equal(t, i, n)
			n, err = iw.Write(b[i:])
			assert.NoError(t, err)
			assert.Equal(t, len(b)-i, n)

			assert.Equal(t, "x"+strings.ReplaceAll(c, "\n", "\nx"),
				buf.String())
		}

	}
}

func TestIndentWriterErrors(t *testing.T) {
	buf := erroringBuffer{t: t, err: errors.New("bang")}

	c := "a\nb"
	expect := "xya\nxyb"
	for i := range len(expect) {
		t.Logf("errorAt = %d", i)
		buf.Reset()
		buf.errorAt = i
		iw := indentWriter{Writer: &buf, prefix: []byte("xy")}

		n, err := iw.Write([]byte(c))
		assert.Equal(t, buf.err, err)
		assert.Equal(t, i, buf.Len())
		assert.Equal(t, strings.ReplaceAll(strings.ReplaceAll(buf.String(), "x", ""), "y", ""), c[:n])
	}
}

type erroringBuffer struct {
	t *testing.T
	bytes.Buffer
	err     error
	errorAt int
}

func (eb *erroringBuffer) Write(p []byte) (int, error) {
	space := eb.errorAt - eb.Len()
	eb.t.Logf("Writing %q, space %d", p, space)
	if len(p) <= space {
		return eb.Buffer.Write(p)
	}
	eb.Buffer.Write(p[:space])
	return space, eb.err
}
