package s3

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/rhnvrm/simples3"

	"github.com/octohelm/unifs/pkg/filesystem"
	"github.com/octohelm/unifs/pkg/filesystem/fsutil"
	"github.com/octohelm/unifs/pkg/strfmt"

	"github.com/octohelm/crkit/pkg/driver"
)

type (
	Driver     = driver.Driver
	FileWriter = driver.FileWriter
)

func FromS3Endpoint(endpoint strfmt.Endpoint) Driver {
	region := endpoint.Extra.Get("region")
	if region == "" {
		region = "us-east-1"
	}

	client := simples3.New(region, endpoint.Username, endpoint.Password)
	client.SetEndpoint(endpointURL(endpoint.Host(), endpoint.Extra.Get("insecure") == "true"))

	return &s3Driver{
		client:          client,
		bucket:          endpoint.Base(),
		prefix:          endpointPrefix(endpoint),
		region:          region,
		skipBucketCheck: endpoint.Extra.Get("skipBucketCheck") == "true",
	}
}

type s3Driver struct {
	client *simples3.S3

	bucket string
	prefix string
	region string

	skipBucketCheck bool
	ensureOnce      sync.Once
	ensureErr       error
}

var _ driver.Driver = (*s3Driver)(nil)

func (d *s3Driver) Stat(ctx context.Context, name string) (filesystem.FileInfo, error) {
	if isRoot(name) {
		return fsutil.NewDirFileInfo("/"), nil
	}
	if err := d.ensureBucket(ctx); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	info, err := d.client.FileDetails(simples3.DetailsInput{
		Bucket:    d.bucket,
		ObjectKey: d.key(name),
	})
	if err != nil {
		if isS3NotFound(err) {
			return d.statDirectory(ctx, name)
		}
		return nil, pathError("stat", name, os.ErrNotExist)
	}

	size, err := strconv.ParseInt(info.ContentLength, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("parse s3 object size %q: %w", info.ContentLength, err)
	}

	return fsutil.NewFileInfo(path.Base(name), size, parseS3Time(info.LastModified)), nil
}

func (d *s3Driver) Delete(ctx context.Context, name string) error {
	if err := d.ensureBucket(ctx); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	if !isRoot(name) {
		if err := d.deletePrefix(ctx, d.dirPrefixKey(name)); err != nil {
			return fmt.Errorf("remove s3 dir %q: %w", name, err)
		}
	}

	key := d.key(name)
	if key == "" {
		return nil
	}
	if err := d.deleteKey(ctx, key); err != nil {
		return fmt.Errorf("remove s3 object %q: %w", name, err)
	}
	return nil
}

func (d *s3Driver) Reader(ctx context.Context, name string) (io.ReadCloser, error) {
	if err := d.ensureBucket(ctx); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	r, err := d.client.FileDownload(simples3.DownloadInput{
		Bucket:    d.bucket,
		ObjectKey: d.key(name),
	})
	if err != nil {
		if isS3NotFound(err) {
			return nil, pathError("open", name, os.ErrNotExist)
		}
		return nil, fmt.Errorf("open s3 object %q for read: %w", name, err)
	}
	return r, nil
}

func (d *s3Driver) WalkDir(ctx context.Context, root string, fn fs.WalkDirFunc) error {
	info, err := d.Stat(ctx, root)
	if err != nil {
		err = fn(".", nil, err)
	} else {
		err = d.walkDir(ctx, root, ".", fs.FileInfoToDirEntry(info), fn)
	}
	if errors.Is(err, fs.SkipDir) || errors.Is(err, fs.SkipAll) {
		return nil
	}
	return err
}

