package http

import (
	"bytes"
	"io"
	"testing"
)

// onlyReader strips WriterTo so io.Copy must use the temporary buffer path.
type onlyReader struct {
	r *bytes.Reader
}

func (o *onlyReader) Read(p []byte) (int, error) { return o.r.Read(p) }

type nopWriter struct{}

func (nopWriter) Write(p []byte) (int, error) { return len(p), nil }

func BenchmarkCopyBufferPool(b *testing.B) {
	payload := bytes.Repeat([]byte("x"), 64*1024)

	b.Run("io.Copy", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(len(payload)))
		br := bytes.NewReader(payload)
		src := &onlyReader{r: br}
		var dst nopWriter
		for b.Loop() {
			br.Reset(payload)
			_, _ = io.Copy(dst, src)
		}
	})

	b.Run("CopyBuffer+pool", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(len(payload)))
		br := bytes.NewReader(payload)
		src := &onlyReader{r: br}
		var dst nopWriter
		for b.Loop() {
			br.Reset(payload)
			bufp := getCopyBuf()
			_, _ = io.CopyBuffer(dst, src, *bufp)
			putCopyBuf(bufp)
		}
	})
}
