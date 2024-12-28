package extractors

import (
	"bytes"
	"github.com/go-gum/gum"
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
		gum.Handler(func(v Try[ContentType]) { extractedValue = v }).ServeHTTP(nil, req)
		equal(t, extractedValue.Value, "application/json")
		equal(t, extractedValue.Error, nil)
	})

	t.Run("No Value", func(t *testing.T) {
		req := &http.Request{}

		var extractedValue Try[ContentType]
		gum.Handler(func(v Try[ContentType]) { extractedValue = v }).ServeHTTP(nil, req)
		equal(t, extractedValue.Value, "")
		notEqual(t, extractedValue.Error, nil)
	})

}

func TestOption_FromRequest(t *testing.T) {
	t.Run("Value", func(t *testing.T) {
		req := &http.Request{ContentLength: 1024}

		var extractedValue Try[ContentLength]
		gum.Handler(func(v Try[ContentLength]) { extractedValue = v }).ServeHTTP(nil, req)
		equal(t, extractedValue.Value, 1024)
		equal(t, extractedValue.Error, nil)
	})

	t.Run("No Value", func(t *testing.T) {
		req := &http.Request{ContentLength: -1}

		var extractedValue Try[ContentLength]
		gum.Handler(func(v Try[ContentLength]) { extractedValue = v }).ServeHTTP(nil, req)
		equal(t, extractedValue.Value, 0)
		notEqual(t, extractedValue.Error, nil)
	})

}

func TestJSON(t *testing.T) {
	body := bytes.NewReader([]byte(`{"foo": "bar"}`))
	req := &http.Request{Body: io.NopCloser(body)}

	type BodyStruct struct{ Foo string }

	var extractedValue BodyStruct
	gum.Handler(func(v JSON[BodyStruct]) { extractedValue = v.Value }).ServeHTTP(nil, req)
	equal(t, extractedValue, BodyStruct{Foo: "bar"})
}

func TestJSONParseError(t *testing.T) {
	body := bytes.NewReader([]byte(`{"foo": "ba`))
	req := &http.Request{Body: io.NopCloser(body)}

	type BodyStruct struct{ Foo string }

	var rw responseWriter
	gum.Handler(func(v JSON[BodyStruct]) { t.FailNow() }).ServeHTTP(&rw, req)
	equal(t, rw.statusCode, http.StatusBadRequest)
}

func TestContentValue(t *testing.T) {
	req := &http.Request{}

	// inject MyValue via middleware
	type MyValue string
	provideValue := ProvideContextValue(MyValue("foo bar"))

	var extractedValue MyValue
	handler := gum.Handler(func(v ContextValue[MyValue]) { extractedValue = v.Value })
	provideValue(handler).ServeHTTP(nil, req)
	equal(t, extractedValue, MyValue("foo bar"))
}
