package stack

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStackResetExceedMaxLen(t *testing.T) {
	const maxInitStackLen = 8
	p := NewPool(maxInitStackLen)
	s := p.Acquire(true)

	// {"foo": <>, "bar": [[<>[<>, {"baz": [[ [ [] ] ]]} ]]]}
	s.Push(Object)
	s.PushField("foo")
	s.PushField("bar")
	s.Push(Array)
	s.Push(Array)
	s.PushElement()
	s.Push(Array)
	s.PushElement()
	s.PushElement()
	s.Push(Object)
	s.PushField("baz")
	s.Push(Array)
	s.Push(Array)
	s.Push(Array)
	s.Push(Array)

	require.Equal(t, 9, s.endOffset)
	require.Len(t, s.elements, 9)

	s.reset(maxInitStackLen)

	require.Len(t, s.elements, maxInitStackLen)
	require.Zero(t, s.endOffset)
}

func TestStackReset(t *testing.T) {
	const maxInitStackLen = 64
	p := NewPool(maxInitStackLen)
	s := p.Acquire(true)

	// {"foo": <>, "bar": [[<>[<>, {"baz": [[ [ [] ] ]]} ]]]}
	s.Push(Object)
	s.PushField("foo")
	s.PushField("bar")
	s.Push(Array)
	s.Push(Array)
	s.PushElement()
	s.Push(Array)
	s.PushElement()
	s.PushElement()
	s.Push(Object)
	s.PushField("baz")
	s.Push(Array)
	s.Push(Array)
	s.Push(Array)
	s.Push(Array)

	require.Len(t, s.elements, 64)
	require.Equal(t, 9, s.endOffset)

	s.reset(maxInitStackLen)

	require.Len(t, s.elements, 64)
	require.Zero(t, s.endOffset)

	require.True(t, s.trackKeys)
	for _, e := range s.elements {
		require.Zero(t, e.objectKeys)
	}
}
