package util

import (
	"crypto/rand"
)

// ID a unique identifier
type ID []byte

// NewID generate a new ID
func NewID() ID {
	ret := make(ID, 20)
	if _, err := rand.Read(ret); err != nil {
		panic(err)
	}
	return ret
}
