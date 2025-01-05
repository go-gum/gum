package gum

import (
	. "github.com/go-gum/gum/internal/test"
	"net/http"
	"testing"
)

func TestQueryValues(t *testing.T) {
	req, _ := http.NewRequest("GET", "/example?name=Albert&age=21&tags=foo&tags=bar&n[]=1&n[]=2", nil)

	type ValueStruct struct {
		Name string   `json:"name"`
		Age  int      `json:"age"`
		Tags []string `json:"tags"`
		N    []int    `json:"n"`
	}

	var extractedValue ValueStruct
	Handler(func(v QueryValues[ValueStruct]) { extractedValue = v.Value }).ServeHTTP(nil, req)
	AssertEqual(t, extractedValue, ValueStruct{Name: "Albert", Age: 21, Tags: []string{"foo", "bar"}, N: []int{1, 2}})
}
