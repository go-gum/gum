package serde

import (
	"fmt"
	. "github.com/go-gum/gum/internal/test"
	"iter"
	"net"
	"reflect"
	"strings"
	"testing"
)

func TestUnmarshalStruct(t *testing.T) {
	type Address struct {
		City    string
		ZipCode int32 `json:"zip"`
	}

	type Student struct {
		Name       string
		AgeInYears int64  `json:"age"`
		SkipThis   string `json:"-"`
		Tags       Tags
		Address    *Address
	}

	sourceValue := dummySourceValue{
		Path: "$",

		Values: map[string]any{
			"$.Name": "Albert",
			"$.age":  int64(21),

			"$.Tags":         "foo,bar",
			"$.Address.City": "Zürich",
			"$.Address.zip":  int64(8015),

			// should not be used
			"$.SkipThis": "FOOBAR",
			"$.-":        "FOOBAR",
		},
	}

	stud, err := UnmarshalNew[Student](sourceValue)
	AssertEqual(t, err, nil)
	AssertEqual(t, stud, Student{
		Name:       "Albert",
		AgeInYears: 21,
		Tags:       Tags{"foo", "bar"},
		Address: &Address{
			City:    "Zürich",
			ZipCode: 8015,
		},
	})
}

type Tags []string

func (t *Tags) UnmarshalText(text []byte) error {
	*t = strings.Split(string(text), ",")
	return nil
}

func TestSetter(t *testing.T) {
	studentSource := dummySourceValue{}

	// get a string setter
	nameSetter, _ := setterOf(inConstructionTypes{}, reflect.TypeFor[string]())

	// get the SourceValue for the name of our student
	nameSource, _ := studentSource.Get("name")

	// apply the setter
	var name string
	var nameValue = reflect.ValueOf(&name).Elem()
	_ = nameSetter(nameSource, nameValue)

	fmt.Println(name)
}

func TestUnmarshalIP(t *testing.T) {
	studentSource := dummySourceValue{
		Values: map[string]any{
			".Host": "127.0.0.1",
			".Port": int64(80),
		},
	}

	type Host struct {
		Host net.IP
		Port *int
	}

	http := 80

	value, err := UnmarshalNew[Host](studentSource)
	AssertEqual(t, err, nil)
	AssertEqual(t, value, Host{
		Host: net.IPv4(127, 0, 0, 1),
		Port: &http,
	})
}

func TestUnmarshalGitCommit(t *testing.T) {
	type GitCommit struct {
		Sha1   string
		Parent *GitCommit
	}

	sourceValue := dummySourceValue{
		Values: map[string]any{
			".Sha1":                 "aaaa",
			".Parent.Sha1":          "bbbb",
			".Parent.Parent.Sha1":   "cccc",
			".Parent.Parent.Parent": nil,
		},
	}

	value, err := UnmarshalNew[GitCommit](sourceValue)
	AssertEqual(t, err, nil)
	AssertEqual(t, value, GitCommit{
		Sha1: "aaaa",
		Parent: &GitCommit{
			Sha1: "bbbb",
			Parent: &GitCommit{
				Sha1:   "cccc",
				Parent: nil,
			},
		},
	})
}

func TestUnmarshalSliceValue(t *testing.T) {
	type Article struct {
		Text string
		Tags []string
	}

	sourceValue := dummySourceValue{
		Values: map[string]any{
			".Text": "some long text",
			".Tags": []string{
				"first",
				"second",
				"third",
			},
		},
	}

	value, err := UnmarshalNew[Article](sourceValue)
	AssertEqual(t, err, nil)
	AssertEqual(t, value, Article{
		Text: "some long text",
		Tags: []string{
			"first",
			"second",
			"third",
		},
	})
}

func TestUnmarshalArrayValue(t *testing.T) {
	sourceValue := dummySourceValue{
		Values: map[string]any{
			"": []string{
				"first",
				"second",
				"third",
			},
		},
	}

	tags4, err := UnmarshalNew[[4]string](sourceValue)
	AssertEqual(t, err, nil)
	AssertEqual(t, tags4, [4]string{"first", "second", "third", ""})

	tags2, err := UnmarshalNew[[2]string](sourceValue)
	AssertEqual(t, err, nil)
	AssertEqual(t, tags2, [2]string{"first", "second"})
}

type dummySourceValue struct {
	Values map[string]any
	Path   string
}

func (d dummySourceValue) Float() (float64, error) {
	//TODO implement me
	panic("implement me")
}

func (d dummySourceValue) Bool() (bool, error) {
	panic("implement me")
}

func (d dummySourceValue) Iter() (iter.Seq[SourceValue], error) {
	if value, ok := d.Values[d.Path]; ok {
		if sliceValue, ok := value.([]string); ok {
			valuesIter := func(yield func(SourceValue) bool) {
				for _, value := range sliceValue {
					elementSource := dummySourceValue{Values: map[string]any{"": value}}
					if !yield(elementSource) {
						break
					}
				}
			}
			return valuesIter, nil
		}
	}

	return nil, ErrInvalidType
}

func (d dummySourceValue) Int() (int64, error) {
	fmt.Println("read int64:", d.Path)

	if value, ok := d.Values[d.Path]; ok {
		if intValue, ok := value.(int64); ok {
			return intValue, nil
		}

		return 0, ErrInvalidType
	}

	return 1234, nil
}

func (d dummySourceValue) String() (string, error) {
	fmt.Println("read string:", d.Path)

	if value, ok := d.Values[d.Path]; ok {
		if strValue, ok := value.(string); ok {
			return strValue, nil
		}

		return "", ErrInvalidType
	}

	return "foobar", nil
}

func (d dummySourceValue) Get(key string) (SourceValue, error) {
	fmt.Println("get child:", d.Path, key)

	path := d.Path + "." + key
	if value, ok := d.Values[path]; ok && value == nil {
		return nil, ErrNoValue
	}

	return dummySourceValue{Values: d.Values, Path: path}, nil
}
