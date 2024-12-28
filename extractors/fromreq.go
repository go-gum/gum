package extractors

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-gum/gum"
	"log/slog"
	"net/http"
	"reflect"
)

type Middleware = func(delegate http.Handler) http.Handler

// ContextValue uses the type T as the key to lookup a value
// of type T in the requests context.Context. Use WithContextValue to
// get a http.Handler middleware that injects a value into the context.Context
type ContextValue[T any] struct {
	Value T
}

var _ = gum.AssertFromRequest[ContextValue[any]]()

func (ContextValue[T]) FromRequest(r *http.Request) (ContextValue[T], error) {
	key := reflect.TypeFor[T]()
	value := r.Context().Value(key)
	if value == nil {
		return ContextValue[T]{}, fmt.Errorf("no value of type %q in context", key)
	}

	valueT, ok := value.(T)
	if !ok {
		return ContextValue[T]{}, fmt.Errorf("expected value of type %q, got %T", key, value)
	}

	return ContextValue[T]{Value: valueT}, nil
}

// ProvideContextValue provides a Middleware that injects a value of type T into the
// requests context. The value can later be extracted by using ContextValue.
func ProvideContextValue[T any](value T) Middleware {
	key := reflect.TypeFor[T]()
	return func(delegate http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			ctx := context.WithValue(request.Context(), key, value)
			request = request.WithContext(ctx)
			delegate.ServeHTTP(writer, request)
		})
	}
}

// ContextValueExtractor returns a gum.Extractor that extracts a value of type T
// from the context.Context that was previous provided using ProvideContextValue.
func ContextValueExtractor[T any]() gum.Extractor[T] {
	return func(r *http.Request) (T, error) {
		val, err := ContextValue[T]{}.FromRequest(r)
		return val.Value, err
	}
}

// JSON parses the requests body as json
type JSON[T any] struct {
	Value T
}

var _ = gum.AssertFromRequest[JSON[any]]()

func (JSON[T]) FromRequest(r *http.Request) (JSON[T], error) {
	var value T
	if err := json.NewDecoder(r.Body).Decode(&value); err != nil {
		return JSON[T]{}, fmt.Errorf("deserialize %T: %w", value, err)
	}

	return JSON[T]{Value: value}, nil
}

// Try tries to extract a T from the request but will not fail the request
// processing if extraction fails.
// A Try has either the Value or the Error field set.
type Try[T any] struct {
	Value T
	Error error
}

var _ = gum.AssertFromRequest[Try[any]]()

// Get gets the Try value and its error
func (o Try[T]) Get() (T, error) {
	return o.Value, o.Error
}

func (Try[T]) FromRequest(r *http.Request) (Try[T], error) {
	tValue, err := gum.Extract[T](r)
	if err != nil {
		result := Try[T]{Error: err}
		return result, nil
	}

	result := Try[T]{Value: tValue}
	return result, nil
}

// Option is similar to Try, it just swallows any error.
type Option[T any] struct {
	Value T
	IsSet bool
}

var _ = gum.AssertFromRequest[Option[any]]()

// Get gets the Option value and a boolean flag to test if
// the value is set.
func (o Option[T]) Get() (T, bool) {
	return o.Value, o.IsSet
}

func (Option[T]) FromRequest(r *http.Request) (Option[T], error) {
	try, err := gum.Extract[Try[T]](r)
	if err != nil {
		// Extracting a Try must never fail
		panic(err)
	}

	if try.Error != nil {
		// swallow error, return empty Option
		return Option[T]{}, nil
	}

	result := Option[T]{
		Value: try.Value,
		IsSet: true,
	}

	return result, nil
}

type Logger struct {
	ctx context.Context
	*slog.Logger
}

var _ = gum.AssertFromRequest[Logger]()

func (l Logger) FromRequest(r *http.Request) (Logger, error) {
	ctx := r.Context()

	log := slog.With(slog.String("path", r.URL.Path))
	log.DebugContext(ctx, "Request started")
	return Logger{ctx: ctx, Logger: log}, nil
}

func (l Logger) Close() error {
	l.DebugContext(l.ctx, "Request finished")
	return nil
}
