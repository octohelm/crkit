package s3

import (
	"bytes"
	"context"
	"errors"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"reflect"
	"strconv"
	"sync"
	"testing"

	"github.com/johannesboyne/gofakes3"
	"github.com/johannesboyne/gofakes3/backend/s3mem"
	"github.com/rhnvrm/simples3"

	"github.com/octohelm/unifs/pkg/strfmt"
)

const testBucket = "testbucket"

func TestS3DriverReadWriteWalkMoveDelete(t *testing.T) {
	server, recorder, backend := newFakeS3Server(t)
	driver := FromS3Endpoint(endpointForServer(t, server, "/"+testBucket+"/base"))
	ctx := context.Background()

	if err := driver.PutContent(ctx, "dir/a.txt", []byte("hello")); err != nil {
		t.Fatalf("put content: %v", err)
	}

	if _, err := backend.HeadObject(testBucket, "base/dir/a.txt"); err != nil {
		t.Fatalf("head object with endpoint prefix: %v", err)
	}

	data, err := driver.GetContent(ctx, "dir/a.txt")
	if err != nil {
		t.Fatalf("get content: %v", err)
	}
	if string(data) != "hello" {
		t.Fatalf("content = %q, want %q", data, "hello")
	}

	info, err := driver.Stat(ctx, "dir")
	if err != nil {
		t.Fatalf("stat dir: %v", err)
	}
	if !info.IsDir() {
		t.Fatalf("stat dir IsDir = false")
	}

	var walked []string
	if err := driver.WalkDir(ctx, ".", func(name string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		walked = append(walked, name)
		return nil
	}); err != nil {
		t.Fatalf("walk dir: %v", err)
	}
	if !reflect.DeepEqual(walked, []string{".", "dir", "dir/a.txt"}) {
		t.Fatalf("walked = %#v", walked)
	}

	walked = nil
	if err := driver.WalkDir(ctx, "dir", func(name string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		walked = append(walked, name)
		return nil
	}); err != nil {
		t.Fatalf("walk sub dir: %v", err)
	}
	if !reflect.DeepEqual(walked, []string{".", "a.txt"}) {
		t.Fatalf("walked sub dir = %#v", walked)
	}

	if err := driver.Move(ctx, "dir/a.txt", "dir/b.txt"); err != nil {
		t.Fatalf("move file: %v", err)
	}
	if _, err := driver.Reader(ctx, "dir/a.txt"); !os.IsNotExist(err) {
		t.Fatalf("reader old path error = %v, want not exist", err)
	}
	moved, err := driver.GetContent(ctx, "dir/b.txt")
	if err != nil {
		t.Fatalf("get moved content: %v", err)
	}
	if string(moved) != "hello" {
		t.Fatalf("moved content = %q, want %q", moved, "hello")
	}

	if err := driver.PutContent(ctx, "dir/sub/c.txt", []byte("nested")); err != nil {
		t.Fatalf("put nested content: %v", err)
	}
	if err := driver.Move(ctx, "dir", "moved"); err != nil {
		t.Fatalf("move dir: %v", err)
	}
	if _, err := driver.Stat(ctx, "dir/b.txt"); !os.IsNotExist(err) {
		t.Fatalf("stat old moved dir path error = %v, want not exist", err)
	}
	nested, err := driver.GetContent(ctx, "moved/sub/c.txt")
	if err != nil {
		t.Fatalf("get moved nested content: %v", err)
	}
	if string(nested) != "nested" {
		t.Fatalf("moved nested content = %q, want %q", nested, "nested")
	}

	if err := driver.Delete(ctx, "moved"); err != nil {
		t.Fatalf("delete dir: %v", err)
	}
	if _, err := driver.Stat(ctx, "moved/b.txt"); !os.IsNotExist(err) {
		t.Fatalf("stat deleted path error = %v, want not exist", err)
	}
	if recorder.completeMultipartCount() != 2 {
		t.Fatalf("complete multipart count = %d, want 2", recorder.completeMultipartCount())
	}
}

