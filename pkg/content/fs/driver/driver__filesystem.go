package driver

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/octohelm/unifs/pkg/filesystem"
	"io"
	"io/fs"
	"os"
	"path"
)

func FromFileSystem(fs filesystem.FileSystem) Driver {
	return &driver{fs: fs}
}

type driver struct {
	fs filesystem.FileSystem
}

func (d *driver) Stat(ctx context.Context, path string) (filesystem.FileInfo, error) {
	return d.fs.Stat(ctx, path)
}

func (d *driver) Delete(ctx context.Context, path string) error {
	return d.fs.RemoveAll(ctx, path)
}

func (d *driver) Reader(ctx context.Context, path string) (io.ReadCloser, error) {
	return filesystem.Open(ctx, d.fs, path)
}

func (d *driver) WalkDir(ctx context.Context, path string, fn fs.WalkDirFunc) error {
	return filesystem.WalkDir(ctx, filesystem.Sub(d.fs, path), ".", fn)
}

func (w *driver) Move(ctx context.Context, oldPath string, newPath string) error {
	if err := filesystem.MkdirAll(ctx, w.fs, path.Dir(newPath)); err != nil {
		return err
	}
	return w.fs.Rename(ctx, oldPath, newPath)
}

func (d *driver) GetContent(ctx context.Context, path string) ([]byte, error) {
	f, err := d.Reader(ctx, path)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = f.Close()
	}()
	return io.ReadAll(f)
}

func (d *driver) PutContent(ctx context.Context, path string, contents []byte) error {
	writer, err := d.Writer(ctx, path, false)
	if err != nil {
		return err
	}
	defer writer.Close()
	_, err = io.Copy(writer, bytes.NewReader(contents))
	if err != nil {
		if cErr := writer.Cancel(ctx); cErr != nil {
			return errors.Join(err, cErr)
		}
		return err
	}
	return writer.Commit(ctx)
}

func (d *driver) Writer(ctx context.Context, pathname string, append bool) (FileWriter, error) {
	dir := path.Dir(pathname)
	if dir != "" {
		if err := filesystem.MkdirAll(ctx, d.fs, dir); err != nil {
			return nil, err
		}
	}

	flag := os.O_WRONLY | os.O_CREATE

	if append {
		flag |= os.O_APPEND
	} else {
		flag |= os.O_TRUNC
	}

	file, err := d.fs.OpenFile(ctx, pathname, flag, 0o666)
	if err != nil {
		return nil, err
	}

	offset := int64(0)

	if append {
		n, err := file.Seek(0, io.SeekEnd)
		if err != nil {
			_ = file.Close()
			return nil, err
		}
		offset = n
	}

	return &fileWriter{driver: d, path: pathname, file: file, written: offset, bw: bufio.NewWriter(file)}, nil
}

type fileWriter struct {
	driver  *driver
	path    string
	written int64

	file filesystem.File
	bw   *bufio.Writer

	closed    bool
	committed bool
	cancelled bool
}

func (fw *fileWriter) Write(p []byte) (int, error) {
	if fw.closed {
		return 0, fmt.Errorf("already closed")
	} else if fw.committed {
		return 0, fmt.Errorf("already committed")
	} else if fw.cancelled {
		return 0, fmt.Errorf("already cancelled")
	}

	n, err := fw.bw.Write(p)

	fw.written += int64(n)

	return n, err
}

func (fw *fileWriter) Size() int64 {
	return fw.written
}

func (fw *fileWriter) Close() error {
	if fw.closed {
		return fmt.Errorf("already closed")
	}

	if err := fw.bw.Flush(); err != nil {
		return err
	}

	if err := fw.file.Close(); err != nil {
		return err
	}

	fw.closed = true

	return nil
}

func (fw *fileWriter) Cancel(ctx context.Context) error {
	if fw.closed {
		return fmt.Errorf("already closed")
	}

	fw.cancelled = true

	_ = fw.file.Close()

	return fw.driver.Delete(ctx, fw.path)
}

func (fw *fileWriter) Commit(ctx context.Context) error {
	if fw.closed {
		return fmt.Errorf("already closed")
	} else if fw.committed {
		return fmt.Errorf("already committed")
	} else if fw.cancelled {
		return fmt.Errorf("already cancelled")
	}

	if err := fw.bw.Flush(); err != nil {
		return err
	}

	fw.committed = true

	return nil
}
