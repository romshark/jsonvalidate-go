package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

type Input struct {
	Name   string
	Source string
}

func validValues() []Input {
	return []Input{
		{"null", `null`},
		{"bolean_true", `true`},
		{"boolean_false", `false`},
		{"number_integer", `4`},
		{"number_float", `42.345`},
		{"number_fE", `1.0E+2`},
		{"number_fe", `1.0e+2`},
		{"number_iE", `1E+2`},
		{"number_ie", `1e+2`},
		{"number_negative integer", `-4`},
		{"number_negative float", `-42.345`},
		{"number_negative fE", `-1.0E+2`},
		{"number_negative fe", `-1.0e+2`},
		{"number_negative iE", `-1E+2`},
		{"number_negative ie", `-1e+2`},
		{"string", `"okay"`},
		{"string_with escape sequences", `"a\r\nb\b\t\"\\\/\f\uAAAA"`},
		{"string_with spaces", "\"\r\n\t foo\""},
		{"array_empty", `[]`},
		{"array_numbers", `[1, 1.4,5]`},
		{"array_strings", `["a","b","c"]`},
		{"array_mixed", `[1, false, null, true, "okay"]`},
	}
}

func validDocuments() []Input {
	return []Input{
		{"object_empty", `{}`},
		{"object_1", `{"foo":42}`},
		{"object_1", `{"foo":"bar"}`},
		{"object_2", `{ "a" : "b", "c" : "d"}`},
		{"object_complex", `{
			"1": true,
			"2": false,
			"3": null,
			"4": 42,
			"5": 3.1415,
			"6": -3.1415,
			"7": [],
			"8": {},
			"9": {"array":[]},
			"10": "t",
			"11": "test",
			"12": [1, 54.2, "z", [[ [] ] ], {"x":"y"}]
		}`},
	}
}

func TestValidateInvalid(t *testing.T) {
	opts := Options{}

	for _, tt := range []struct {
		name   string
		in     string
		offset int
	}{
		{
			"empty input",
			``, 0,
		}, {
			"list of values",
			`true,false`, 4,
		}, {
			"missing value",
			`{"foo"}`, 6,
		},
		{
			"missing collon",
			`{"foo""bar"}`, 6,
		},
		{
			"missing closing quote on key",
			`{"foo}`, 1,
		},
		{
			"missing quotes on key",
			`{foo:"bar"}`, 1,
		},
		{
			"trailing comma after field",
			`{"x":"y",}`, 9,
		},
		{
			"trailing comma after element",
			`["x","y",]`, 9,
		},
		{
			"invalid number value",
			`{"x":123.23.2}`, 11,
		},
		{
			"invalid number value_missing digits",
			`{"x":-}`, 5,
		},

		// Invalid key
		{
			"empty key",
			`{"":""}`, 1,
		},
		{
			"key with invalid escape sequence",
			`{"x\x":""}`, 1,
		},
		{
			"key with invalid escape sequence",
			"{\"x\n\":\"\"}", 1,
		},

		// Invalid escape
		{
			"illegal escape sequence",
			`{"x":"\x"}`, 5,
		},
		{
			"illegal escape sequence_too short",
			`{"x":"\1"}`, 5,
		},
		{
			"illegal escape sequence_too short",
			`{"x":"\12"}`, 5,
		},
		{
			"illegal escape sequence_too short",
			`{"x":"\123"}`, 5,
		},

		// Invalid object
		{
			"duplicate key",
			`{"y": 1, "x": 2, "z": 3, "x": 4}`, 25,
		},
		{
			"duplicate key",
			`{"x": 1, "x": 2}`, 9,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewParser(0)
			err := parser.Validate(tt.in, opts)
			require.NotZero(t, err.DebugCode, "unexpected debug code")
			require.Equal(t, tt.offset, err.Offset, "unexpected error offset")
		})
		t.Run(tt.name+"_bytes", func(t *testing.T) {
			parser := NewParser(0)
			err := parser.ValidateBytes([]byte(tt.in), opts)
			require.NotZero(t, err.DebugCode, "unexpected debug code")
			require.Equal(t, tt.offset, err.Offset, "unexpected error offset")
		})
	}
}

func TestValidateValid(t *testing.T) {
	inputs := append(validValues(), validDocuments()...)
	for _, tt := range inputs {
		t.Run(tt.Name, func(t *testing.T) {
			parser := NewParser(0)
			err := parser.Validate(tt.Source, Options{})
			require.Zero(
				t, err.DebugCode,
				"unexpected debug code at offset: %d", err.Offset,
			)
		})
		t.Run(tt.Name+"_bytes", func(t *testing.T) {
			parser := NewParser(0)
			err := parser.ValidateBytes([]byte(tt.Source), Options{})
			require.Zero(
				t, err.DebugCode,
				"unexpected debug code at offset: %d", err.Offset,
			)
		})
	}
}

func TestValidateIgnoreDuplicateKeys(t *testing.T) {
	parser := NewParser(0)
	err := parser.Validate(`{"x":1,"x":2}`, Options{
		AllowDuplicateKeys: true,
	})
	require.Zero(
		t, err.DebugCode,
		"unexpected debug code at offset: %d", err.Offset,
	)
}

func TestValidateDocumentInvalid(t *testing.T) {
	opts := Options{
		ExpectDocument: true,
	}
	for _, tt := range validValues() {
		t.Run(tt.Name, func(t *testing.T) {
			parser := NewParser(0)
			err := parser.Validate(tt.Source, opts)
			require.NotZero(t, err.DebugCode, "unexpected debug code")
			require.Zero(t, err.Offset, "unexpected error offset")
		})
		t.Run(tt.Name+"_bytes", func(t *testing.T) {
			parser := NewParser(0)
			err := parser.ValidateBytes([]byte(tt.Source), opts)
			require.NotZero(t, err.DebugCode, "unexpected debug code")
			require.Zero(t, err.Offset, "unexpected error offset")
		})
	}
}

func TestValidateDocument(t *testing.T) {
	opts := Options{
		ExpectDocument: true,
	}

	for _, tt := range validDocuments() {
		t.Run(tt.Name, func(t *testing.T) {
			parser := NewParser(0)
			err := parser.Validate(tt.Source, opts)
			require.Zero(
				t, err.DebugCode,
				"unexpected debug code at offset: %d", err.Offset,
			)
		})
		t.Run(tt.Name+"_bytes", func(t *testing.T) {
			parser := NewParser(0)
			err := parser.ValidateBytes([]byte(tt.Source), opts)
			require.Zero(
				t, err.DebugCode,
				"unexpected debug code at offset: %d", err.Offset,
			)
		})
	}
}
