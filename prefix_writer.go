package main

import (
	"io"
)

type PrefixWriter struct {
	w      io.Writer
	Prefix string
}

func NewPrefixWriter(w io.Writer, prefix string) *PrefixWriter {
	return &PrefixWriter{
		w:      w,
		Prefix: prefix,
	}
}

func (pw *PrefixWriter) Write(b []byte) (n int, err error) {
	np, err := pw.w.Write([]byte(pw.Prefix + " "))
	if err != nil {
		return n, err
	}
	nb, err := pw.w.Write(b)
	return np + nb, err
}
