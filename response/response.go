package response

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"github.com/timewasted/go-accept-headers"
	"io"
	"log/slog"
	"maps"
	"net/http"
)

// Raw just writes the given bytes to the http.ResponseWriter.
// It does not touch the headers nor the status code.
func Raw(content []byte) Response {
	var body WriteBody = func(writer io.Writer) error {
		// the only error could be a disconnect on the client side,
		// and that is fine for now
		_, err := writer.Write(content)
		return err
	}

	return New(body)
}

func Text(content string) Response {
	return Raw([]byte(content)).
		SetHeader("Content-Type", "text/plain; charset=utf8")
}

func HTML(content string) Response {
	return Raw([]byte(content)).
		SetHeader("Content-Type", "text/html; charset=utf8")
}

func Error(err error, statusCode int) Response {
	return Text(err.Error()).
		WithStatusCode(statusCode)
}

func Reader(r io.Reader) Response {
	return New(func(w io.Writer) error {
		_, err := io.Copy(w, r)
		return err
	})
}

func ReadCloser(r io.ReadCloser) Response {
	return New(func(w io.Writer) error {
		defer func() { _ = r.Close() }()

		_, err := io.Copy(w, r)
		return err
	})
}

// JSON prepares a Response handler that encodes the provided value using json.Encoder and
// and sets the content type header to "application/json"
func JSON(value any) Lazy {
	return LazyNew(func(statusCode int, headers http.Header, req *http.Request) http.Handler {
		encoded, err := json.Marshal(value)
		if err != nil {
			slog.WarnContext(req.Context(),
				"Failed to write json response",
				slog.String("err", err.Error()),
			)

			err = fmt.Errorf("encoding json: %w", err)
			return Error(err, http.StatusInternalServerError)
		}

		return Raw(encoded).
			UpdateWith(statusCode, headers).
			SetHeader("Content-Type", "application/xml; charset=utf8")
	})
}

// XML prepares a Response handler that encodes the provided value using xml.Encoder and
// and sets the content type header to "application/xml"
func XML(value any) Lazy {
	return LazyNew(func(statusCode int, headers http.Header, req *http.Request) http.Handler {
		encoded, err := xml.Marshal(value)
		if err != nil {
			slog.WarnContext(req.Context(),
				"Failed to write xml response",
				slog.String("err", err.Error()),
			)

			err = fmt.Errorf("encoding xml: %w", err)
			return Error(err, http.StatusInternalServerError)
		}

		return Raw(encoded).
			UpdateWith(statusCode, headers).
			SetHeader("Content-Type", "application/xml; charset=utf8")
	})
}

// Encoded prepares a Lazy handler that encodes the provided value according to the
// http.Request Accept header
func Encoded(value any) Lazy {
	return LazyNew(func(statusCode int, header http.Header, req *http.Request) http.Handler {
		acceptSlice := accept.Parse(req.Header.Get("Accept"))

		// decide on the content type
		ctype, err := acceptSlice.Negotiate("application/json", "application/xml")
		if err != nil {
			slog.WarnContext(
				req.Context(),
				"negotiate content type",
				slog.String("err", err.Error()),
			)

			return Error(err, http.StatusBadRequest)
		}

		switch ctype {
		case "application/xml":
			return XML(value).UpdateWith(statusCode, header)
		default:
			return JSON(value).UpdateWith(statusCode, header)
		}
	})
}

type WriteBody func(w io.Writer) error

type Response struct {
	statusCode int
	header     http.Header
	body       WriteBody
}

func New(body WriteBody) Response {
	return Response{
		header: make(http.Header),
		body:   body,
	}
}

func NoContent() Response {
	return New(nil)
}

func (r Response) WithStatusCode(statusCode int) Response {
	r.statusCode = statusCode
	return r
}

func (r Response) AddHeader(key, value string) Response {
	r.header.Add(key, value)
	return r
}

func (r Response) SetHeader(key, value string) Response {
	r.header.Set(key, value)
	return r
}

func (r Response) DelHeader(key string) Response {
	r.header.Del(key)
	return r
}

func (r Response) UpdateWith(statusCode int, header http.Header) Response {
	if statusCode > 0 {
		r.statusCode = statusCode
	}

	maps.Copy(r.header, header)
	return r
}

func (r Response) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	maps.Copy(writer.Header(), r.header)

	if r.statusCode == 0 && r.body == nil {
		// default to 204 No Content if no status code is set
		// and no body is defined
		r.statusCode = http.StatusNoContent
	}

	if r.statusCode > 0 {
		writer.WriteHeader(r.statusCode)
	}

	if r.body != nil {
		err := r.body(writer)
		if err != nil {
			slog.WarnContext(request.Context(),
				"writing body",
				slog.String("err", err.Error()),
			)
		}
	}
}

type Lazy struct {
	statusCode int
	header     http.Header
	body       func(statusCode int, headers http.Header, req *http.Request) http.Handler
}

func LazyNew(body func(statusCode int, headers http.Header, req *http.Request) http.Handler) Lazy {
	return Lazy{
		header: make(http.Header),
		body:   body,
	}
}

func (l Lazy) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	l.body(l.statusCode, l.header, request).ServeHTTP(writer, request)
}

func (l Lazy) WithStatusCode(statusCode int) Lazy {
	l.statusCode = statusCode
	return l
}

func (l Lazy) AddHeader(key, value string) Lazy {
	l.header.Add(key, value)
	return l
}

func (l Lazy) SetHeader(key, value string) Lazy {
	l.header.Set(key, value)
	return l
}

func (l Lazy) DelHeader(key string) Lazy {
	l.header.Del(key)
	return l
}

func (l Lazy) UpdateWith(statusCode int, header http.Header) Lazy {
	if statusCode > 0 {
		l.statusCode = statusCode
	}

	maps.Copy(l.header, header)
	return l
}
