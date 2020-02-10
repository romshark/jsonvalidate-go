package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidation(t *testing.T) {
	for _, tt := range []struct {
		in     string
		offset int
	}{
		{``, 0},
		{`[]`, 0},
		{`null`, 0},
		{`42`, 0},
		{`{"foo"}`, 6},
		{`{"foo""bar"}`, 6},
		{`{"foo}`, 5},
		{`{foo:"bar"}`, 1},
		{`{"x":"y",}`, 9},

		// Invalid escape
		{`{"x\x":""}`, 4},
	} {
		t.Run(tt.in, func(t *testing.T) {
			parser := NewParser(0)
			out, err := parser.Parse([]byte(tt.in), true)
			require.NotZero(t, err.DebugCode)
			require.Zero(t, out)
			require.Equal(t, tt.offset, err.Offset)
		})
	}
}

func TestCompaction(t *testing.T) {
	for _, tt := range []struct {
		in  string
		out string
	}{
		// No compact
		{`{}`, `{}`},
		{`{"":""}`, `{"":""}`},
		{`{"foo":"bar"}`, `{"foo":"bar"}`},
		{`{"foo":{"bar":"baz"}}`, `{"foo":{"bar":"baz"}}`},

		// Array-fields
		{`{"a":[]}`, `{"a":[]}`},
		{`{"a":["a"]}`, `{"a":["a"]}`},
		{`{"a":["b","c","d"]}`, `{"a":["b","c","d"]}`},
		{`{"a":["b",true,null]}`, `{"a":["b",true,null]}`},

		// Compact
		{` { " foo " : " bar " } `, `{" foo ":" bar "}`},

		// Escape
		{`{"\rf\"o\to":"b\"a\\r"}`, `{"\rf\"o\to":"b\"a\\r"}`},
	} {
		t.Run(tt.in, func(t *testing.T) {
			parser := NewParser(0)
			out, err := parser.Parse([]byte(tt.in), true)
			require.Zero(t, err.DebugCode)
			require.Equal(t, tt.out, string(out))
		})
	}
}

func TestCompactionDisabled(t *testing.T) {
	for _, tt := range []struct {
		in  string
		out string
	}{
		{` { "foo" : "bar" } `, ` { "foo" : "bar" } `},
		{"\n{\n\"foo\"\n:\n  \"bar\"\t\t} ", "\n{\n\"foo\"\n:\n  \"bar\"\t\t} "},
	} {
		t.Run(tt.in, func(t *testing.T) {
			parser := NewParser(0)
			out, err := parser.Parse([]byte(tt.in), false)
			require.Zero(t, err.DebugCode)
			require.Equal(t, tt.out, string(out))
		})
	}
}
