package gum

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
)

// Host is the value of the [http.Request.Host] field
type Host string

// Method is the value of the [http.Request.Method] field, e.g. GET, POST, etc
type Method string

// ContentLength is the value of the [http.Request.ContentLength] field.
// Only available if the value requests value is not negative.
type ContentLength int64

// ContentType holds the value of the requests Content-Type header.
type ContentType string

// RawBody is a byte slice holding the content of the requests [http.Request.Body] field.
type RawBody []byte

// Form contains the requests [http.Request.Form] as url.Values
type Form struct {
	url.Values
}

// PostForm contains the requests parsed [http.Request.PostForm] as url.Values
type PostForm struct {
	url.Values
}

// Query contains the requests query values as url.Values
type Query struct {
	url.Values
}

type MultipartFormMaxMemory int64

func init() {
	Register(func(r *http.Request) (*http.Request, error) {
		return r, nil
	})

	Register(func(r *http.Request) (http.Header, error) {
		return r.Header, nil
	})

	Register(func(r *http.Request) (io.Reader, error) {
		return r.Body, nil
	})

	Register(func(r *http.Request) (io.ReadCloser, error) {
		return r.Body, nil
	})

	Register(func(r *http.Request) (context.Context, error) {
		return r.Context(), nil
	})

	Register(func(r *http.Request) (*url.URL, error) {
		return r.URL, nil
	})

	Register(func(r *http.Request) (Query, error) {
		return Query{r.URL.Query()}, nil
	})

	Register(func(r *http.Request) (Method, error) {
		return Method(r.Method), nil
	})

	Register(func(r *http.Request) (Host, error) {
		return Host(r.Host), nil
	})

	Register(func(r *http.Request) (ContentLength, error) {
		if r.ContentLength == -1 {
			return 0, errors.New("ContentLength is unknown")
		}

		return ContentLength(r.ContentLength), nil
	})

	Register(func(r *http.Request) (Form, error) {
		if err := r.ParseForm(); err != nil {
			return Form{}, fmt.Errorf("parse form: %w", err)
		}

		return Form{r.Form}, nil
	})

	Register(func(r *http.Request) (PostForm, error) {
		if err := r.ParseForm(); err != nil {
			return PostForm{}, fmt.Errorf("parse form: %w", err)
		}

		return PostForm{r.PostForm}, nil
	})

	Register(func(r *http.Request) (*multipart.Form, error) {
		var maxMemory int64 = 1024 * 1024

		// try to get the max memory from the context
		memoryValue, _ := Extract[Option[ContextValue[MultipartFormMaxMemory]]](r)
		if value, ok := memoryValue.Get(); ok {
			maxMemory = int64(value.Value)
		}

		if err := r.ParseMultipartForm(maxMemory); err != nil {
			return nil, fmt.Errorf("parse multipart form: %w", err)
		}

		return r.MultipartForm, nil
	})

	Register(func(r *http.Request) (RawBody, error) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			return nil, fmt.Errorf("reading body: %w", err)
		}

		return body, nil
	})

	Register(func(r *http.Request) (ContentType, error) {
		contentType := r.Header.Get("Content-Type")
		if contentType == "" {
			return "", fmt.Errorf("no Content-Type header in request")
		}

		return ContentType(contentType), nil
	})
}
