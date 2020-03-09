package main

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/valyala/fastjson"
)

var (
	minisculeValid = `{"x":2}`
	tinyValid      = `{"foo":2,"bar":"okay"}`
	smallValid     = `{
		"foo": {
			"baar": {
				"bazz": [
					null,
					[
						"fuzzz"
					],
					true,
					34.632e+2,
					42
				]
			}
		},
		"kraz": "nazzz"
	}`

	minisculeInvalid = `{"x":"y}`
	tinyInvalid      = `{"foo":2,"bar":"okay}`
	smallInvalid     = `{"foo":{"baar":{"bazz":[null,["fuzzz"],true]}}, "kraz": "nazzz"}}`

	// Stack depth: 64
	deeplyNestedInvalid = `[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[{"y`
)

var (
	ge1 Err
	ge2 bool
	ge3 error
)

var gParser = NewParser(
	0, // use defalt max init stack len
)

func BenchmarkValidateValid(b *testing.B) {
	testData := []struct {
		nm string
		in []byte
	}{
		{"ms", []byte(minisculeValid)},
		{"tn", []byte(tinyValid)},
		{"sm", []byte(smallValid)},
		{"md", []byte(mdValid)},
		{"lg", []byte(lgValid)},
	}

	testSuites := []struct {
		nm string
		fn func(in []byte)
	}{
		{"jsonvalidate", func(in []byte) {
			if ge1 = gParser.ValidateBytes(in, Options{
				AllowDuplicateKeys: true,
			}); ge1.DebugCode != 0 {
				panic(fmt.Errorf("unexpected error: %v", ge1))
			}
		}},
		{"encoding_json", func(in []byte) {
			if ge2 = json.Valid(in); !ge2 {
				panic(fmt.Errorf("unexpected result"))
			}
		}},
		{"fastjson", func(in []byte) {
			if ge3 = fastjson.ValidateBytes(in); ge3 != nil {
				panic(fmt.Errorf("unexpected error: %s", ge3))
			}
		}},
	}

	for _, b1 := range testData {
		b.Run(b1.nm, func(b *testing.B) {
			for _, b2 := range testSuites {
				b.Run(b2.nm, func(b *testing.B) {
					for i := 0; i < b.N; i++ {
						b2.fn(b1.in)
					}
				})
			}
		})
	}
}

func BenchmarkValidateInvalid(b *testing.B) {
	testData := []struct {
		nm string
		in []byte
	}{
		{"miniscule", []byte(minisculeInvalid)},
		{"tiny", []byte(tinyInvalid)},
		{"small", []byte(smallInvalid)},
		{"medium", []byte(mediumInvalid)},
		{"deeplyNested", []byte(deeplyNestedInvalid)},
	}

	testSuites := []struct {
		nm string
		fn func(in []byte)
	}{
		{"jsonvalidate", func(in []byte) {
			if ge1 = gParser.ValidateBytes(in, Options{
				AllowDuplicateKeys: true,
			}); ge1.DebugCode == 0 {
				panic(fmt.Errorf("unexpected error: %v", ge1))
			}
		}},
		{"encoding_json", func(in []byte) {
			if ge2 = json.Valid(in); ge2 {
				panic(fmt.Errorf("unexpected result"))
			}
		}},
		{"fastjson", func(in []byte) {
			if ge3 = fastjson.ValidateBytes(in); ge3 == nil {
				panic(fmt.Errorf("unexpected error: nil"))
			}
		}},
	}

	for _, b1 := range testData {
		b.Run(b1.nm, func(b *testing.B) {
			for _, b2 := range testSuites {
				b.Run(b2.nm, func(b *testing.B) {
					for i := 0; i < b.N; i++ {
						b2.fn(b1.in)
					}
				})
			}
		})
	}
}
