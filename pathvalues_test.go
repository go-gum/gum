package gum

import (
	. "github.com/go-gum/gum/internal/test"
	"net/http"
	"testing"
)

func TestPathValues(t *testing.T) {
	req := &http.Request{}
	req.SetPathValue("Name", "Albert")
	req.SetPathValue("Age", "21")

	type ValueStruct struct {
		Name string
		Age  int
	}

	var extractedValue ValueStruct
	Handler(func(v PathValues[ValueStruct]) { extractedValue = v.Value }).ServeHTTP(nil, req)
	AssertEqual(t, extractedValue, ValueStruct{Name: "Albert", Age: 21})
}
