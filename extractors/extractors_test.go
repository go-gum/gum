package extractors

import (
	"bytes"
	"github.com/go-gum/gum"
	"net/http"
	"testing"
)

func TestExtractHost(t *testing.T) {
	req := &http.Request{Host: "example.com"}

	var extractedValue Host
	gum.Handler(func(v Host) { extractedValue = v }).ServeHTTP(nil, req)
	equal(t, extractedValue, "example.com")
}

func TestExtractContentType(t *testing.T) {
	req := &http.Request{
		Header: map[string][]string{
			"Content-Type": {"application/json"},
		},
	}

	var extractedValue ContentType
	gum.Handler(func(v ContentType) { extractedValue = v }).ServeHTTP(nil, req)
	equal(t, extractedValue, "application/json")
}

func TestExtractNoContentType(t *testing.T) {
	req := &http.Request{}

	var rw responseWriter
	gum.Handler(func(v ContentType) { t.FailNow() }).ServeHTTP(&rw, req)
	equal(t, rw.statusCode, http.StatusBadRequest)
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

func equal[T comparable](t *testing.T, actual, expected T) {
	if actual != expected {
		t.Fatalf("expected %#v to equal %#v", actual, expected)
	}
}

func notEqual[T comparable](t *testing.T, actual, expected T) {
	if actual == expected {
		t.Fatalf("expected %#v to not equal %#v", actual, expected)
	}
}
