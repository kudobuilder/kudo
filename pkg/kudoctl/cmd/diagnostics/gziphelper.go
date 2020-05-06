package diagnostics

import (
	"compress/gzip"
	"io"
)

type streamGzipper struct {
	bufSize int
	w io.Writer
}
// TODO: remove todo when logs are back
func newGzipWriter(w io.Writer, size int) *streamGzipper {
	return &streamGzipper{
		bufSize: size,
		w:       w,
	}
}

func (z *streamGzipper) Write(r io.ReadCloser) error {
	buf := make([]byte, z.bufSize)
	zw := gzip.NewWriter(z.w)
	var err error
	for {
		var n int
		n, err = r.Read(buf)
		if n > 0 {
			zw.Write(buf[:n])
		}
		if err != nil  {
			zw.Close()
			r.Close()
			break
		}
	}
	return err
}