package gum

import (
	"fmt"
	"github.com/go-gum/gum/serde"
	"net/http"
)

// PathValues parses the path parameters to a struct T
type PathValues[T any] struct {
	Value T
}

var _ = AssertFromRequest[PathValues[any]]()

func (PathValues[T]) FromRequest(r *http.Request) (PathValues[T], error) {
	target, err := serde.UnmarshalNew[T](pathSourceValue{req: r})
	if err != nil {
		return PathValues[T]{}, fmt.Errorf("deserialize %T: %w", target, err)
	}

	return PathValues[T]{Value: target}, nil
}

type pathSourceValue struct {
	serde.InvalidValue
	req *http.Request
}

func (p pathSourceValue) Get(key string) (serde.SourceValue, error) {
	value := p.req.PathValue(key)
	if value == "" {
		return nil, serde.ErrNoValue
	}

	return serde.StringValue(value), nil
}
