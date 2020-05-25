package diagnostics

import (
	"io"

	"compress/gzip"
)

// streamGzipper - a helper for gzipping a stream
type streamGzipper struct {
	w io.Writer
}

func newGzipWriter(w io.Writer) *streamGzipper {
	return &streamGzipper{
		w: w,
	}
}

// write - gzip the provided stream by sequential reads into the underlying bytes buffer and gzipping the bytes
func (z *streamGzipper) write(r io.ReadCloser) error {
	zw := gzip.NewWriter(z.w)
	var err error
	var written int64
	for {
		written, err = io.Copy(zw, r)
		if err != nil || written == 0 {
			_ = zw.Close()
			_ = r.Close()
			break
		}
	}
	if err == io.EOF {
		return nil
	}
	return err
}
