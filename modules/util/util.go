// Copyright 2017 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package util

import (
	"bytes"
	"crypto/rand"
	"errors"
	"math/big"
	"strconv"
	"strings"
)

// OptionalBool a boolean that can be "null"
type OptionalBool byte

const (
	// OptionalBoolNone a "null" boolean value
	OptionalBoolNone OptionalBool = iota
	// OptionalBoolTrue a "true" boolean value
	OptionalBoolTrue
	// OptionalBoolFalse a "false" boolean value
	OptionalBoolFalse
)

// IsTrue return true if equal to OptionalBoolTrue
func (o OptionalBool) IsTrue() bool {
	return o == OptionalBoolTrue
}

// IsFalse return true if equal to OptionalBoolFalse
func (o OptionalBool) IsFalse() bool {
	return o == OptionalBoolFalse
}

// IsNone return true if equal to OptionalBoolNone
func (o OptionalBool) IsNone() bool {
	return o == OptionalBoolNone
}

// OptionalBoolOf get the corresponding OptionalBool of a bool
func OptionalBoolOf(b bool) OptionalBool {
	if b {
		return OptionalBoolTrue
	}
	return OptionalBoolFalse
}

// OptionalBoolParse get the corresponding OptionalBool of a string using strconv.ParseBool
func OptionalBoolParse(s string) OptionalBool {
	b, e := strconv.ParseBool(s)
	if e != nil {
		return OptionalBoolNone
	}
	return OptionalBoolOf(b)
}

// Max max of two ints
func Max(a, b int) int {
	if a < b {
		return b
	}
	return a
}

// Min min of two ints
func Min(a, b int) int {
	if a > b {
		return b
	}
	return a
}

// IsEmptyString checks if the provided string is empty
func IsEmptyString(s string) bool {
	return len(strings.TrimSpace(s)) == 0
}

// NormalizeEOL will convert Windows (CRLF) and Mac (CR) EOLs to UNIX (LF)
func NormalizeEOL(input []byte) []byte {
	var right, left, pos int
	if right = bytes.IndexByte(input, '\r'); right == -1 {
		return input
	}
	length := len(input)
	tmp := make([]byte, length)

	// We know that left < length because otherwise right would be -1 from IndexByte.
	copy(tmp[pos:pos+right], input[left:left+right])
	pos += right
	tmp[pos] = '\n'
	left += right + 1
	pos++

	for left < length {
		if input[left] == '\n' {
			left++
		}

		right = bytes.IndexByte(input[left:], '\r')
		if right == -1 {
			copy(tmp[pos:], input[left:])
			pos += length - left
			break
		}
		copy(tmp[pos:pos+right], input[left:left+right])
		pos += right
		tmp[pos] = '\n'
		left += right + 1
		pos++
	}
	return tmp[:pos]
}

// MergeInto merges pairs of values into a "dict"
func MergeInto(dict map[string]interface{}, values ...interface{}) (map[string]interface{}, error) {
	for i := 0; i < len(values); i++ {
		switch key := values[i].(type) {
		case string:
			i++
			if i == len(values) {
				return nil, errors.New("specify the key for non array values")
			}
			dict[key] = values[i]
		case map[string]interface{}:
			m := values[i].(map[string]interface{})
			for i, v := range m {
				dict[i] = v
			}
		default:
			return nil, errors.New("dict values must be maps")
		}
	}

	return dict, nil
}

// RandomInt returns a random integer between 0 and limit, inclusive
func RandomInt(limit int64) (int64, error) {
	rInt, err := rand.Int(rand.Reader, big.NewInt(limit))
	if err != nil {
		return 0, err
	}
	return rInt.Int64(), nil
}

const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// RandomString generates a random alphanumerical string
func RandomString(length int64) (string, error) {
	bytes := make([]byte, length)
	limit := int64(len(letters))
	for i := range bytes {
		num, err := RandomInt(limit)
		if err != nil {
			return "", err
		}
		bytes[i] = letters[num]
	}
	return string(bytes), nil
}

// RandomBytes generates `length` bytes
// This differs from RandomString, as RandomString is limits each byte to have
// a maximum value of 63 instead of 255(max byte size)
func RandomBytes(length int64) ([]byte, error) {
	bytes := make([]byte, length)
	limit := int64(^uint8(0))
	for i := range bytes {
		num, err := RandomInt(limit)
		if err != nil {
			return []byte{}, err
		}
		bytes[i] = uint8(num)
	}
	return bytes, nil
}
