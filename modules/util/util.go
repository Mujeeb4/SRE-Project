// Copyright 2017 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package util

import (
	"bytes"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"os"
	"strconv"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
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

// CryptoRandomInt returns a crypto random integer between 0 and limit, inclusive
func CryptoRandomInt(limit int64) (int64, error) {
	rInt, err := rand.Int(rand.Reader, big.NewInt(limit))
	if err != nil {
		return 0, err
	}
	return rInt.Int64(), nil
}

const alphanumericalChars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// CryptoRandomString generates a crypto random alphanumerical string, each byte is generated by [0,61] range
func CryptoRandomString(length int64) (string, error) {
	buf := make([]byte, length)
	limit := int64(len(alphanumericalChars))
	for i := range buf {
		num, err := CryptoRandomInt(limit)
		if err != nil {
			return "", err
		}
		buf[i] = alphanumericalChars[num]
	}
	return string(buf), nil
}

// CryptoRandomBytes generates `length` crypto bytes
// This differs from CryptoRandomString, as each byte in CryptoRandomString is generated by [0,61] range
// This function generates totally random bytes, each byte is generated by [0,255] range
func CryptoRandomBytes(length int64) ([]byte, error) {
	buf := make([]byte, length)
	_, err := rand.Read(buf)
	return buf, err
}

// ToUpperASCII returns s with all ASCII letters mapped to their upper case.
func ToUpperASCII(s string) string {
	b := []byte(s)
	for i, c := range b {
		if 'a' <= c && c <= 'z' {
			b[i] -= 'a' - 'A'
		}
	}
	return string(b)
}

var (
	titleCaser        = cases.Title(language.English)
	titleCaserNoLower = cases.Title(language.English, cases.NoLower)
)

// ToTitleCase returns s with all english words capitalized
func ToTitleCase(s string) string {
	return titleCaser.String(s)
}

// ToTitleCaseNoLower returns s with all english words capitalized without lowercasing
func ToTitleCaseNoLower(s string) string {
	return titleCaserNoLower.String(s)
}

func logError(msg string, args ...any) {
	// TODO: the "util" package can not import the "modules/log" package, so we use the "fmt" package here temporarily.
	// In the future, we should decouple the dependency between them.
	_, _ = fmt.Fprintf(os.Stderr, msg, args...)
}

// ToInt64 transform a given int into int64.
func ToInt64(number interface{}) int64 {
	var value int64
	switch v := number.(type) {
	case int:
		value = int64(v)
	case int8:
		value = int64(v)
	case int16:
		value = int64(v)
	case int32:
		value = int64(v)
	case int64:
		value = v
	case uint:
		value = int64(v)
	case uint8:
		value = int64(v)
	case uint16:
		value = int64(v)
	case uint32:
		value = int64(v)
	case uint64:
		value = int64(v)
	case string:
		var err error
		if value, err = strconv.ParseInt(v, 10, 64); err != nil {
			logError("strconv.ParseInt failed for %q: %v", v, err)
		}
	default:
		logError("unable to convert %q to int64", v)
	}
	return value
}
