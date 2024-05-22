package ocitar

import (
	"fmt"
	
	v1 "github.com/google/go-containerregistry/pkg/v1"
)

type ErrNotFound struct {
	Digest v1.Hash
}

func (e *ErrNotFound) Error() string {
	return fmt.Sprintf("%s not found", e.Digest)
}
