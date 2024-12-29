package gum

import (
	"context"
	"errors"
	"fmt"
	"github.com/go-gum/gum/response"
	"io"
	"log/slog"
	"net/http"
	"reflect"
	"sync"
)

// FromRequest defines a method that extracts a T from a http.Request.
// See FromRequest.FromRequest for more details.
// I would like to Type it as FromRequest[T FromRequest[T]],
// but that is not possible as of the time of writing.
type FromRequest[T any] interface {
	// FromRequest creates a new instance of T.
	//
	// It should be seen as a static method and only be implemented
	// on the type T itself, e.g. for a type Foo:
	//
	//   func (Foo) FromRequest(*http.Request) (Foo, error) {
	//     return "foo", nil
	//   }
	FromRequest(r *http.Request) (T, error)
}

// AssertFromRequest asserts that the FromRequest interface is correctly
// implemented for the given type T.
//
// You can use it as a compile time check with a static variable or in an init function:
//
//	// static variable
//	var _ = AssertFromRequest[JSON[any]]()
//
//	func init() {
//	  AssertFromRequest[JSON[any]]()
//	}
func AssertFromRequest[T FromRequest[T]]() T {
	var Nil T
	return Nil
}

// Extract extracts a value of type T from the http.Request. T must either
// implement the FromRequest interface, or have an Extractor registered using
// the Register function.
//
// TODO document error, maybe panic
func Extract[T any](r *http.Request) (T, error) {
	rValue, err := extractorOf(reflect.TypeFor[T]())(r)
	if err != nil {
		var tNil T
		return tNil, err
	}

	return rValue.Interface().(T), nil
}

// Stores a mapping from reflect.TypeFor[T] to a Extractor[T]
var extractors sync.Map

// Extractor extracts a T from a request. This should be used for non
// generic types. Implement FromRequest for type T if T itself is generic.
type Extractor[T any] func(r *http.Request) (T, error)

// extractor extracts a generic reflect.Value for a request
type extractor Extractor[reflect.Value]

// Register registers an Extractor function for the given T.
// An already existing registration for T will be replaced.
// This method is threadsafe.
func Register[T any](fn Extractor[T]) {
	ty := reflect.TypeFor[T]()

	ex := func(request *http.Request) (reflect.Value, error) {
		value, err := fn(request)
		if err != nil {
			return reflect.Value{}, err
		}

		return reflect.ValueOf(value), nil
	}

	extractors.Store(ty, extractor(ex))
}

// Handler adapts a gum handler into an http.Handler. If for any of the handlers parameters
// cannot be provided by any registered Extractor, nor it implements FromRequest, a panic
// is raised immediately.
//
// The provided handler function must have either
//   - no return type
//   - a single error value
//   - a single value that implements http.Handler
//   - a value that implements http.Handler and an error value
func Handler(f any) http.Handler {
	fn := reflect.ValueOf(f)
	fnType := fn.Type()

	// must be a function
	if fnType.Kind() != reflect.Func {
		panic(fmt.Errorf("expected Func, got %q", fn.Type()))
	}

	// build one extractor per argument
	var extractors []extractor
	for idx := range fnType.NumIn() {
		extractors = append(extractors, extractorOf(fnType.In(idx)))
	}

	// build an output mapper
	mapOutputs := mapOutputsOf(fnType)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// TODO do we want to keep this?
		// inject the ResponseWriter into the requests context so
		// an Extractor can extract it if needed
		ctx := context.WithValue(r.Context(), reflect.TypeFor[http.ResponseWriter](), w)
		r = r.WithContext(ctx)

		var params []reflect.Value

		// extract all values into the params array
		for idx, extractor := range extractors {
			param, err := extractor(r)
			if err != nil {
				// TODO handle Extractor errors
				err = fmt.Errorf("extract parameter %d of %q: %w", idx, fnType, err)
				response.
					Error(err, http.StatusBadRequest).
					ServeHTTP(w, r)

				return
			}

			params = append(params, param)
		}

		// call the handler function with the collected parameters
		outputs := fn.Call(params)

		// map the generic output values
		result, err := mapOutputs(outputs)
		switch {
		case err != nil:
			// TODO handle Handler errors
			response.
				Error(err, http.StatusInternalServerError).
				ServeHTTP(w, r)

		case result != nil:
			result.ServeHTTP(w, r)
		}

		// if any of the actual parameters implement io.Closer, the
		// close function will be called now
		for idx, param := range params {
			if closer, ok := param.Interface().(io.Closer); ok {
				err := closer.Close()
				if err != nil {
					slog.WarnContext(ctx, "Call Close() on parameter failed",
						slog.Int("idx", idx),
						slog.String("fnType", fnType.String()),
						slog.String("err", err.Error()),
					)
				}
			}
		}
	})
}

