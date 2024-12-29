package gum

import (
	"bytes"
	"net/http"
	"testing"
)

func TestExtractHost(t *testing.T) {
	req := &http.Request{Host: "example.com"}

	var extractedValue Host
	Handler(func(v Host) { extractedValue = v }).ServeHTTP(nil, req)
	AssertEqual(t, extractedValue, "example.com")
}

func TestExtractContentType(t *testing.T) {
	req := &http.Request{
		Header: map[string][]string{
			"Content-Type": {"application/json"},
		},
	}

	var extractedValue ContentType
	Handler(func(v ContentType) { extractedValue = v }).ServeHTTP(nil, req)
	AssertEqual(t, extractedValue, "application/json")
}

func TestExtractNoContentType(t *testing.T) {
	req := &http.Request{}

	var rw responseWriter
	Handler(func(v ContentType) { t.FailNow() }).ServeHTTP(&rw, req)
	AssertEqual(t, rw.statusCode, http.StatusBadRequest)
}

type responseWriter struct {
	header     http.Header
	body       bytes.Buffer
	statusCode int
}

func (r *responseWriter) Header() http.Header {
	if r.header == nil {
		r.header = http.Header{}
	}

	return r.header
}

func (r *responseWriter) Write(bytes []byte) (int, error) {
	return r.body.Write(bytes)
}

func (r *responseWriter) WriteHeader(statusCode int) {
	r.statusCode = statusCode
}

func AssertEqual[T comparable](t *testing.T, actual, expected T) {
	if actual != expected {
		t.Fatalf("expected %#v to equal %#v", actual, expected)
	}
}

func AssertNotEqual[T comparable](t *testing.T, actual, expected T) {
	if actual == expected {
		t.Fatalf("expected %#v to not equal %#v", actual, expected)
	}
}

func AssertTrue(t *testing.T, actual bool) {
	if !actual {
		t.Fatal("expected value to be true")
	}
}