func (d *s3Driver) Move(ctx context.Context, oldName string, newName string) error {
	if newName == oldName {
		return nil
	}
	if err := d.ensureBucket(ctx); err != nil {
		return err
	}

	info, err := d.Stat(ctx, oldName)
	if err != nil {
		return fmt.Errorf("stat rename source %q: %w", oldName, err)
	}

	oldClean := cleanName(oldName)
	newClean := cleanName(newName)
	if oldClean == "" || strings.HasPrefix(newClean, oldClean+"/") {
		return &os.LinkError{
			Op:  "rename",
			Old: oldName,
			New: newName,
			Err: os.ErrPermission,
		}
	}

	if info.IsDir() {
		oldPrefix := d.dirPrefixKey(oldName)
		newPrefix := d.dirPrefixKey(newName)
		if err := d.forEachObject(ctx, simples3.ListInput{
			Bucket: d.bucket,
			Prefix: oldPrefix,
		}, func(obj simples3.Object) error {
			rel := strings.TrimPrefix(obj.Key, oldPrefix)
			if rel == "" {
				return nil
			}
			destKey := path.Join(newPrefix, rel)
			if err := d.copyKey(ctx, obj.Key, destKey); err != nil {
				return fmt.Errorf("copy s3 object %q to %q: %w", obj.Key, destKey, err)
			}
			return nil
		}); err != nil {
			return fmt.Errorf("list rename source %q: %w", oldName, err)
		}
		if err := d.Delete(ctx, oldName); err != nil {
			return fmt.Errorf("remove renamed source dir %q: %w", oldName, err)
		}
		return nil
	}

	if err := d.copyKey(ctx, d.key(oldName), d.key(newName)); err != nil {
		return fmt.Errorf("copy s3 object %q to %q: %w", oldName, newName, err)
	}
	if err := d.deleteKey(ctx, d.key(oldName)); err != nil {
		return fmt.Errorf("remove renamed source file %q: %w", oldName, err)
	}
	return nil
}

func (d *s3Driver) GetContent(ctx context.Context, name string) ([]byte, error) {
	f, err := d.Reader(ctx, name)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = f.Close()
	}()
	return io.ReadAll(f)
}

func (d *s3Driver) PutContent(ctx context.Context, name string, contents []byte) error {
	writer, err := d.Writer(ctx, name, false)
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

func (d *s3Driver) Writer(ctx context.Context, name string, append bool) (FileWriter, error) {
	if err := d.ensureBucket(ctx); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	file, err := os.CreateTemp("", "crkit-s3-*")
	if err != nil {
		return nil, fmt.Errorf("create s3 write temp file %q: %w", name, err)
	}

	writer := &s3FileWriter{
		driver: d,
		ctx:    ctx,
		path:   name,
		file:   file,
	}

	if append {
		if err := writer.copyCurrent(ctx); err != nil {
			_ = writer.closeTemp()
			return nil, err
		}
	}

	return writer, nil
}

func (d *s3Driver) walkDir(ctx context.Context, name string, walkName string, de fs.DirEntry, fn fs.WalkDirFunc) error {
	if err := fn(walkName, de, nil); err != nil || !de.IsDir() {
		if errors.Is(err, fs.SkipDir) && de.IsDir() {
			return nil
		}
		return err
	}

	dirs, err := d.readDir(ctx, name)
	if err != nil {
		err = fn(walkName, de, err)
		if err != nil {
			if errors.Is(err, fs.SkipDir) && de.IsDir() {
				return nil
			}
			return err
		}
	}

	for _, d1 := range dirs {
		name1 := path.Join(name, d1.Name())
		walkName1 := d1.Name()
		if walkName != "." {
			walkName1 = path.Join(walkName, d1.Name())
		}
		if err := d.walkDir(ctx, name1, walkName1, d1, fn); err != nil {
			if errors.Is(err, fs.SkipDir) {
				break
			}
			return err
		}
	}
	return nil
}

func (d *s3Driver) readDir(ctx context.Context, name string) ([]fs.DirEntry, error) {
	objects, prefixes, err := d.listObjects(ctx, simples3.ListInput{
		Bucket:    d.bucket,
		Prefix:    d.dirPrefixKey(name),
		Delimiter: "/",
	})
	if err != nil {
		return nil, fmt.Errorf("list s3 dir %q: %w", name, err)
	}

	entriesByName := map[string]fs.DirEntry{}
	dirPrefix := d.dirPrefixKey(name)

	for _, obj := range objects {
		rel := strings.TrimPrefix(obj.Key, dirPrefix)
		if rel == "" || strings.Contains(rel, "/") {
			continue
		}
		entriesByName[rel] = fs.FileInfoToDirEntry(fsutil.NewFileInfo(
			path.Base(rel),
			obj.Size,
			parseS3Time(obj.LastModified),
		))
	}

	for _, prefix := range prefixes {
		rel := strings.TrimSuffix(strings.TrimPrefix(prefix, dirPrefix), "/")
		if rel == "" || strings.Contains(rel, "/") {
			continue
		}
		entriesByName[rel] = fs.FileInfoToDirEntry(fsutil.NewDirFileInfo(path.Base(rel)))
	}

	entries := make([]fs.DirEntry, 0, len(entriesByName))
	for _, entry := range entriesByName {
		entries = append(entries, entry)
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})
	return entries, nil
}

