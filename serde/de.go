package serde

import (
	"encoding"
	"errors"
	"fmt"
	"iter"
	"reflect"
	"strconv"
	"strings"
	"sync"
)

type NotSupportedError struct {
	Type reflect.Type
}

func (n NotSupportedError) Error() string {
	return fmt.Sprintf("type %q is not supported", n.Type)
}

type ValueType int

var ErrInvalidType = errors.New("invalid type")
var ErrNoValue = errors.New("no value")

// SourceValue describes a source value that can be feed into the UnmarshalNew function.
type SourceValue interface {
	// Bool returns the current value as a bool.
	// Returns error ErrInvalidType if the value can not be represented as such.
	Bool() (bool, error)

	// Int returns the current value as an int64.
	// Returns error ErrInvalidType if the value can not be represented as such.
	Int() (int64, error)

	// Float returns the current value as a float64.
	// Returns error ErrInvalidType if the value can not be represented as such.
	Float() (float64, error)

	// String returns the current value as a string.
	// Returns error ErrInvalidType if the value can not be represented as such.
	String() (string, error)

	// Get returns a child value of this SourceValue if it exists.
	// Returns error ErrInvalidType if the current SourceValue does not have any
	// child values. If the SourceValue does have children, but just not the
	// requested child, ErrNoValue must be returned.
	Get(key string) (SourceValue, error)

	// Iter interprets the SourceValue as a slice and iterates over the
	// elements within. Returns ErrInvalidType if the SourceValue is not iterable
	Iter() (iter.Seq[SourceValue], error)
}

func Unmarshal(source SourceValue, target any) error {
	targetValue := reflect.ValueOf(target).Elem()

	// build the setter for the targets type
	setter, err := setterOf(inConstructionTypes{}, targetValue.Type())
	if err != nil {
		return err
	}

	return setter(source, targetValue)
}

func UnmarshalNew[T any](source SourceValue) (T, error) {
	var target T
	err := Unmarshal(source, &target)
	return target, err
}

// A setter sets the reflect.Value to the value extracted from the given SourceValue
type setter func(SourceValue, reflect.Value) error

var tyTextUnmarshaler = reflect.TypeFor[encoding.TextUnmarshaler]()

var cachedSetters sync.Map

type inConstructionTypes map[reflect.Type]struct{}

func setterOf(inConstruction inConstructionTypes, ty reflect.Type) (setter, error) {
	if cached, ok := cachedSetters.Load(ty); ok {
		return cached.(setter), nil
	}

	if _, ok := inConstruction[ty]; ok {
		// detected a cycle. return a setter that does a cache lookup when executed.
		// we assume that the actual setter will be in the cache once this setter is executed.
		lazySetter := func(source SourceValue, target reflect.Value) error {
			cached, _ := cachedSetters.Load(ty)
			return cached.(setter)(source, target)
		}

		return lazySetter, nil
	}

	inConstruction[ty] = struct{}{}

	setter, err := makeSetterOf(inConstruction, ty)
	if err != nil {
		return nil, err
	}

	cachedSetters.Store(ty, setter)

	return setter, nil
}

func makeSetterOf(inConstruction inConstructionTypes, ty reflect.Type) (setter, error) {
	if reflect.PointerTo(ty).Implements(tyTextUnmarshaler) {
		return setTextUnmarshaler, nil
	}

	switch ty.Kind() {
	case reflect.Bool:
		return setBool, nil

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return setInt, nil

	case reflect.Float32, reflect.Float64:
		return setFloat, nil

	case reflect.String:
		return setString, nil

	case reflect.Pointer:
		return makeSetPointer(inConstruction, ty)

	case reflect.Struct:
		return makeSetStruct(inConstruction, ty)

	case reflect.Slice:
		return makeSetSlice(inConstruction, ty)

	case reflect.Array:
		return makeSetArray(inConstruction, ty)

	default:
		return nil, NotSupportedError{Type: ty}
	}
}

