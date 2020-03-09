package main

import (
	"fmt"
	"strconv"
	"strings"
	"unsafe"

	"github.com/romshark/jsonvalidate-go/internal/stack"
)

// Err represents a parser error
type Err struct {
	DebugCode int
	Offset    int
}

func (err Err) Error() string {
	return fmt.Sprintf(
		"error (%d) at offset %d",
		err.DebugCode,
		err.Offset,
	)
}

// Options defines validation options
type Options struct {
	ExpectDocument     bool
	AllowDuplicateKeys bool
}

// Parser represents a JSON parser
type Parser struct {
	stackPool *stack.Pool
}

// NewParser creates a new parser instance
func NewParser(maxInitStackLen int) *Parser {
	if maxInitStackLen < 1 {
		maxInitStackLen = 1024 * 64 // By default: 1 MiB
	}
	return &Parser{
		stackPool: stack.NewPool(maxInitStackLen),
	}
}

// ValidateBytes validates a JSON value from the given byte slice
func (pr *Parser) ValidateBytes(s []byte, opts Options) Err {
	return pr.validate(b2s(s), opts)
}

// Validate validates a JSON value from the given string
func (pr *Parser) Validate(s string, opts Options) Err {
	return pr.validate(s, opts)
}

// validate validates the given document
func (pr *Parser) validate(
	input string,
	opts Options,
) (err Err) {
	var (
		containerType  stack.ContainerType
		containerLevel int
		elementIndex   int
		sv             string
		s              = input
	)

	stk := pr.stackPool.Acquire(
		// Tell the stack to keep track of the keys
		!opts.AllowDuplicateKeys,
	)
	defer pr.stackPool.Release(stk)

	currentOffset := func() int { return len(input) - len(s) }
	error := func(debugCode int) Err {
		return Err{
			DebugCode: debugCode,
			Offset:    currentOffset(),
		}
	}

	if opts.ExpectDocument {
		// Scan object
		s = skipWS(s)
		if len(s) == 0 {
			return error(1)
		}

		if s[0] != '{' {
			return error(1)
		}
		stk.Push(stack.Object)
		containerLevel++
		containerType = stack.Object
		s = s[1:]
	}

	// Check for premature EOF
	s = skipWS(s)
	if len(s) == 0 {
		return error(67)
	}

	// Scan elements
	for {
		s = skipWS(s)
		if len(s) == 0 {
			if containerLevel > 0 {
				return error(8)
			}
			return
		}

		containerType, elementIndex, containerLevel = stk.Top()

		switch containerType {
		case stack.Object:
			// In object
			switch {
			case s[0] == '}':
				// Object termination
				if !stk.Pop() {
					// No container to terminate
					return error(7)
				}
				containerType, elementIndex, containerLevel = stk.Top()
				s = s[1:]
				continue
			case elementIndex > 0:
				// Parse subsequent object element value
				if s[0] != ',' {
					return error(16)
				}
				s = s[1:]
			}

			// Scan field name
			s = skipWS(s)
			if len(s) == 0 || s[0] != '"' {
				// Unexpected token, expected field initializer
				return error(21)
			}

			err.Offset = currentOffset()
			sv, s, err.DebugCode = scanKey(s[1:])
			if err.DebugCode != 0 {
				return
			}
			// Check key length
			if len(sv) < 1 {
				err.DebugCode = 78
				return
			}

			// Scan the key for control chars.
			for i := 0; i < len(sv); i++ {
				if sv[i] < 0x20 {
					err.DebugCode = 29
					return
				}
			}

			if !opts.AllowDuplicateKeys {
				// Check for duplicate keys
				if !stk.PushField(sv) {
					err.DebugCode = 91
					return
				}
			} else {
				// Ignore duplicate keys
				stk.PushElement()
			}

			// Scan ':'
			s = skipWS(s)
			if len(s) == 0 {
				// Unexpected EOF
				return error(13)
			}
			if s[0] != ':' {
				// Unexpected token
				return error(5)
			}
			s = s[1:]

			// Scan field value

		case stack.Array:
			// In array
			switch {
			case s[0] == ']':
				// Array termination
				if !stk.Pop() {
					// No container to terminate
					return error(14)
				}
				containerType, elementIndex, containerLevel = stk.Top()
				s = s[1:]
				continue
			case elementIndex > 0:
				// Parse subsequent array element value
				if s[0] != ',' {
					return error(15)
				}
				s = s[1:]
			}

			// Push a new element onto the current stack object
			stk.PushElement()

		default:
			// Void
			if s[0] == ',' {
				// Unexpected element separator
				return error(61)
			}
		}

		// Parse value
		s = skipWS(s)
		if len(s) == 0 {
			return error(50)
		}

		switch s[0] {
		case '"':
			// String value
			err.Offset = currentOffset()
			sv, s, err.DebugCode = scanString(s[1:])
			if err.DebugCode != 0 {
				return
			}
			continue

		case 'n':
			// Null
			if len(s) < len("null") || s[:len("null")] != "null" {
				return error(24)
			}
			s = s[len("null"):]
			continue

		case '[':
			// Array
			stk.Push(stack.Array)
			s = s[1:]
			continue

		case '{':
			// Object
			stk.Push(stack.Object)
			s = s[1:]
			continue

		case 't':
			// Boolean (true)
			if len(s) < len("true") || s[:len("true")] != "true" {
				return error(22)
			}
			s = s[len("true"):]
			continue

		case 'f':
			// Boolean (false)
			if len(s) < len("false") || s[:len("false")] != "false" {
				return error(23)
			}
			s = s[len("false"):]
			continue

		case '-', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
			// Number
			err.Offset = currentOffset()
			s, err.DebugCode = scanNumber(s)
			if err.DebugCode != 0 {
				return
			}

		default:
			return error(20)
		}
	}
}

