package main

import (
	"bytes"
	"testing"
)

func TestPrefixWriter(t *testing.T) {
	w := bytes.NewBuffer([]byte{})
	pw := NewPrefixWriter(w, "prefix")
	if pw == nil {
		t.Errorf("Nil PrefixWriter returned\n")
	}
}

func TestPrefixWriterWriter(t *testing.T) {
	w := bytes.NewBuffer([]byte{})
	pw := NewPrefixWriter(w, "prefix")
	n, err := pw.Write([]byte("some-message"))
	if err != nil {
		t.Errorf("pw.Write failed with %q\n", err)
	}
	expected_n := 19
	if n != expected_n {
		t.Errorf("Invalid amount of bytes written %d, expected: %d\n", n, expected_n)
	}
	expected_written := "prefix some-message"
	if w.String() != expected_written {
		t.Errorf("Wrote: %q, expected: %q\n", w, expected_written)
	}
}
