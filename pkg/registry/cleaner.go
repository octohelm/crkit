package registry

import (
	"context"
	"github.com/distribution/distribution/v3"
	"github.com/distribution/distribution/v3/registry/storage"
	"github.com/distribution/reference"
	"github.com/go-courier/logr"
	"github.com/innoai-tech/infra/pkg/cron"
	"github.com/octohelm/kubekit/pkg/kubeclient"
	corev1 "k8s.io/api/core/v1"
	"strings"

	"github.com/distribution/distribution/v3/registry/storage/driver"
)

type Cleaner struct {
	cron.Job

	UntagWhenNotExistsInKube bool `flag:",omitempty"`

	baseHost BaseHost
	driver   driver.StorageDriver
	registry distribution.Namespace
}

func (c *Cleaner) Init(ctx context.Context) error {
	c.ApplyAction("container registry gc", func(ctx context.Context) {
		_ = c.Run(ctx)
	})

	return c.Job.Init(ctx)
}

func (c *Cleaner) ApplyRegistry(cr distribution.Namespace, storageDriver driver.StorageDriver, baseHost BaseHost) {
	c.registry = cr
	c.driver = storageDriver
	c.baseHost = baseHost
}

func (c *Cleaner) Run(ctx context.Context) error {
	l := logr.FromContext(ctx)

	l.Info("running cleaner")

	images, err := c.collectionImages(ctx)
	if err != nil {
		l.Error(err)
		return nil
	}

	// can failed
	if err := c.untagIfNeed(ctx, images); err != nil {
		l.Error(err)
	}

	if err := c.remoteUntaggedBlobs(ctx); err != nil {
		l.Error(err)
		return nil
	}

	return nil
}

func (c *Cleaner) remoteUntaggedBlobs(ctx context.Context) error {
	return storage.MarkAndSweep(ctx, c.driver, c.registry, storage.GCOpts{
		DryRun:         false,
		RemoveUntagged: true,
	})
}

func (c *Cleaner) collectionImages(ctx context.Context) (map[reference.NamedTagged]distribution.Repository, error) {
	images := map[reference.NamedTagged]distribution.Repository{}

	err := c.registry.(distribution.RepositoryEnumerator).Enumerate(ctx, func(repoName string) error {
		named, err := reference.WithName(repoName)
		if err != nil {
			return err
		}
		resp, err := c.registry.Repository(ctx, named)
		if err != nil {
			return err
		}
		tags, err := resp.Tags(ctx).All(ctx)
		if err != nil {
			return err
		}
		for _, tag := range tags {
			namedTag, _ := reference.WithTag(named, tag)
			images[namedTag] = resp
		}
		return nil
	})

	return images, err
}

func (c *Cleaner) untagIfNeed(ctx context.Context, images map[reference.NamedTagged]distribution.Repository) error {
	if !c.UntagWhenNotExistsInKube {
		return nil
	}

	l := logr.FromContext(ctx)

	if kc, ok := kubeclient.Context.MayFrom(ctx); ok {
		nodeList := &corev1.NodeList{}

		if err := kc.List(ctx, nodeList); err != nil {
			return err
		}

		for _, n := range nodeList.Items {
			for _, i := range n.Status.Images {
				for _, name := range i.Names {
					if strings.Contains(name, "@") {
						continue
					}

					named, err := reference.ParseNamed(name)
					if err != nil {
						continue
					}

					fixedNamed, ok := c.baseHost.CompletedNamed(named).(reference.NamedTagged)
					if !ok {
						continue
					}

					if name := fixedNamed.String(); strings.HasPrefix(name, "docker.io/") {
						sortName, err := reference.Parse(name[len("docker.io/"):])
						if err == nil {
							l.WithValues("name", sortName).Debug("remain")
							delete(images, sortName.(reference.NamedTagged))
						}
					}

					l.WithValues("name", name).Debug("remain")
					delete(images, fixedNamed.(reference.NamedTagged))
				}
			}
		}
	}

	for tagged, repo := range images {
		l.WithValues("name", tagged).Info("remove")

		if err := repo.Tags(ctx).Untag(ctx, tagged.Tag()); err != nil {
			return err
		}
	}

	return nil
}