func TestS3WriterCloseAndCommitSemantics(t *testing.T) {
	server, recorder, _ := newFakeS3Server(t)
	driver := FromS3Endpoint(endpointForServer(t, server, "/"+testBucket+"/uploads"))
	ctx := context.Background()

	writer, err := driver.Writer(ctx, "resume.txt", false)
	if err != nil {
		t.Fatalf("writer: %v", err)
	}
	if _, err := writer.Write([]byte("hello")); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	if err := writer.Commit(ctx); err == nil || err.Error() != "already closed" {
		t.Fatalf("commit after close error = %v, want already closed", err)
	}
	if recorder.completeMultipartCount() != 0 {
		t.Fatalf("complete multipart count after close = %d, want 0", recorder.completeMultipartCount())
	}

	data, err := driver.GetContent(ctx, "resume.txt")
	if err != nil {
		t.Fatalf("get closed content: %v", err)
	}
	if string(data) != "hello" {
		t.Fatalf("closed content = %q, want %q", data, "hello")
	}

	writer, err = driver.Writer(ctx, "resume.txt", true)
	if err != nil {
		t.Fatalf("append writer: %v", err)
	}
	if writer.Size() != int64(len("hello")) {
		t.Fatalf("append writer size = %d, want %d", writer.Size(), len("hello"))
	}
	if _, err := writer.Write([]byte(" world")); err != nil {
		t.Fatalf("append write: %v", err)
	}
	if err := writer.Commit(ctx); err != nil {
		t.Fatalf("commit: %v", err)
	}
	if _, err := writer.Write([]byte("!")); err == nil || err.Error() != "already committed" {
		t.Fatalf("write after commit error = %v, want already committed", err)
	}
	if err := writer.Commit(ctx); err == nil || err.Error() != "already committed" {
		t.Fatalf("second commit error = %v, want already committed", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close after commit: %v", err)
	}
	if err := writer.Close(); err == nil || err.Error() != "already closed" {
		t.Fatalf("second close error = %v, want already closed", err)
	}

	data, err = driver.GetContent(ctx, "resume.txt")
	if err != nil {
		t.Fatalf("get committed content: %v", err)
	}
	if string(data) != "hello world" {
		t.Fatalf("committed content = %q, want %q", data, "hello world")
	}
	if recorder.completeMultipartCount() != 1 {
		t.Fatalf("complete multipart count after commit = %d, want 1", recorder.completeMultipartCount())
	}
}

func TestS3WriterCommitMultipartPreservesContent(t *testing.T) {
	server, recorder, _ := newFakeS3Server(t)
	driver := FromS3Endpoint(endpointForServer(t, server, "/"+testBucket+"/multipart"))
	ctx := context.Background()

	body := bytes.Repeat([]byte("0123456789abcdef"), simples3.DefaultPartSize/16+1)

	writer, err := driver.Writer(ctx, "large.bin", false)
	if err != nil {
		t.Fatalf("writer: %v", err)
	}
	if _, err := writer.Write(body); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := writer.Commit(ctx); err != nil {
		t.Fatalf("commit: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	got, err := driver.GetContent(ctx, "large.bin")
	if err != nil {
		t.Fatalf("get content: %v", err)
	}
	if !bytes.Equal(got, body) {
		t.Fatalf("multipart content changed: got %d bytes, want %d bytes", len(got), len(body))
	}
	if recorder.completeMultipartCount() != 1 {
		t.Fatalf("complete multipart count = %d, want 1", recorder.completeMultipartCount())
	}
}

func TestS3WriterCancel(t *testing.T) {
	server, _, _ := newFakeS3Server(t)
	driver := FromS3Endpoint(endpointForServer(t, server, "/"+testBucket+"/cancel"))
	ctx := context.Background()

	if err := driver.PutContent(ctx, "delete-me.txt", []byte("old")); err != nil {
		t.Fatalf("put old content: %v", err)
	}

	writer, err := driver.Writer(ctx, "delete-me.txt", true)
	if err != nil {
		t.Fatalf("writer: %v", err)
	}
	if _, err := writer.Write([]byte("new")); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := writer.Cancel(ctx); err != nil {
		t.Fatalf("cancel: %v", err)
	}
	if _, err := writer.Write([]byte("again")); err == nil || err.Error() != "already cancelled" {
		t.Fatalf("write after cancel error = %v, want already cancelled", err)
	}
	if _, err := driver.Reader(ctx, "delete-me.txt"); !os.IsNotExist(err) {
		t.Fatalf("reader cancelled path error = %v, want not exist", err)
	}
}

func TestS3DriverMissingPathErrors(t *testing.T) {
	server, _, _ := newFakeS3Server(t)
	driver := FromS3Endpoint(endpointForServer(t, server, "/"+testBucket))
	ctx := context.Background()

	if _, err := driver.Stat(ctx, "missing.txt"); !os.IsNotExist(err) {
		t.Fatalf("stat missing error = %v, want not exist", err)
	}
	if _, err := driver.Reader(ctx, "missing.txt"); !os.IsNotExist(err) {
		t.Fatalf("reader missing error = %v, want not exist", err)
	}

	var seenErr error
	err := driver.WalkDir(ctx, "missing", func(name string, d fs.DirEntry, err error) error {
		seenErr = err
		return err
	})
	if !errors.Is(err, seenErr) || !os.IsNotExist(seenErr) {
		t.Fatalf("walk missing err = %v callback err = %v, want not exist", err, seenErr)
	}
}

type requestRecorder struct {
	handler http.Handler

	mu                sync.Mutex
	completeMultipart int
	requests          []string
}

func (r *requestRecorder) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.mu.Lock()
	if req.Method == http.MethodPost && req.URL.Query().Get("uploadId") != "" {
		r.completeMultipart++
	}
	r.requests = append(r.requests, req.Method+" "+req.URL.RequestURI())
	r.mu.Unlock()

	r.handler.ServeHTTP(w, req)
}

func (r *requestRecorder) completeMultipartCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.completeMultipart
}

func newFakeS3Server(t *testing.T) (*httptest.Server, *requestRecorder, *s3mem.Backend) {
	t.Helper()

	backend := s3mem.New()
	if err := backend.CreateBucket(testBucket); err != nil {
		t.Fatalf("create bucket: %v", err)
	}

	fake := gofakes3.New(backend, gofakes3.WithTimeSkewLimit(0))
	recorder := &requestRecorder{handler: fake.Server()}
	server := httptest.NewServer(recorder)
	t.Cleanup(server.Close)

	return server, recorder, backend
}

func endpointForServer(t *testing.T, server *httptest.Server, pathname string) strfmt.Endpoint {
	t.Helper()

	u, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parse server url: %v", err)
	}

	port, err := strconv.ParseUint(u.Port(), 10, 16)
	if err != nil {
		t.Fatalf("parse server port: %v", err)
	}

	extra := url.Values{}
	extra.Set("region", "us-east-1")
	extra.Set("insecure", "true")

	return strfmt.Endpoint{
		Scheme:   "s3",
		Hostname: u.Hostname(),
		Port:     uint16(port),
		Path:     pathname,
		Username: "access-key",
		Password: "secret-key",
		Extra:    extra,
	}
}
