package serde

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

func handleNumberErr[T any](inputValue string, value T, err error) (T, error) {
	var zeroValue T
	if errors.Is(err, strconv.ErrSyntax) {
		err := fmt.Errorf("parse number %q: %w", inputValue, err)
		return zeroValue, errors.Join(err, ErrInvalidType)
	}

	if err != nil {
		return zeroValue, err
	}

	return value, nil
}

type StringValue string

var _ IntSourceValue = StringValue("")

func (s StringValue) Int8() (int8, error) {
	intValue, err := strconv.ParseInt(string(s), 10, 8)
	return handleNumberErr(string(s), int8(intValue), err)
}

func (s StringValue) Int16() (int16, error) {
	intValue, err := strconv.ParseInt(string(s), 10, 16)
	return handleNumberErr(string(s), int16(intValue), err)
}

func (s StringValue) Int32() (int32, error) {
	intValue, err := strconv.ParseInt(string(s), 10, 32)
	return handleNumberErr(string(s), int32(intValue), err)
}

func (s StringValue) Int64() (int64, error) {
	intValue, err := strconv.ParseInt(string(s), 10, 64)
	return handleNumberErr(string(s), intValue, err)
}

func (s StringValue) Uint8() (uint8, error) {
	intValue, err := strconv.ParseUint(string(s), 10, 8)
	return handleNumberErr(string(s), uint8(intValue), err)
}

func (s StringValue) Uint16() (uint16, error) {
	intValue, err := strconv.ParseUint(string(s), 10, 16)
	return handleNumberErr(string(s), uint16(intValue), err)
}

func (s StringValue) Uint32() (uint32, error) {
	intValue, err := strconv.ParseUint(string(s), 10, 32)
	return handleNumberErr(string(s), uint32(intValue), err)
}

func (s StringValue) Uint64() (uint64, error) {
	intValue, err := strconv.ParseUint(string(s), 10, 64)
	return handleNumberErr(string(s), intValue, err)
}

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
