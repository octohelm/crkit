package fs_test

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/distribution/reference"
	"github.com/johannesboyne/gofakes3"
	"github.com/johannesboyne/gofakes3/backend/s3mem"
	"github.com/opencontainers/go-digest"
	ocispecv1 "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/innoai-tech/infra/pkg/configuration"
	"github.com/innoai-tech/infra/pkg/configuration/testingutil"
	"github.com/innoai-tech/infra/pkg/otel"
	"github.com/octohelm/courier/pkg/courierhttp/handler/httprouter"
	"github.com/octohelm/unifs/pkg/strfmt"
	. "github.com/octohelm/x/testing/v2"

	"github.com/octohelm/crkit/pkg/content"
	contentapi "github.com/octohelm/crkit/pkg/content/api"
	"github.com/octohelm/crkit/pkg/content/collect"
	contentremote "github.com/octohelm/crkit/pkg/content/remote"
	"github.com/octohelm/crkit/pkg/oci"
	"github.com/octohelm/crkit/pkg/oci/random"
	"github.com/octohelm/crkit/pkg/oci/remote"
	"github.com/octohelm/crkit/pkg/registryhttp/apis"
)

const namespaceS3Bucket = "namespace-test"

func TestNamespaceS3(t *testing.T) {
	s3Server := newNamespaceFakeS3Server(t)

	image := MustValue(t, func() (oci.Image, error) {
		return random.Image(6*1024*1024, 1)
	})

	ctx, _ := testingutil.BuildContext(t, func(d *struct {
		otel.Otel
		contentapi.NamespaceProvider
	}) {
		d.Content.Backend = endpointForNamespaceS3Server(t, s3Server, "/"+namespaceS3Bucket+"/content")
	})

	s := MustValue(t, func() (*httptest.Server, error) {
		injector := configuration.ContextInjectorFromContext(ctx)

		h, err := httprouter.New(apis.R, "registry")
		if err != nil {
			return nil, err
		}

		return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			if strings.HasSuffix(req.URL.Path, "/") {
				req.URL.Path = req.URL.Path[0 : len(req.URL.Path)-1]
			}

			h.ServeHTTP(w, req.WithContext(injector.InjectContext(req.Context())))
		})), nil
	})
	t.Cleanup(s.Close)

	reg := MustValue(t, func() (content.Namespace, error) {
		return contentremote.New(ctx, contentremote.Registry{
			Endpoint: s.URL,
		})
	})

	remoteRepo := MustValue(t, func() (content.Repository, error) {
		named, err := reference.WithName("test/manifest")
		if err != nil {
			return nil, err
		}
		return reg.Repository(ctx, named)
	})

	ns, _ := content.NamespaceFromContext(ctx)

	Then(t, "推送镜像到 S3-backed 容器注册表",
		ExpectDo(
			func() error {
				return remote.Push(ctx, image, remoteRepo, "latest")
			},
		),
	)

	Then(t, "获取目录列表",
		ExpectMustValue(
			func() ([]string, error) {
				return collect.Catalogs(ctx, ns)
			},
			Equal([]string{remoteRepo.Named().Name()}),
		),
	)

	Then(t, "验证推送的 manifest 数量",
		ExpectMustValue(
			func() (int, error) {
				repo, err := ns.Repository(ctx, remoteRepo.Named())
				if err != nil {
					return 0, err
				}

				manifests, err := repo.Manifests(ctx)
				if err != nil {
					return 0, err
				}

				revisions, err := collect.Manifests(ctx, manifests)
				return len(revisions), err
			},
			Equal(1),
		),
	)

	layer := MustValue(t, func() (ocispecv1.Descriptor, error) {
		for blob, err := range image.Layers(ctx) {
			if err != nil {
				return ocispecv1.Descriptor{}, err
			}
			return blob.Descriptor(ctx)
		}
		return ocispecv1.Descriptor{}, fmt.Errorf("image has no layers")
	})

	Then(t, "验证 S3 中读取的 layer digest 与 descriptor 一致",
		ExpectMustValue(
			func() (digest.Digest, error) {
				blobs, err := remoteRepo.Blobs(ctx)
				if err != nil {
					return "", err
				}

				r, err := blobs.Open(ctx, layer.Digest)
				if err != nil {
					return "", err
				}
				defer r.Close()

				digester := layer.Digest.Algorithm().Digester()
				n, err := io.Copy(digester.Hash(), r)
				if err != nil {
					return "", err
				}
				if n != layer.Size {
					return "", fmt.Errorf("unexpected layer size %d, expected %d", n, layer.Size)
				}
				return digester.Digest(), nil
			},
			Equal(layer.Digest),
		),
	)

	imagePushed := MustValue(t, func() (oci.Manifest, error) {
		return remote.Manifest(ctx, remoteRepo, "latest")
	})

	Then(t, "拉取后重新推送为 v1 标签",
		ExpectDo(
			func() error {
				return remote.Push(ctx, imagePushed, remoteRepo, "v1")
			},
		),
	)

	tags := MustValue(t, func() (content.TagService, error) {
		return remoteRepo.Tags(ctx)
	})

	Then(t, "验证存在两个标签",
		ExpectMustValue(
			func() ([]string, error) {
				return tags.All(ctx)
			},
			Equal([]string{"latest", "v1"}),
		),
	)
}

func newNamespaceFakeS3Server(t *testing.T) *httptest.Server {
	t.Helper()

	backend := s3mem.New()
	if err := backend.CreateBucket(namespaceS3Bucket); err != nil {
		t.Fatalf("create bucket: %v", err)
	}

	fake := gofakes3.New(backend, gofakes3.WithTimeSkewLimit(0))
	server := httptest.NewServer(fake.Server())
	t.Cleanup(server.Close)

	return server
}

func endpointForNamespaceS3Server(t *testing.T, server *httptest.Server, pathname string) strfmt.Endpoint {
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
	extra.Set("skipBucketCheck", "true")

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