func makeSetPointer(inConstruction inConstructionTypes, ty reflect.Type) (setter, error) {
	pointeeType := ty.Elem()

	pointeeSetter, err := setterOf(inConstruction, pointeeType)
	if err != nil {
		return nil, err
	}

	setter := func(source SourceValue, target reflect.Value) error {
		// newValue is now a pointer to an instance of the pointeeType
		newValue := reflect.New(pointeeType)
		if err := pointeeSetter(source, newValue.Elem()); err != nil {
			return err
		}

		// set pointer to the new value
		target.Set(newValue)

		return nil
	}

	return setter, err
}

func setBool(source SourceValue, target reflect.Value) error {
	boolValue, err := source.Bool()
	if err != nil {
		return fmt.Errorf("get bool value: %w", err)
	}

	target.SetBool(boolValue)
	return nil
}

func setInt(source SourceValue, target reflect.Value) error {
	intValue, err := source.Int()
	if err != nil {
		return fmt.Errorf("get int value: %w", err)
	}

	target.SetInt(intValue)
	return nil
}

func setFloat(source SourceValue, target reflect.Value) error {
	floatValue, err := source.Float()
	if err != nil {
		return fmt.Errorf("get float value: %w", err)
	}

	target.SetFloat(floatValue)
	return nil
}

func setString(source SourceValue, target reflect.Value) error {
	stringValue, err := source.String()
	if err != nil {
		return fmt.Errorf("get string value: %w", err)
	}

	target.SetString(stringValue)

	return nil
}

func setTextUnmarshaler(source SourceValue, target reflect.Value) error {
	text, err := source.String()
	if err != nil {
		return fmt.Errorf("get string value: %w", err)
	}

	m := target.Addr().Interface().(encoding.TextUnmarshaler)
	return m.UnmarshalText([]byte(text))
}

func makeSetStruct(inConstruction inConstructionTypes, ty reflect.Type) (setter, error) {
	var setters []setter

	fields := collectStructFields(ty)

	for _, field := range fields {
		de, err := setterOf(inConstruction, field.Type)
		if err != nil {
			return nil, fmt.Errorf("setter for field %q: %w", field.Name, err)
		}

		setters = append(setters, de)
	}

	setter := func(source SourceValue, target reflect.Value) error {
		for idx, field := range fields {
			fieldSource, err := source.Get(field.Name)
			switch {
			case errors.Is(err, ErrNoValue):
				continue
			case err != nil:
				return fmt.Errorf("lookup child %q: %w", field.Name, err)
			}

			fieldValue := target.FieldByIndex(field.Index)
			if err := setters[idx](fieldSource, fieldValue); err != nil {
				return fmt.Errorf("set field %q on %q: %w", field.Name, target.Type(), err)
			}
		}

		return nil
	}

	return setter, nil
}

func makeSetSlice(inConstruction inConstructionTypes, ty reflect.Type) (setter, error) {
	elementSetter, err := setterOf(inConstruction, ty.Elem())
	if err != nil {
		return nil, fmt.Errorf("setter for element type %q: %w", ty, err)
	}

	// a empty element
	placeholderValue := reflect.New(ty.Elem()).Elem()

	setter := func(source SourceValue, target reflect.Value) error {
		sourceIter, err := source.Iter()
		if err != nil {
			return fmt.Errorf("as iter: %w", err)
		}

		for elementSource := range sourceIter {
			// add an empty element to grow the list
			target.Set(reflect.Append(target, placeholderValue))

			idx := target.Len() - 1
			elementValue := target.Index(idx)
			if err := elementSetter(elementSource, elementValue); err != nil {
				return fmt.Errorf("set element idx=%d: %w", idx, err)
			}
		}

		return nil
	}

	return setter, nil
}

