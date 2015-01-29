package main

import (
	"io"
)

type PrefixWriter struct {
	w      io.Writer
	Prefix string
}

func (pw *PrefixWriter) Write(b []byte) (n int, err error) {
	pw.w.Write([]byte(pw.Prefix + " "))
	return pw.w.Write(b)
}