// newValue returns a new instance of type ty. If ty is a pointer,
// it will also create an instance of the type ty points to, recursively.
func newValue(ty reflect.Type) reflect.Value {
	if ty.Kind() == reflect.Pointer {
		ptr := reflect.New(ty).Elem()
		ptr.Set(newValue(ty.Elem()).Addr())
		return ptr
	}

	return reflect.New(ty).Elem()
}

func mapOutputsOf(fnType reflect.Type) func(values []reflect.Value) (http.Handler, error) {
	switch fnType.NumOut() {
	case 0:
		// mapping function does nothing, we have no output values
		return func(values []reflect.Value) (http.Handler, error) { return nil, nil }

	case 1:
		isHandler := fnType.Out(0).Implements(reflect.TypeFor[http.Handler]())

		if isHandler {
			return func(values []reflect.Value) (http.Handler, error) {
				handler := interfaceOf[http.Handler](values[0])
				return handler, nil
			}
		} else {
			return func(values []reflect.Value) (http.Handler, error) {
				err := interfaceOf[error](values[0])
				return nil, err
			}
		}

	case 2:
		o0, o1 := fnType.Out(0), fnType.Out(1)

		if !o0.Implements(reflect.TypeFor[http.Handler]()) {
			panic(fmt.Errorf("%s does not implement http.Handler", o0))
		}

		if !o1.Implements(reflect.TypeFor[error]()) {
			panic(fmt.Errorf("%s does not implement error", o1))
		}

		return func(values []reflect.Value) (http.Handler, error) {
			handler := interfaceOf[http.Handler](values[0])
			err := interfaceOf[error](values[1])
			return handler, err
		}

	default:
		panic(fmt.Errorf("function has unsupported return type %s", fnType))
	}
}

// Builds an extractor for he given type.
// This method panics if building an extractor is not possible.
func extractorOf(ty reflect.Type) extractor {
	// first check list of registered extractors
	if ex, ok := extractors.Load(ty); ok && ex != nil {
		return ex.(extractor)
	}

	// ty must implement FromRequest[ty]
	fromRequest, err := lookupFromRequestMethod(ty)
	if err != nil {
		panic(fmt.Errorf("lookup FromRequest of %s: %w", ty, err))
	}

	return func(req *http.Request) (reflect.Value, error) {
		// instantiate a new zero value
		zeroValue := newValue(ty)

		// call the FromRequest method on the zero value
		params := []reflect.Value{zeroValue, reflect.ValueOf(req)}
		outputs := fromRequest.Func.Call(params)

		// unpack return values
		value, err := outputs[0], outputs[1]

		// check the error from the second return value
		if err := interfaceOf[error](err); err != nil {
			return reflect.Value{}, fmt.Errorf("extract %q: %w", ty, err)
		}

		// we have successfully extracted a value
		return value, nil
	}
}

func lookupFromRequestMethod(ty reflect.Type) (reflect.Method, error) {
	m, ok := ty.MethodByName("FromRequest")
	if !ok {
		return m, errors.New("method is missing")
	}

	if m.Type.NumIn() != 2 ||
		m.Type.In(0) != ty ||
		m.Type.In(1) != reflect.TypeFor[*http.Request]() {

		return m, fmt.Errorf("must have signature func (%s) FromRequest(*http.Request) (%s, error)", ty, ty)
	}

	if m.Type.NumOut() != 2 ||
		m.Type.Out(0) != ty ||
		m.Type.Out(1) != reflect.TypeFor[error]() {

		return m, fmt.Errorf("must return tuple (%s, error)", ty)
	}

	return m, nil
}

// Extracts the value of type T from the given reflection value
// if the reflection value is valid and not nil.
// Returns nil otherwise and panics, if the value is not assignable to T.
func interfaceOf[T any](value reflect.Value) T {
	if !value.IsValid() || value.IsNil() {
		var tNil T
		return tNil
	}

	return value.Interface().(T)
}