func makeSetArray(inConstruction inConstructionTypes, ty reflect.Type) (setter, error) {
	elementSetter, err := setterOf(inConstruction, ty.Elem())
	if err != nil {
		return nil, fmt.Errorf("setter for element type %q: %w", ty, err)
	}

	// number of elements in the array
	elementCount := ty.Len()

	setter := func(source SourceValue, target reflect.Value) error {
		sourceIter, err := source.Iter()
		if err != nil {
			return fmt.Errorf("as iter: %w", err)
		}

		next, stop := iter.Pull(sourceIter)
		defer stop()

		for idx := 0; idx < elementCount; idx++ {
			elementSource, ok := next()
			if !ok {
				break
			}

			elementValue := target.Index(idx)
			if err := elementSetter(elementSource, elementValue); err != nil {
				return fmt.Errorf("set element idx=%d: %w", idx, err)
			}
		}

		return nil
	}

	return setter, nil
}

type field struct {
	Name  string
	Type  reflect.Type
	Index []int
}

func collectStructFields(ty reflect.Type) []field {
	var fields []field

	// collect all fields
	for fi := range fieldsIter(ty) {
		name := nameOf(fi)

		if name == "" {
			// skip this field
			continue
		}

		fields = append(fields, field{
			Name:  name,
			Type:  fi.Type,
			Index: fi.Index,
		})
	}

	return fields
}

func nameOf(fi reflect.StructField) string {
	// the name of the field
	name := fi.Name

	// parse json struct tag to get renamed alias
	if tag := fi.Tag.Get("json"); tag != "" {
		if tag == "-" {
			// empty name, skip this field
			return ""
		}

		idx := strings.IndexByte(tag, ',')
		switch {
		case idx == -1:
			// no comma, take the full tag as name
			name = tag

		case idx > 0:
			// non emtpy alias, take up to comma
			name = tag[:idx]
		}
	}

	return name
}

func fieldsIter(ty reflect.Type) iter.Seq[reflect.StructField] {
	if ty.Kind() != reflect.Struct {
		panic("not a struct")
	}

	return func(yield func(reflect.StructField) bool) {
		for idx := range ty.NumField() {
			fi := ty.Field(idx)
			if !fi.IsExported() {
				// skip not exported field
				continue
			}

			if fi.Anonymous {
				// TODO support this
				panic(fmt.Sprintf("anonymous field %q currently not supported", fi.Name))
			}

			if !yield(fi) {
				break
			}
		}
	}
}

type StringValue string

func (s StringValue) Bool() (bool, error) {
	switch {
	case strings.EqualFold(string(s), "true"):
		return true, nil
	case strings.EqualFold(string(s), "false"):
		return false, nil
	}

	return false, ErrInvalidType
}

func (s StringValue) Int() (int64, error) {
	parsedValue, err := strconv.ParseInt(string(s), 10, 64)
	if err != nil {
		return 0, errors.Join(ErrInvalidType, err)
	}

	return parsedValue, nil
}

func (s StringValue) Float() (float64, error) {
	parsedValue, err := strconv.ParseFloat(string(s), 64)
	if err != nil {
		return 0, errors.Join(ErrInvalidType, err)
	}

	return parsedValue, nil
}

func (s StringValue) String() (string, error) {
	return string(s), nil
}

func (s StringValue) Get(key string) (SourceValue, error) {
	return nil, ErrInvalidType
}

func (s StringValue) Iter() (iter.Seq[SourceValue], error) {
	return nil, ErrInvalidType
}

type InvalidValue struct{}

func (i InvalidValue) Bool() (bool, error) {
	return false, ErrInvalidType
}

func (i InvalidValue) Int() (int64, error) {
	return 0, ErrInvalidType
}

func (i InvalidValue) Float() (float64, error) {
	return 0, ErrInvalidType
}

func (i InvalidValue) String() (string, error) {
	return "", ErrInvalidType
}

func (i InvalidValue) Get(key string) (SourceValue, error) {
	return nil, ErrInvalidType
}

func (i InvalidValue) Iter() (iter.Seq[SourceValue], error) {
	return nil, ErrInvalidType
}
