package extractors

import (
	"context"
	"errors"
	"fmt"
	"github.com/go-gum/gum"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
)

// Host is the value of the Host header
type Host string

// Method is the value of the requests Method field, e.g. GET, POST, etc
type Method string

// ContentLength is the value of the Content-Length header. Only available
// if the value requests value is not negative.
type ContentLength int64

// ContentType holds the value of the requests Content-Type header.
type ContentType string

// RawBody is a byte slice holding the requests body.
type RawBody []byte

// Form contains the requests parsed form as url.Values
type Form struct {
	url.Values
}

// Query contains the requests query values as url.Values
type Query struct {
	url.Values
}

func init() {
	gum.Register(func(r *http.Request) (*http.Request, error) {
		return r, nil
	})

	gum.Register(func(r *http.Request) (http.Header, error) {
		return r.Header, nil
	})

	gum.Register(func(r *http.Request) (io.Reader, error) {
		return r.Body, nil
	})

	gum.Register(func(r *http.Request) (io.ReadCloser, error) {
		return r.Body, nil
	})

	gum.Register(func(r *http.Request) (context.Context, error) {
		return r.Context(), nil
	})

	gum.Register(func(r *http.Request) (*url.URL, error) {
		return r.URL, nil
	})

	gum.Register(func(r *http.Request) (Query, error) {
		return Query{r.URL.Query()}, nil
	})

	gum.Register(func(r *http.Request) (Method, error) {
		return Method(r.Method), nil
	})

	gum.Register(func(r *http.Request) (Host, error) {
		return Host(r.Host), nil
	})

	gum.Register(func(r *http.Request) (ContentLength, error) {
		if r.ContentLength == -1 {
			return 0, errors.New("ContentLength is unknown")
		}

		return ContentLength(r.ContentLength), nil
	})

	gum.Register(func(r *http.Request) (Form, error) {
		if err := r.ParseForm(); err != nil {
			return Form{}, fmt.Errorf("parse form: %w", err)
		}

		return Form{r.Form}, nil
	})

	gum.Register(func(r *http.Request) (*multipart.Form, error) {
		// TODO get maxMemory from request
		if err := r.ParseMultipartForm(1024 * 1024); err != nil {
			return nil, fmt.Errorf("parse multipart form: %w", err)
		}

		return r.MultipartForm, nil
	})

	gum.Register(func(r *http.Request) (RawBody, error) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			return nil, fmt.Errorf("reading body: %w", err)
		}

		return body, nil
	})

	gum.Register(func(r *http.Request) (ContentType, error) {
		contentType := r.Header.Get("Content-Type")
		if contentType == "" {
			return "", fmt.Errorf("no Content-Type header in request")
		}

		return ContentType(contentType), nil
	})
}