func skipWS(s string) string {
	if len(s) == 0 || s[0] > 0x20 {
		// Fast path.
		return s
	}
	return skipWSSlow(s)
}

func skipWSSlow(s string) string {
	if len(s) == 0 || s[0] != 0x20 && s[0] != 0x0A && s[0] != 0x09 && s[0] != 0x0D {
		return s
	}
	for i := 1; i < len(s); i++ {
		if s[i] != 0x20 && s[i] != 0x0A && s[i] != 0x09 && s[i] != 0x0D {
			return s[i:]
		}
	}
	return ""
}

// scanKey is similar to scanString, but is optimized
// for typical object keys, which are quite small and have no escape sequences.
func scanKey(s string) (string, string, int) {
	for i := 0; i < len(s); i++ {
		if s[i] == '"' {
			// Fast path - the key doesn't contain escape sequences.
			return s[:i], s[i+1:], 0
		}
		if s[i] == '\\' {
			// Slow path - the key contains escape sequences.
			return scanString(s)
		}
	}
	// Missing closing "
	return "", s, 800
}

func scanString(s string) (string, string, int) {
	// Try fast path - a string without escape sequences.
	if n := strings.IndexByte(s, '"'); n >= 0 && strings.IndexByte(s[:n], '\\') < 0 {
		return s[:n], s[n+1:], 0
	}

	// Slow path - escape sequences are present.
	rs, tail, errCode := scanRawString(s)
	if errCode != 0 {
		return rs, tail, errCode
	}
	for {
		n := strings.IndexByte(rs, '\\')
		if n < 0 {
			return rs, tail, 0
		}
		n++
		if n >= len(rs) {
			return rs, tail, 500
		}
		ch := rs[n]
		rs = rs[n+1:]
		switch ch {
		case '"', '\\', '/', 'b', 'f', 'n', 'r', 't':
			// Valid escape sequences - see http://json.org/
			break
		case 'u':
			if len(rs) < 4 {
				// Escape sequence too short
				return rs, tail, 400
			}
			xs := rs[:4]
			_, err := strconv.ParseUint(xs, 16, 16)
			if err != nil {
				// Invalid escape sequence
				return rs, tail, 401
			}
			rs = rs[4:]
		default:
			// Unknown escape sequence
			return rs, tail, 402
		}
	}
}

func scanNumber(s string) (string, int) {
	if len(s) == 0 {
		// zero-length number
		return s, 700
	}
	if s[0] == '-' {
		s = s[1:]
		if len(s) == 0 {
			// missing number after minus
			return s, 701
		}
	}
	i := 0
	for i < len(s) {
		if s[i] < '0' || s[i] > '9' {
			break
		}
		i++
	}
	if i <= 0 {
		// non 0..9 digit
		return s, 702
	}
	if s[0] == '0' && i != 1 {
		// unexpected number starting from 0
		return s, 703
	}
	if i >= len(s) {
		return "", 0
	}
	if s[i] == '.' {
		// Validate fractional part
		s = s[i+1:]
		if len(s) == 0 {
			// Missing fractional part
			return s, 704
		}
		i = 0
		for i < len(s) {
			if s[i] < '0' || s[i] > '9' {
				break
			}
			i++
		}
		if i == 0 {
			// Expecting 0..9 digit in fractional part
			return s, 705
		}
		if i >= len(s) {
			return "", 0
		}
	}
	if s[i] == 'e' || s[i] == 'E' {
		// Validate exponent part
		s = s[i+1:]
		if len(s) == 0 {
			// Missing exponent part
			return s, 706
		}
		if s[0] == '-' || s[0] == '+' {
			s = s[1:]
			if len(s) == 0 {
				// Missing exponent part
				return s, 707
			}
		}
		i = 0
		for i < len(s) {
			if s[i] < '0' || s[i] > '9' {
				break
			}
			i++
		}
		if i == 0 {
			// Expecting 0..9 digit in exponent part
			return s, 708
		}
		if i >= len(s) {
			return "", 0
		}
	}
	return s[i:], 0
}

func scanRawString(s string) (string, string, int) {
	n := strings.IndexByte(s, '"')
	if n < 0 {
		// Missing closing "
		return s, "", 600
	}
	if n == 0 || s[n-1] != '\\' {
		// Fast path. No escaped ".
		return s[:n], s[n+1:], 0
	}

	// Slow path - possible escaped " found.
	ss := s
	for {
		i := n - 1
		for i > 0 && s[i-1] == '\\' {
			i--
		}
		if uint(n-i)%2 == 0 {
			return ss[:len(ss)-len(s)+n], s[n+1:], 0
		}
		s = s[n+1:]

		n = strings.IndexByte(s, '"')
		if n < 0 {
			// Missing closing "
			return ss, "", 601
		}
		if n == 0 || s[n-1] != '\\' {
			return ss[:len(ss)-len(s)+n], s[n+1:], 0
		}
	}
}

// Ensure len(s) > 0 before calling
func scanRawNumber(s string) (string, string, int) {

	// Find the end of the number.
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if (ch >= '0' && ch <= '9') || ch == '.' || ch == '-' || ch == 'e' || ch == 'E' || ch == '+' {
			continue
		}
		if i == 0 || i == 1 && (s[0] == '-' || s[0] == '+') {
			if len(s[i:]) >= 3 {
				xs := s[i : i+3]
				if strings.EqualFold(xs, "inf") || strings.EqualFold(xs, "nan") {
					return s[:i+3], s[i+3:], 0
				}
			}
			// Unexpected char
			return "", s, 900
		}
		ns := s[:i]
		s = s[i:]
		return ns, s, 0
	}
	return s, "", 0
}

func b2s(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}
