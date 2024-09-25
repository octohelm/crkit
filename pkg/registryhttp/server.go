package registryhttp

import (
	"context"
	"net/http"
	"strings"

	infraconfiguration "github.com/innoai-tech/infra/pkg/configuration"
	infrahttp "github.com/innoai-tech/infra/pkg/http"
	"github.com/octohelm/crkit/pkg/registryhttp/apis"
)

type Server struct {
	infrahttp.Server
}

func (s *Server) SetDefaults() {
	if s.Addr == "" {
		s.Addr = ":5000"
	}
}

func (s *Server) Init(ctx context.Context) error {
	s.ApplyRouter(apis.R)

	s.ApplyGlobalHandlers(func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			if req.URL.Path != "/" && strings.HasSuffix(req.URL.Path, "/") {
				req.URL.Path = req.URL.Path[0 : len(req.URL.Path)-1]
				req.RequestURI = req.URL.RequestURI()
			}

			h.ServeHTTP(w, req)
		})
	})

	return infraconfiguration.TypedInit(ctx, &s.Server)
}

func (s *Server) InjectContext(ctx context.Context) context.Context {
	return infraconfiguration.InjectContext(ctx)
}
