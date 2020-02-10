package main

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/valyala/fastjson"
)

var smValid = []byte(
	`{"foo":{"baar":{"bazz":[null,["fuzzz"],true]}}, "kraz": "nazzz"}`,
)

var md1Invalid = []byte(
	`{"x":[[[[[[[[[[[[[[[[[[[[[["y]]]]]]]]]]]]]]]]]]]]]]}`,
)

var smInvalid = []byte(
	`{"foo":{"baar":{"bazz":[null,["fuzzz"],true]}}, "kraz": "nazzz"}}`,
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
		{"small", smValid},
		{"medium", []byte(mdValid)},
		// {"large", []byte(lgValid)}, //TODO: implement numbers
	}

	testSuites := []struct {
		nm string
		fn func(in []byte)
	}{
		{"jsonvalidate", func(in []byte) {
			_, ge1 = gParser.Parse(in, false)
			if ge1.DebugCode != 0 {
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
		{"small", smInvalid},
		{"medium", []byte(mdInvalid)},
		{"md1Invalid", []byte(md1Invalid)},
	}

	testSuites := []struct {
		nm string
		fn func(in []byte)
	}{
		{"jsonvalidate", func(in []byte) {
			_, ge1 = gParser.Parse(in, false)
			if ge1.DebugCode == 0 {
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
