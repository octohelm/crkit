package internal

import (
	"context"
	"io"
)

type Opener = func(ctx context.Context) (io.ReadCloser, error)

func ReadAllFromOpener(ctx context.Context, opener Opener) ([]byte, error) {
	f, err := opener(ctx)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return io.ReadAll(f)
}
