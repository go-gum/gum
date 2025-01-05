package gum

import (
	"fmt"
	"github.com/go-gum/gum/serde"
	"net/http"
)

// FormValues parses the form parameters to a struct T.
// Works the same as QueryValues just for the requests Form
type FormValues[T any] struct {
	Value T
}

var _ = AssertFromRequest[FormValues[any]]()

func (FormValues[T]) FromRequest(r *http.Request) (FormValues[T], error) {
	form, err := Extract[Form](r)
	if err != nil {
		return FormValues[T]{}, err
	}

	target, err := serde.UnmarshalNew[T](querySourceValue{values: form.Values})
	if err != nil {
		return FormValues[T]{}, fmt.Errorf("deserialize %T: %w", target, err)
	}

	return FormValues[T]{Value: target}, nil
}

// PostFormValues parses the form parameters to a struct T.
// Works the same as QueryValues just for the requests PostForm
type PostFormValues[T any] struct {
	Value T
}

var _ = AssertFromRequest[PostFormValues[any]]()

func (PostFormValues[T]) FromRequest(r *http.Request) (PostFormValues[T], error) {
	form, err := Extract[PostForm](r)
	if err != nil {
		return PostFormValues[T]{}, err
	}

	target, err := serde.UnmarshalNew[T](querySourceValue{values: form.Values})
	if err != nil {
		return PostFormValues[T]{}, fmt.Errorf("deserialize %T: %w", target, err)
	}

	return PostFormValues[T]{Value: target}, nil
}
