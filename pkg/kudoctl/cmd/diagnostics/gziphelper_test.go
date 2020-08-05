package diagnostics

import (
	"bytes"
	"io"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

type failingReader struct {
	io.ReadCloser
}

func (r *failingReader) Read(p []byte) (n int, err error) {
	n, _ = r.ReadCloser.Read(p)
	return n, errFakeTestError
}

func Test_streamGzipper_write(t *testing.T) {
	tests := []struct {
		desc     string
		stream   io.ReadCloser
		expected string
		wantErr  bool
	}{
		{
			desc:     "gzip OK",
			stream:   ioutil.NopCloser(strings.NewReader(testLog)),
			expected: testLogGZipped,
			wantErr:  false,
		},
		{
			desc:     "gzip fails, flush what's read",
			stream:   &failingReader{ioutil.NopCloser(strings.NewReader(testLog))},
			expected: testLogGZipped,
			wantErr:  true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.desc, func(t *testing.T) {
			var b bytes.Buffer
			z := &streamGzipper{
				w: &b,
			}
			err := z.write(tt.stream)
			assert.True(t, err != nil == tt.wantErr)
			assert.Equal(t, tt.expected, b.String())
		})
	}
}
