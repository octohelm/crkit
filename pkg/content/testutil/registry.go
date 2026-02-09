package testutil

import (
	"fmt"
	"net/http"
	"os"
	"sync"

	"github.com/octohelm/courier/pkg/courierhttp/handler/httprouter"
	"github.com/octohelm/unifs/pkg/filesystem/local"
	testingv2 "github.com/octohelm/x/testing/v2"

	"github.com/octohelm/crkit/pkg/content"
	contentfs "github.com/octohelm/crkit/pkg/content/fs"
	"github.com/octohelm/crkit/pkg/registryhttp/apis"
)

func NewRegistry(t testingv2.TB) http.Handler {
	tmp := t.TempDir()
	t.Cleanup(func() {
		_ = os.Remove(tmp)
	})
	return &registry{
		namespace: contentfs.NewNamespace(local.NewFS(tmp)),
	}
}

type registry struct {
	namespace content.Namespace
	h         http.Handler
	err       error
	once      sync.Once
}

func (r *registry) ServeHTTP(rw http.ResponseWriter, request *http.Request) {
	r.once.Do(func() {
		h, err := httprouter.New(apis.R, "test-registry")
		if err != nil {
			r.err = err
			return
		}
		r.h = h
	})

	if r.err != nil {
		rw.WriteHeader(http.StatusServiceUnavailable)
		_, _ = fmt.Fprintf(rw, "%s", r.err)
		return
	}

	ctx := request.Context()

	r.h.ServeHTTP(rw, request.WithContext(content.NamespaceInjectContext(ctx, r.namespace)))
}
