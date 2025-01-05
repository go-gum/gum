package gum

import (
	"bytes"
	. "github.com/go-gum/gum/internal/test"
	"io"
	"net/http"
	"testing"
)

func TestTry_FromRequest(t *testing.T) {
	t.Run("Value", func(t *testing.T) {
		req := &http.Request{
			Header: map[string][]string{
				"Content-Type": {"application/json"},
			},
		}

		var extractedValue Try[ContentType]
		Handler(func(v Try[ContentType]) { extractedValue = v }).ServeHTTP(nil, req)
		AssertEqual(t, extractedValue.Value, "application/json")
		AssertEqual(t, extractedValue.Error, nil)
	})

	t.Run("No Value", func(t *testing.T) {
		req := &http.Request{}

		var extractedValue Try[ContentType]
		Handler(func(v Try[ContentType]) { extractedValue = v }).ServeHTTP(nil, req)
		AssertEqual(t, extractedValue.Value, "")
		AssertNotEqual(t, extractedValue.Error, nil)
	})

}

func TestOption_FromRequest(t *testing.T) {
	t.Run("Value", func(t *testing.T) {
		req := &http.Request{ContentLength: 1024}

		var extractedValue Try[ContentLength]
		Handler(func(v Try[ContentLength]) { extractedValue = v }).ServeHTTP(nil, req)
		AssertEqual(t, extractedValue.Value, 1024)
		AssertEqual(t, extractedValue.Error, nil)
	})

	t.Run("No Value", func(t *testing.T) {
		req := &http.Request{ContentLength: -1}

		var extractedValue Try[ContentLength]
		Handler(func(v Try[ContentLength]) { extractedValue = v }).ServeHTTP(nil, req)
		AssertEqual(t, extractedValue.Value, 0)
		AssertNotEqual(t, extractedValue.Error, nil)
	})

}

func TestJSON(t *testing.T) {
	body := bytes.NewReader([]byte(`{"foo": "bar"}`))
	req := &http.Request{Body: io.NopCloser(body)}

	type BodyStruct struct{ Foo string }

	var extractedValue BodyStruct
	Handler(func(v JSON[BodyStruct]) { extractedValue = v.Value }).ServeHTTP(nil, req)
	AssertEqual(t, extractedValue, BodyStruct{Foo: "bar"})
}

func TestJSONParseError(t *testing.T) {
	body := bytes.NewReader([]byte(`{"foo": "ba`))
	req := &http.Request{Body: io.NopCloser(body)}

	type BodyStruct struct{ Foo string }

	var rw responseWriter
	Handler(func(v JSON[BodyStruct]) { t.FailNow() }).ServeHTTP(&rw, req)
	AssertEqual(t, rw.statusCode, http.StatusBadRequest)
}

func TestContentValue(t *testing.T) {
	req := &http.Request{}

	// inject MyValue via middleware
	type MyValue string
	provideValue := ProvideContextValue(MyValue("foo bar"))

	var extractedValue MyValue
	handler := Handler(func(v ContextValue[MyValue]) { extractedValue = v.Value })
	provideValue(handler).ServeHTTP(nil, req)
	AssertEqual(t, extractedValue, MyValue("foo bar"))
}
