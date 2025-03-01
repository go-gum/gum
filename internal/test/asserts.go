package test

import (
	"reflect"
	"testing"
)

func AssertEqual[T any](t *testing.T, actual, expected T) {
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("expected\n%#v\nto equal\n%#v", actual, expected)
	}
}

func AssertNotEqual[T comparable](t *testing.T, actual, expected T) {
	if actual == expected {
		t.Fatalf("expected\n%#v\nto not equal\n%#v", actual, expected)
	}
}

func AssertTrue(t *testing.T, actual bool) {
	if !actual {
		t.Fatal("expected value to be true")
	}
}
