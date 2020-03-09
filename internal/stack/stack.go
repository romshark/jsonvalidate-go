package stack

import (
	"sync"
)

// ContainerType represents the type of a container
type ContainerType byte

// Container types
const (
	_ ContainerType = iota
	Object
	Array
)

// Layer represents a stack layer
type Layer struct {
	numElements   int
	objectKeys    map[string]struct{}
	containerType ContainerType
}

// Stack represents a stack
type Stack struct {
	elements  []Layer
	endOffset int
	trackKeys bool
}

// Top returns the current top level stack
func (s *Stack) Top() (
	containerType ContainerType,
	numElements,
	endOffset int,
) {
	if s.endOffset < 1 {
		return 0, 0, 0
	}
	x := s.elements[s.endOffset-1]
	return x.containerType, x.numElements, s.endOffset
}

// Push pushes a new stack on top of the current top level stack
func (s *Stack) Push(o ContainerType) {
	newLayer := Layer{
		containerType: o,
		numElements:   0,
	}
	if s.endOffset >= len(s.elements) {
		// Grow stack
		s.elements = append(s.elements, newLayer)
	} else {
		s.elements[s.endOffset] = newLayer
	}
	s.endOffset++
}

// PushField increments the number of elements of the current
// top level stack returning true if the field was pushed,
// otherwise returning false indicating that a field with a
// similar name was already registered
func (s *Stack) PushField(name string) bool {
	s.elements[s.endOffset-1].numElements++
	k := s.elements[s.endOffset-1].objectKeys
	if k == nil {
		s.elements[s.endOffset-1].objectKeys = map[string]struct{}{
			name: struct{}{},
		}
		return true
	} else if _, ok := k[name]; ok {
		return false
	}
	k[name] = struct{}{}
	return true
}

// PushElement increments the number of
// elements of the current top level stack
func (s *Stack) PushElement() {
	s.elements[s.endOffset-1].numElements++
}

// Pop pops the current top level stack
func (s *Stack) Pop() bool {
	if s.endOffset > 0 {
		s.endOffset--
		return true
	}
	return false
}

func (s *Stack) reset(maxInitStackLen int) {
	// Reset released stack
	s.endOffset = 0

	// Reset stack length if necessary
	if len(s.elements) > maxInitStackLen {
		s.elements = make([]Layer, maxInitStackLen)
	} else if s.trackKeys {
		for i := 0; i < len(s.elements); i++ {
			s.elements[i].objectKeys = nil
		}
	}
}

// Pool holds a pool of stacks
type Pool struct {
	maxInitStackLen int
	pool            sync.Pool
}

// NewPool creates a new stack pool instance
func NewPool(maxInitStackLen int) *Pool {
	return &Pool{
		maxInitStackLen: maxInitStackLen,
		pool: sync.Pool{
			New: func() interface{} {
				return &Stack{
					elements: make([]Layer, maxInitStackLen),
				}
			},
		},
	}
}

// Acquire acquires and returns a new stack which must be released later
func (p *Pool) Acquire(trackKeys bool) *Stack {
	s := p.pool.Get().(*Stack)
	s.trackKeys = trackKeys
	return s
}

// Release returns the stack back to the pool
func (p *Pool) Release(s *Stack) {
	s.reset(p.maxInitStackLen)
	p.pool.Put(s)
}
