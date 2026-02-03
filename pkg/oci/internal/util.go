package internal

import (
	"context"
	"io"
)

type Opener = func(ctx context.Context) (io.ReadCloser, error)