func (d *s3Driver) statDirectory(ctx context.Context, name string) (filesystem.FileInfo, error) {
	objects, prefixes, err := d.listObjects(ctx, simples3.ListInput{
		Bucket:  d.bucket,
		Prefix:  d.dirPrefixKey(name),
		MaxKeys: 1,
	})
	if err == nil && len(objects)+len(prefixes) > 0 {
		return fsutil.NewDirFileInfo(path.Base(name)), nil
	}
	return nil, pathError("stat", name, os.ErrNotExist)
}

func (d *s3Driver) listObjects(ctx context.Context, input simples3.ListInput) ([]simples3.Object, []string, error) {
	var objects []simples3.Object
	var prefixes []string

	err := d.forEachListPage(ctx, input, func(resp simples3.ListResponse) error {
		objects = append(objects, resp.Objects...)
		prefixes = append(prefixes, resp.CommonPrefixes...)
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	return objects, prefixes, nil
}

func (d *s3Driver) forEachObject(ctx context.Context, input simples3.ListInput, fn func(simples3.Object) error) error {
	return d.forEachListPage(ctx, input, func(resp simples3.ListResponse) error {
		for _, obj := range resp.Objects {
			if err := fn(obj); err != nil {
				return err
			}
		}
		return nil
	})
}

func (d *s3Driver) forEachListPage(ctx context.Context, input simples3.ListInput, fn func(simples3.ListResponse) error) error {
	total := int64(0)

	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		resp, err := d.client.List(input)
		if err != nil {
			return err
		}

		if err := fn(resp); err != nil {
			return err
		}

		total += int64(len(resp.Objects) + len(resp.CommonPrefixes))
		if input.MaxKeys > 0 && total >= input.MaxKeys {
			return nil
		}
		if !resp.IsTruncated || resp.NextContinuationToken == "" {
			return nil
		}

		input.ContinuationToken = resp.NextContinuationToken
	}
}

func (d *s3Driver) deletePrefix(ctx context.Context, prefix string) error {
	input := simples3.ListInput{
		Bucket: d.bucket,
		Prefix: prefix,
	}

	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		resp, err := d.client.List(input)
		if err != nil {
			return err
		}

		lastKey := ""
		for _, obj := range resp.Objects {
			lastKey = obj.Key
			if err := d.deleteKey(ctx, obj.Key); err != nil {
				return fmt.Errorf("remove child s3 object %q: %w", obj.Key, err)
			}
		}

		if !resp.IsTruncated {
			return nil
		}

		if lastKey != "" {
			input.StartAfter = lastKey
			input.ContinuationToken = ""
			continue
		}
		if resp.NextContinuationToken == "" {
			return nil
		}
		input.ContinuationToken = resp.NextContinuationToken
	}
}

