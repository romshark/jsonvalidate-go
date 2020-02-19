package main

import "sync"

// ContainerType represents the type of a container
type ContainerType byte

// Container types
const (
	_ ContainerType = iota
	ContainerDict
	ContainerArray
)

type stackLayer struct {
	numElements   int
	containerType ContainerType
}

type stack struct {
	elements  []stackLayer
	endOffset int
}

// Top returns the current top level stack
func (s *stack) Top() (
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

// Push pushes a new stack on top of
// the current top level stack
func (s *stack) Push(o ContainerType) {
	newLayer := stackLayer{
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

// PushElement increments the number of
// elements of the current top level stack
func (s *stack) PushElement() {
	s.elements[s.endOffset-1].numElements++
}

// Pop pops the current top level stack
func (s *stack) Pop() bool {
	if s.endOffset > 0 {
		s.endOffset--
		return true
	}
	return false
}

type tStackPool struct {
	maxInitStackLen int
	p               sync.Pool
}

func newStackPool(maxInitStackLen int) *tStackPool {
	return &tStackPool{
		maxInitStackLen: maxInitStackLen,
		p: sync.Pool{
			New: func() interface{} {
				return &stack{
					elements: make([]stackLayer, maxInitStackLen),
				}
			},
		},
	}
}

// acquire acquires and returns a new stack which must be released later
func (p *tStackPool) acquire() *stack { return p.p.Get().(*stack) }

// release returns the stack back to the pool
func (p *tStackPool) release(s *stack) {
	// Reset released stack
	s.endOffset = 0

	// Reset stack length if necessary
	if len(s.elements) > p.maxInitStackLen {
		s.elements = make([]stackLayer, p.maxInitStackLen)
	}

	p.p.Put(s)
}

// Err represents a parser error
type Err struct {
	DebugCode int
	Offset    int
}

// Parser represents a JSON parser
type Parser struct {
	stackPool *tStackPool
}

// NewParser creates a new parser instance
func NewParser(maxInitStackLen int) *Parser {
	if maxInitStackLen < 1 {
		maxInitStackLen = 1024 * 64 // By default: 1 MiB
	}
	return &Parser{
		stackPool: newStackPool(maxInitStackLen),
	}
}

// Parse verify and compact tries to verify
// and compact the given code
func (pr *Parser) Parse(in []byte, compact bool) (out []byte, err Err) {
	var (
		current        byte
		offsetRd       int
		offsetWr       int
		containerType  ContainerType
		containerLevel int
		elementIndex   int
		debugCode      int
	)

	// Non-compacting scanner, scans the next non-space symbol
	Scan := func() bool {
		for {
			if offsetRd >= len(in) {
				// EOF
				return true
			}
			current = in[offsetRd]
			offsetRd++
			switch current {
			case ' ':
			case '\t':
			case '\n':
			case '\r':
			default:
				return false
			}
		}
	}

	// Non-compacting string scanner, scans a string after '"'
	ScanString := func() (debugCode int) {
		inEscaped := false
		for {
			if offsetRd >= len(in) {
				// Unexpected EOF
				return 17
			}
			current = in[offsetRd]
			offsetRd++
			switch {
			case inEscaped:
				switch current {
				case 'r':
				case 'n':
				case 't':
				case '\\':
				case '"':
				default:
					// Illegal escape
					return 18
				}
				inEscaped = false

			case current == '\\':
				inEscaped = true

			case current == '"':
				// String termination
				return 0
			}
		}
	}

	if compact {
		// Compaction enabled
		Scan = func() bool {
			for {
				if offsetRd >= len(in) {
					// EOF
					return true
				}
				current = in[offsetRd]
				offsetRd++
				switch current {
				// Skip spaces
				case ' ':
				case '\n':
				case '\r':
				case '\t':
				default:
					// Write non-space runes
					in[offsetWr] = current
					offsetWr++
					return false
				}
			}
		}

		// Compacting string scanner, scans a string after '"'
		ScanString = func() (debugCode int) {
			inEscaped := false
			for {
				if offsetRd >= len(in) {
					// Unexpected EOF
					return 17
				}
				current = in[offsetRd]
				offsetRd++
				switch {
				case inEscaped:
					switch current {
					case 'r':
					case 'n':
					case 't':
					case '\\':
					case '"':
					default:
						// Illegal escape
						return 18
					}
					inEscaped = false

				case current == '\\':
					inEscaped = true

				case current == '"':
					// String termination
					in[offsetWr] = current
					offsetWr++
					return 0
				}
				in[offsetWr] = current
				offsetWr++
			}
		}
	}

	stack := pr.stackPool.acquire()
	defer pr.stackPool.release(stack)

	// Scan dict
	if Scan() {
		debugCode = 1
		goto ERR
	}

	if current != '{' {
		debugCode = 2
		goto ERR
	}
	stack.Push(ContainerDict)
	containerLevel++
	containerType = ContainerDict

ScanElements:
	// Scan elements
	for {
		if Scan() {
			goto EOF
		}

		containerType, elementIndex, containerLevel = stack.Top()
		switch containerType {
		case ContainerDict:
			// In dictionary
			switch {
			case current == '}':
				// Dict termination
				if !stack.Pop() {
					// No container to terminate
					debugCode = 7
					goto ERR
				}
				containerType, elementIndex, containerLevel = stack.Top()
				goto ScanElements

			case elementIndex > 0:
				// Parse subsequent dictionary element value
				switch {
				case current == ']':
					// Array termination
					if !stack.Pop() {
						// No container to terminate
						debugCode = 14
						goto ERR
					}
				case current == ',':
					if Scan() {
						debugCode = 21
						goto ERR
					}
				default:
					debugCode = 15
					goto ERR
				}

			case current != '"':
				// Unexpected token, expected field initializer
				debugCode = 3
				goto ERR
			}

			// Scan field name
			if debugCode = ScanString(); debugCode != 0 {
				goto ERR
			}

			if Scan() {
				// Unexpected EOF
				debugCode = 13
				goto ERR
			}
			if current != ':' {
				// Unexpected token
				debugCode = 5
				goto ERR
			}

			// Scan field value
			if Scan() {
				// Unexpected EOF
				debugCode = 6
				goto ERR
			}

		case ContainerArray:
			// In array
			switch {
			case current == ']':
				// Array termination
				if !stack.Pop() {
					// No container to terminate
					debugCode = 14
					goto ERR
				}
				containerType, elementIndex, containerLevel = stack.Top()
				goto ScanElements

			case elementIndex > 0:
				// Parse subsequent array element value
				switch {
				case current == ']':
					// Array termination
					if !stack.Pop() {
						// No container to terminate
						debugCode = 14
						goto ERR
					}
				case current == ',':
					if Scan() {
						debugCode = 21
						goto ERR
					}
				default:
					debugCode = 15
					goto ERR
				}
			}
		}

		// Parse value
		switch {
		case current == '"':
			stack.PushElement()
			// String value
			if debugCode = ScanString(); debugCode != 0 {
				goto ERR
			}

		case current == 'n':
			// Null
			stack.PushElement()
			goto ScanNull

		case current == '[':
			// Array
			stack.PushElement()
			stack.Push(ContainerArray)
			goto ScanElements

		case current == '{':
			// Dictionary
			stack.PushElement()
			stack.Push(ContainerDict)
			goto ScanElements

		case current == 't':
			// Boolean (true)
			stack.PushElement()
			goto ScanBooleanTrue

		case current == 'f':
			// Boolean (false)
			stack.PushElement()
			goto ScanBooleanFalse

		case current >= '0' && current <= '9':
			// Number
			stack.PushElement()
			// TODO: add support for number values
			panic("not yet supported 5")

		default:
			debugCode = 20
			goto ERR
		}
	}

ScanBooleanTrue:
	// Start scanning after a 't'
	if offsetRd+2 >= len(in) ||
		in[offsetRd] != 'r' ||
		in[offsetRd+1] != 'u' ||
		in[offsetRd+2] != 'e' {
		// EOF before end
		debugCode = 22
		goto ERR
	}
	if compact {
		in[offsetWr] = 'r'
		in[offsetWr+1] = 'u'
		in[offsetWr+2] = 'e'
		offsetWr += 3
	}
	offsetRd += 3
	goto ScanElements

ScanBooleanFalse:
	// Start scanning after an 'f'
	if offsetRd+3 >= len(in) ||
		in[offsetRd] != 'a' ||
		in[offsetRd+1] != 'l' ||
		in[offsetRd+2] != 's' ||
		in[offsetRd+3] != 'e' {
		// EOF before end
		debugCode = 23
		goto ERR
	}
	if compact {
		in[offsetWr] = 'a'
		in[offsetWr+1] = 'l'
		in[offsetWr+2] = 's'
		in[offsetWr+3] = 'e'
		offsetWr += 4
	}
	offsetRd += 4
	goto ScanElements

ScanNull:
	// Start scanning after a 'n'
	if offsetRd+3 > len(in) ||
		in[offsetRd] != 'u' ||
		in[offsetRd+1] != 'l' ||
		in[offsetRd+2] != 'l' {
		// EOF before end
		debugCode = 24
		goto ERR
	}
	if compact {
		in[offsetWr] = 'u'
		in[offsetWr+1] = 'l'
		in[offsetWr+2] = 'l'
		offsetWr += 3
	}
	offsetRd += 3
	goto ScanElements

EOF:
	if containerLevel > 0 {
		debugCode = 8
		goto ERR
	}
	if compact {
		out = in[:offsetWr]
	} else {
		out = in
	}
	return

ERR:
	if offsetRd > 0 {
		offsetRd--
	}
	err = Err{
		DebugCode: debugCode,
		Offset:    offsetRd,
	}
	return
}
