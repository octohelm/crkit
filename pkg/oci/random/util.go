package random

import (
	randv2 "math/rand/v2"
)

func randomBytes(byteSize int64) []byte {
	b := make([]byte, byteSize)
	for i := range b {
		b[i] = byte(randv2.Uint32())
	}
	return b
}