func (d *s3Driver) ensureBucket(ctx context.Context) error {
	if d.skipBucketCheck {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	d.ensureOnce.Do(func() {
		_, err := d.client.List(simples3.ListInput{
			Bucket:  d.bucket,
			MaxKeys: 1,
		})
		if err == nil {
			return
		}
		if isS3NotFound(err) {
			_, _ = d.client.CreateBucket(simples3.CreateBucketInput{
				Bucket: d.bucket,
				Region: d.region,
			})
			return
		}
		d.ensureErr = fmt.Errorf("check bucket %q: %w", d.bucket, err)
	})

	return d.ensureErr
}

func (d *s3Driver) putObject(ctx context.Context, name string, body io.ReadSeeker) error {
	if err := d.ensureBucket(ctx); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	_, err := d.client.FilePut(simples3.UploadInput{
		Bucket:      d.bucket,
		ObjectKey:   d.key(name),
		Body:        body,
		ContentType: contentType(ctx),
	})
	return err
}

func (d *s3Driver) putObjectMultipart(ctx context.Context, name string, body io.Reader, size int64) error {
	if err := d.ensureBucket(ctx); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	key := d.key(name)
	initOutput, err := d.client.InitiateMultipartUpload(simples3.InitiateMultipartUploadInput{
		Bucket:      d.bucket,
		ObjectKey:   key,
		ContentType: contentType(ctx),
	})
	if err != nil {
		return err
	}

	abort := func(uploadErr error) error {
		abortErr := d.client.AbortMultipartUpload(simples3.AbortMultipartUploadInput{
			Bucket:    d.bucket,
			ObjectKey: key,
			UploadID:  initOutput.UploadID,
		})
		if abortErr != nil {
			return errors.Join(uploadErr, abortErr)
		}
		return uploadErr
	}

	partSize := int64(simples3.DefaultPartSize)
	if size > partSize*int64(simples3.MaxParts) {
		return abort(fmt.Errorf("file too large: requires more than %d parts", simples3.MaxParts))
	}

	partNumber := 1
	buf := make([]byte, partSize)
	parts := make([]simples3.CompletedPart, 0, (size+partSize-1)/partSize)

	for {
		if err := ctx.Err(); err != nil {
			return abort(err)
		}

		n, readErr := io.ReadFull(body, buf)
		if readErr == io.EOF {
			break
		}
		if readErr != nil && readErr != io.ErrUnexpectedEOF {
			return abort(readErr)
		}
		if n == 0 {
			break
		}

		if partNumber > simples3.MaxParts {
			return abort(fmt.Errorf("file too large: requires more than %d parts", simples3.MaxParts))
		}

		output, err := d.client.UploadPart(simples3.UploadPartInput{
			Bucket:     d.bucket,
			ObjectKey:  key,
			UploadID:   initOutput.UploadID,
			PartNumber: partNumber,
			Body:       bytes.NewReader(buf[:n]),
			Size:       int64(n),
		})
		if err != nil {
			return abort(err)
		}

		parts = append(parts, simples3.CompletedPart{
			PartNumber: output.PartNumber,
			ETag:       output.ETag,
		})
		partNumber++

		if readErr == io.ErrUnexpectedEOF {
			break
		}
	}

	_, err = d.client.CompleteMultipartUpload(simples3.CompleteMultipartUploadInput{
		Bucket:    d.bucket,
		ObjectKey: key,
		UploadID:  initOutput.UploadID,
		Parts:     parts,
	})
	if err != nil {
		return abort(err)
	}
	return nil
}

func (d *s3Driver) copyKey(ctx context.Context, sourceKey string, destKey string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	_, err := d.client.CopyObject(simples3.CopyObjectInput{
		SourceBucket: d.bucket,
		SourceKey:    sourceKey,
		DestBucket:   d.bucket,
		DestKey:      destKey,
	})
	return err
}

func (d *s3Driver) deleteKey(ctx context.Context, key string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return d.client.FileDelete(simples3.DeleteInput{
		Bucket:    d.bucket,
		ObjectKey: key,
	})
}

func (d *s3Driver) key(name string) string {
	name = cleanName(name)
	if d.prefix == "" || d.prefix == "/" {
		return name
	}
	if name == "" {
		return strings.TrimPrefix(path.Clean(d.prefix), "/")
	}
	return strings.TrimPrefix(path.Join(d.prefix, name), "/")
}

func (d *s3Driver) dirPrefixKey(name string) string {
	key := d.key(name)
	if key != "" && !strings.HasSuffix(key, "/") {
		key += "/"
	}
	return key
}

type s3FileWriter struct {
	driver *s3Driver
	ctx    context.Context
	path   string

	written int64
	file    *os.File

	closed          bool
	committed       bool
	cancelled       bool
	commitAttempted bool
}

func (fw *s3FileWriter) Write(p []byte) (int, error) {
	if fw.closed {
		return 0, fmt.Errorf("already closed")
	} else if fw.committed {
		return 0, fmt.Errorf("already committed")
	} else if fw.cancelled {
		return 0, fmt.Errorf("already cancelled")
	}

	n, err := fw.file.Write(p)
	fw.written += int64(n)
	return n, err
}

func (fw *s3FileWriter) Size() int64 {
	return fw.written
}

func (fw *s3FileWriter) Close() error {
	if fw.closed {
		return fmt.Errorf("already closed")
	}

	if !fw.committed && !fw.cancelled && !fw.commitAttempted {
		if err := fw.upload(context.WithoutCancel(fw.ctx)); err != nil {
			return err
		}
	}

	if err := fw.closeTemp(); err != nil {
		return err
	}

	fw.closed = true
	return nil
}

func (fw *s3FileWriter) Cancel(ctx context.Context) error {
	if fw.closed {
		return fmt.Errorf("already closed")
	}

	fw.cancelled = true
	_ = fw.closeTemp()

	return fw.driver.Delete(ctx, fw.path)
}

func (fw *s3FileWriter) Commit(ctx context.Context) error {
	if fw.closed {
		return fmt.Errorf("already closed")
	} else if fw.committed {
		return fmt.Errorf("already committed")
	} else if fw.cancelled {
		return fmt.Errorf("already cancelled")
	}

	fw.commitAttempted = true

	if err := fw.upload(ctx); err != nil {
		return err
	}

	fw.committed = true
	return nil
}

func (fw *s3FileWriter) copyCurrent(ctx context.Context) error {
	r, err := fw.driver.Reader(ctx, fw.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer func() {
		_ = r.Close()
	}()

	n, err := io.Copy(fw.file, r)
	if err != nil {
		return fmt.Errorf("copy current s3 object %q for append: %w", fw.path, err)
	}
	fw.written = n
	return nil
}

func (fw *s3FileWriter) upload(ctx context.Context) error {
	if _, err := fw.file.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("seek s3 write temp file %q: %w", fw.path, err)
	}

	info, err := fw.file.Stat()
	if err != nil {
		return fmt.Errorf("stat s3 write temp file %q: %w", fw.path, err)
	}

	if info.Size() == 0 {
		if err := fw.driver.putObject(ctx, fw.path, fw.file); err != nil {
			return fmt.Errorf("put s3 object %q: %w", fw.path, err)
		}
		return nil
	}

	if err := fw.driver.putObjectMultipart(ctx, fw.path, fw.file, info.Size()); err != nil {
		return fmt.Errorf("put s3 multipart object %q: %w", fw.path, err)
	}
	return nil
}

func (fw *s3FileWriter) closeTemp() error {
	if fw.file == nil {
		return nil
	}

	name := fw.file.Name()
	err := fw.file.Close()
	_ = os.Remove(name)
	fw.file = nil
	return err
}

func endpointPrefix(endpoint strfmt.Endpoint) string {
	bucket := endpoint.Base()
	n := len(bucket + "/")
	if len(endpoint.Path) > n {
		return endpoint.Path[n:]
	}
	return "/"
}

func endpointURL(host string, insecure bool) string {
	scheme := "https"
	if insecure {
		scheme = "http"
	}
	return scheme + "://" + host
}

func cleanName(name string) string {
	if name == "" || name == "." || name == "/" {
		return ""
	}
	return strings.TrimPrefix(path.Clean(name), "/")
}

func isRoot(name string) bool {
	return cleanName(name) == ""
}

func pathError(op string, name string, err error) error {
	return &os.PathError{
		Op:   op,
		Path: name,
		Err:  err,
	}
}

func contentType(ctx context.Context) string {
	metadata := filesystem.MetadataFromContext(ctx)
	if v := metadata.Get("Content-Type"); v != "" {
		return v
	}
	return "application/octet-stream"
}

func isS3NotFound(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return strings.Contains(s, "status code: 404") ||
		strings.Contains(s, "404 Not Found") ||
		strings.Contains(s, "NoSuchKey") ||
		strings.Contains(s, "NoSuchBucket")
}

func parseS3Time(v string) time.Time {
	t, err := http.ParseTime(v)
	if err == nil {
		return t
	}
	t, err = time.Parse(time.RFC3339, v)
	if err == nil {
		return t
	}
	return time.Time{}
}
