//go:build !solution

package otp

import (
	"io"
)

type StreamReader struct {
	io.Reader
	r    io.Reader
	prng io.Reader
}

type StreamWriter struct {
	io.Writer
	w    io.Writer
	prng io.Reader
}

func (sr *StreamReader) Read(p []byte) (n int, err error) {
	n, err = sr.r.Read(p)
	if n == 0 {
		return
	}
	prng := make([]byte, n)
	sr.prng.Read(prng)
	for i := 0; i < n; i++ {
		p[i] ^= prng[i]
	}
	return
}

func (sw *StreamWriter) Write(p []byte) (n int, err error) {
	prng := make([]byte, len(p))
	sw.prng.Read(prng)
	for i := 0; i < len(p); i++ {
		prng[i] ^= p[i]
	}
	return sw.w.Write(prng)
}

func NewReader(r io.Reader, prng io.Reader) io.Reader {
	return &StreamReader{r: r, prng: prng}
}

func NewWriter(w io.Writer, prng io.Reader) io.Writer {
	return &StreamWriter{w: w, prng: prng}
}
