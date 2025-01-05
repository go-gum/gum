package gum

import (
	"errors"
	. "github.com/go-gum/gum/internal/test"
	"reflect"
	"testing"
)

func TestNewValue(t *testing.T) {
	val := newValue(reflect.TypeFor[int]())
	AssertTrue(t, val.Equal(reflect.ValueOf(0)))

	val = newValue(reflect.TypeFor[*string]())
	AssertTrue(t, val.Elem().Equal(reflect.ValueOf("")))

	val = newValue(reflect.TypeFor[**string]())
	AssertTrue(t, val.Elem().Elem().Equal(reflect.ValueOf("")))

	val = newValue(reflect.TypeFor[***string]())
	AssertTrue(t, val.Elem().Elem().Elem().Equal(reflect.ValueOf("")))
}

func TestInterfaceOf(t *testing.T) {
	expected := errors.New("foobar")
	actual := interfaceOf[error](reflect.ValueOf(expected))
	AssertEqual(t, actual, expected)

	expected = nil
	actual = interfaceOf[error](reflect.ValueOf(expected))
	AssertEqual(t, actual, nil)

	actual = interfaceOf[error](reflect.ValueOf(nil))
	AssertEqual(t, actual, nil)

	actual = interfaceOf[error](reflect.Value{})
	AssertEqual(t, actual, nil)
}
