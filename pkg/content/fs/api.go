package fs

import (
	"context"
	"os"
	"path/filepath"

	"github.com/octohelm/crkit/pkg/content"
	"github.com/octohelm/unifs/pkg/filesystem"
	"github.com/octohelm/unifs/pkg/filesystem/api"
	"github.com/octohelm/unifs/pkg/strfmt"
)

type NamespaceProvider struct {
	api.FileSystemBackend

	namespace content.Namespace
}

func (s *NamespaceProvider) Init(ctx context.Context) error {
	if s.Backend.IsZero() {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		endpoint, _ := strfmt.ParseEndpoint("file://" + filepath.Join(cwd, ".tmp/container-registry"))
		s.Backend = *endpoint
	}

	if err := s.FileSystemBackend.Init(ctx); err != nil {
		return err
	}

	if err := filesystem.MkdirAll(ctx, s.FileSystem(), "."); err != nil {
		return err
	}

	s.namespace = NewNamespace(s.FileSystem())

	return nil
}

func (s *NamespaceProvider) InjectContext(ctx context.Context) context.Context {
	return content.NamespaceContext.Inject(ctx, s.namespace)
}

func (s *NamespaceProvider) Namespace() content.Namespace {
	return s.namespace
}
