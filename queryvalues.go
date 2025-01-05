package gum

import (
	"fmt"
	"github.com/go-gum/gum/serde"
	"iter"
	"net/http"
	"net/url"
)

// QueryValues parses the query parameters to a struct T.
// It supports multiple definitions of the same parameter for slices.
type QueryValues[T any] struct {
	Value T
}

var _ = AssertFromRequest[QueryValues[any]]()

func (QueryValues[T]) FromRequest(r *http.Request) (QueryValues[T], error) {
	target, err := serde.UnmarshalNew[T](querySourceValue{values: r.URL.Query()})
	if err != nil {
		return QueryValues[T]{}, fmt.Errorf("deserialize %T: %w", target, err)
	}

	return QueryValues[T]{Value: target}, nil
}

type querySourceValue struct {
	serde.InvalidValue
	values url.Values
}

func (p querySourceValue) Get(key string) (serde.SourceValue, error) {
	// check if we have an explicit slice for this key in the data
	if values, ok := p.values[key+"[]"]; ok {
		return stringSliceValue(values), nil
	}

	values := p.values[key]
	if len(values) == 0 {
		return nil, serde.ErrNoValue
	}

	return stringSliceValue(values), nil
}

type stringSliceValue []string

func (s stringSliceValue) Bool() (bool, error) {
	singleValue, err := singleValueOf(s)
	if err != nil {
		return false, nil
	}

	return serde.StringValue(singleValue).Bool()
}

func (s stringSliceValue) Int() (int64, error) {
	singleValue, err := singleValueOf(s)
	if err != nil {
		return 0, nil
	}

	return serde.StringValue(singleValue).Int()
}

func (s stringSliceValue) Float() (float64, error) {
	singleValue, err := singleValueOf(s)
	if err != nil {
		return 0, nil
	}

	return serde.StringValue(singleValue).Float()
}

func (s stringSliceValue) String() (string, error) {
	singleValue, err := singleValueOf(s)
	if err != nil {
		return "", nil
	}

	return serde.StringValue(singleValue).String()
}

func (s stringSliceValue) Get(key string) (serde.SourceValue, error) {
	return nil, serde.ErrInvalidType
}

func (s stringSliceValue) Iter() (iter.Seq[serde.SourceValue], error) {
	it := func(yield func(serde.SourceValue) bool) {
		for _, value := range s {
			if !yield(serde.StringValue(value)) {
				break
			}
		}
	}

	return it, nil
}

func singleValueOf(values []string) (string, error) {
	switch len(values) {
	case 0:
		return "", serde.ErrNoValue

	case 1:
		return values[0], nil

	default:
		return "", serde.ErrInvalidType
	}
}
