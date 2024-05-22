package ocitar

import (
	"archive/tar"
	"io"
	"os"
)

type Opener func() (io.ReadCloser, error)

type FileOpener interface {
	Open(filename string) (io.ReadCloser, error)
}

type tarReader struct {
	opener Opener
}

func (i *tarReader) Open(filename string) (io.ReadCloser, error) {
	f, err := i.opener()
	if err != nil {
		return nil, err
	}

	tr := tar.NewReader(f)

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		if hdr.Name == filename {
			return &readCloser{
				Reader: tr,
				close:  f.Close,
			}, nil
		}
	}

	_ = f.Close()
	return nil, os.ErrNotExist
}
