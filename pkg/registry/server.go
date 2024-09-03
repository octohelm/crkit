package registry

import (
	"context"
	"net/http"
	"os"
	"runtime"

	"github.com/distribution/distribution/v3"
	"github.com/distribution/distribution/v3/configuration"
	"github.com/distribution/distribution/v3/registry/handlers"
	regsitrymiddleware "github.com/distribution/distribution/v3/registry/middleware/registry"
	"github.com/distribution/distribution/v3/registry/storage/driver"
	"github.com/go-courier/logr"
	infraconfiguration "github.com/innoai-tech/infra/pkg/configuration"
	"github.com/innoai-tech/infra/pkg/http/middleware"
	"github.com/octohelm/courier/pkg/courierhttp/handler"
	"github.com/octohelm/crkit/pkg/registry/remote"
	"golang.org/x/sync/errgroup"

	_ "github.com/distribution/distribution/v3/manifest/manifestlist"
	_ "github.com/distribution/distribution/v3/manifest/ocischema"
	_ "github.com/distribution/distribution/v3/manifest/schema2"
)

type Server struct {
	Storage        Storage
	RemoteRegistry remote.RegistryConfig

	// The address the server endpoint binds to
	Addr string `flag:",omitempty,expose=http"`

	Cleaner Cleaner

	Publisher

	s *http.Server
}

func (s *Server) SetDefaults() {
	if s.Addr == "" {
		s.Addr = ":5000"
	}

	s.Cleaner.SetDefaults()
	s.Publisher.SetDefaults()
}

func (s *Server) Init(ctx context.Context) error {
	c := &Configuration{}

	c.StorageRoot = s.Storage.Root

	if err := os.MkdirAll(c.StorageRoot, os.ModePerm); err != nil {
		return err
	}

	c.RegistryAddr = s.Addr

	if s.RemoteRegistry.Endpoint != "" {
		c.Proxy = &Proxy{
			RemoteURL: s.RemoteRegistry.Endpoint,
			Username:  s.RemoteRegistry.Username,
			Password:  s.RemoteRegistry.Password,
		}
	}

	reg, localReg, err := c.New(ctx)
	if err != nil {
		return err
	}

	_ = regsitrymiddleware.Register("custom", func(ctx context.Context, registry distribution.Namespace, driver driver.StorageDriver, options map[string]interface{}) (distribution.Namespace, error) {
		return reg, nil
	})

	app := handlers.NewApp(ctx, &configuration.Configuration{
		Storage: configuration.Storage{
			"filesystem": map[string]any{
				"rootdirectory": c.StorageRoot,
			},
		},
		Middleware: map[string][]configuration.Middleware{
			"registry": {
				{Name: "custom"},
			},
		},
	})

	svc := &http.Server{}

	svc.Addr = c.RegistryAddr
	svc.Handler = handler.ApplyMiddlewares(
		middleware.HealthCheckHandler(),
		middleware.ContextInjectorMiddleware(infraconfiguration.ContextInjectorFromContext(ctx)),
		middleware.LogAndMetricHandler(),
		enableMirrors,
	)(app)

	s.s = svc

	s.Cleaner.ApplyRegistry(localReg, c.MustStorage(), BaseHost(c.RegistryBaseHost))

	if err := s.Cleaner.Init(ctx); err != nil {
		return err
	}

	return s.Publisher.InitWith(s.Addr)
}

func (s *Server) Run(ctx context.Context) error {
	g, c := errgroup.WithContext(ctx)

	g.Go(func() error {
		return s.Publisher.Run(c)
	})

	return g.Wait()
}

func (s *Server) Serve(ctx context.Context) error {
	g, c := errgroup.WithContext(ctx)
	if s.s != nil {
		g.Go(func() error {
			l := logr.FromContext(c)

			l.Info("container registry serve on %s (%s/%s)", s.s.Addr, runtime.GOOS, runtime.GOARCH)

			if s.RemoteRegistry.Endpoint != "" {
				l.Info("proxy fallback %s enabled", s.RemoteRegistry.Endpoint)
			}

			return s.s.ListenAndServe()
		})
	}

	g.Go(func() error {
		return s.Cleaner.Serve(c)
	})

	return g.Wait()
}

func (s *Server) Shutdown(ctx context.Context) error {
	g, c := errgroup.WithContext(ctx)

	if s.s != nil {
		g.Go(func() error {
			return s.s.Shutdown(c)
		})
	}

	g.Go(func() error {
		return s.Cleaner.Shutdown(c)
	})

	return g.Wait()
}
