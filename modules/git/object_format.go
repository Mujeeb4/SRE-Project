// Copyright 2023 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package git

import (
	"crypto/sha1"
	"regexp"
)

// sha1Pattern can be used to determine if a string is an valid sha
var sha1Pattern = regexp.MustCompile(`^[0-9a-f]{4,40}$`)

type ObjectFormat interface {
	String() string

	// Empty is the hash of empty git
	Empty() ObjectID
	// EmptyTree is the hash of an empty tree
	EmptyTree() ObjectID
	// FullLength is the length of the hash's hex string
	FullLength() int

	IsValid(input string) bool
	MustID(b []byte) ObjectID
	MustIDFromString(s string) ObjectID
	NewID(b []byte) (ObjectID, error)
	NewIDFromString(s string) (ObjectID, error)
	NewEmptyID() ObjectID

	NewHasher() HasherInterface
}

type Sha1ObjectFormatImpl struct{}

func (Sha1ObjectFormatImpl) String() string  { return "sha1" }
func (Sha1ObjectFormatImpl) Empty() ObjectID { return &Sha1Hash{} }
func (Sha1ObjectFormatImpl) EmptyTree() ObjectID {
	return &Sha1Hash{
		0x4b, 0x82, 0x5d, 0xc6, 0x42, 0xcb, 0x6e, 0xb9, 0xa0, 0x60,
		0xe5, 0x4b, 0xf8, 0xd6, 0x92, 0x88, 0xfb, 0xee, 0x49, 0x04,
	}
}
func (Sha1ObjectFormatImpl) FullLength() int { return 40 }
func (Sha1ObjectFormatImpl) IsValid(input string) bool {
	return sha1Pattern.MatchString(input)
}

func (Sha1ObjectFormatImpl) MustID(b []byte) ObjectID {
	var id Sha1Hash
	copy(id[0:20], b)
	return &id
}

func (h Sha1ObjectFormatImpl) MustIDFromString(s string) ObjectID {
	return MustIDFromString(h, s)
}

func (h Sha1ObjectFormatImpl) NewID(b []byte) (ObjectID, error) {
	return IDFromRaw(h, b)
}

func (h Sha1ObjectFormatImpl) NewIDFromString(s string) (ObjectID, error) {
	return genericIDFromString(h, s)
}

func (Sha1ObjectFormatImpl) NewEmptyID() ObjectID {
	return NewSha1()
}

func (h Sha1ObjectFormatImpl) NewHasher() HasherInterface {
	return &Sha1Hasher{sha1.New()}
}

var Sha1ObjectFormat ObjectFormat = Sha1ObjectFormatImpl{}

func ObjectFormatFromName(name string) ObjectFormat {
	switch name {
	case Sha1ObjectFormat.String():
		return Sha1ObjectFormat
	default:
		return nil
	}
}

func IsValidObjectFormat(name string) bool {
	return ObjectFormatFromName(name) != nil
}
