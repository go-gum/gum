package serde

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	. "github.com/go-gum/gum/internal/test"
	"io"
	"iter"
	"net"
	"reflect"
	"strings"
	"testing"
	"unsafe"
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

type binarySourceValue struct {
	r io.Reader
}

func (b binarySourceValue) Iter() (iter.Seq[SourceValue], error) {
	it := func(yield func(SourceValue) bool) {
		for {
			if !yield(b) {
				break
			}
		}
	}

	return it, nil
}

func (b binarySourceValue) Get(key string) (SourceValue, error) {
	return b, nil
}

func (b binarySourceValue) Bool() (bool, error) {
	var buf [1]byte
	if _, err := b.r.Read(buf[:]); err != nil {
		return false, err
	}

	return buf[0] != 0, nil
}

func (b binarySourceValue) Int() (int64, error) {
	// no support for unsized int types
	return 0, ErrInvalidType
}

func (b binarySourceValue) Float() (float64, error) {
	// no support for unsized int types
	return 0, ErrInvalidType
}

func (b binarySourceValue) String() (string, error) {
	bc, err := b.Int32()
	if err != nil {
		return "", err
	}

	buf := make([]byte, bc)
	if _, err := b.r.Read(buf); err != nil {
		return "", err
	}

	return unsafe.String(&buf[0], bc), nil
}

func (b binarySourceValue) Int8() (int8, error) {
	var buf [1]byte
	if _, err := b.r.Read(buf[:]); err != nil {
		return 0, err
	}

	return int8(buf[0]), nil
}

func (b binarySourceValue) Int16() (int16, error) {
	var buf [2]byte
	if _, err := b.r.Read(buf[:]); err != nil {
		return 0, err
	}

	return int16(binary.LittleEndian.Uint16(buf[:])), nil
}

func (b binarySourceValue) Int32() (int32, error) {
	var buf [4]byte
	if _, err := b.r.Read(buf[:]); err != nil {
		return 0, err
	}

	return int32(binary.LittleEndian.Uint32(buf[:])), nil
}

func (b binarySourceValue) Int64() (int64, error) {
	var buf [8]byte
	if _, err := b.r.Read(buf[:]); err != nil {
		return 0, err
	}

	return int64(binary.LittleEndian.Uint64(buf[:])), nil
}

func (b binarySourceValue) Uint8() (uint8, error) {
	var buf [1]byte
	if _, err := b.r.Read(buf[:]); err != nil {
		return 0, err
	}

	return buf[0], nil
}

func (b binarySourceValue) Uint16() (uint16, error) {
	var buf [2]byte
	if _, err := b.r.Read(buf[:]); err != nil {
		return 0, err
	}

	return binary.LittleEndian.Uint16(buf[:]), nil
}

func (b binarySourceValue) Uint32() (uint32, error) {
	var buf [4]byte
	if _, err := b.r.Read(buf[:]); err != nil {
		return 0, err
	}

	return binary.LittleEndian.Uint32(buf[:]), nil
}

func (b binarySourceValue) Uint64() (uint64, error) {
	var buf [8]byte
	if _, err := b.r.Read(buf[:]); err != nil {
		return 0, err
	}

	return binary.LittleEndian.Uint64(buf[:]), nil
}

func TestDecodeBitmapHeader(t *testing.T) {
	type BitmapFileHeader struct {
		Signature        [2]byte // Signature ("BM" for Bitmap files)
		FileSize         uint32  // File size in bytes
		Reserved1        uint16  // Reserved, must be zero
		Reserved2        uint16  // Reserved, must be zero
		PixelArrayOffset uint32  // Offset to the start of the pixel array
	}

	// BitmapInfoHeader represents the DIB Header (40 bytes for BITMAPINFOHEADER)
	type BitmapInfoHeader struct {
		HeaderSize      uint32 // Size of this header (40 bytes)
		Width           int32  // Bitmap width in pixels
		Height          int32  // Bitmap height in pixels
		Planes          uint16 // Number of color planes (must be 1)
		BitsPerPixel    uint16 // Bits per pixel (1, 4, 8, 16, 24, or 32)
		Compression     uint32 // Compression method (0 = BI_RGB, no compression)
		ImageSize       uint32 // Image size (may be 0 for BI_RGB)
		XPixelsPerMeter int32  // Horizontal resolution (pixels per meter)
		YPixelsPerMeter int32  // Vertical resolution (pixels per meter)
		ColorsUsed      uint32 // Number of colors in the color palette
		ImportantColors uint32 // Number of important colors used (0 = all)
	}

	type Header struct {
		File BitmapFileHeader
		Info BitmapInfoHeader
	}

	buf, _ := base64.StdEncoding.DecodeString(`Qk3GAAAAAAAAAIoAAAB8AAAAAwAAAAUAAAABABgAAAAAADwAAAAAAAAAAAAAAAAAAAAAAAAAAAD/AAD/AAD/AAAAAAAA/0JHUnOPwvUoUbgeFR6F6wEzMzMTZmZmJmZmZgaZmZkJPQrXAyhcjzIAAAAAAAAAAAAAAAAEAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA`)
	sourceValue := binarySourceValue{bytes.NewReader(buf)}

	parsed, err := UnmarshalNew[Header](sourceValue)
	AssertEqual(t, err, nil)

	expected := Header{
		File: BitmapFileHeader{Signature: [2]byte{66, 77}, FileSize: 198, Reserved1: 0, Reserved2: 0, PixelArrayOffset: 138},
		Info: BitmapInfoHeader{HeaderSize: 124, Width: 3, Height: 5, Planes: 1, BitsPerPixel: 24, Compression: 0, ImageSize: 60, XPixelsPerMeter: 0, YPixelsPerMeter: 0, ColorsUsed: 0, ImportantColors: 0},
	}

	AssertEqual(t, parsed, expected)
}
