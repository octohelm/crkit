package garbagecollector

import (
	"context"
	"github.com/octohelm/x/sync/singleflight"
	"time"

	"github.com/innoai-tech/infra/pkg/agent"
	"github.com/innoai-tech/infra/pkg/cron"
	"github.com/octohelm/crkit/pkg/content"
	"github.com/octohelm/crkit/pkg/content/fs/driver"
	"github.com/octohelm/exp/xiter"
	"k8s.io/kube-openapi/pkg/validation/strfmt"
)

// +gengo:injectable
type Executor struct {
	ExcludeModifiedIn strfmt.Duration `flags:",omitzero"`
	DryRun            bool            `flags:",omitzero"`

	driver    driver.Driver     `inject:",opt"`
	namespace content.Namespace `inject:",opt"`
}

func (a *Executor) SetDefaults() {
	if a.ExcludeModifiedIn == 0 {
		a.ExcludeModifiedIn = strfmt.Duration(time.Hour)
	}
}

func (gc *Executor) Run(ctx context.Context) error {
	if gc.namespace == nil || gc.driver == nil {
		return nil
	}

	return MarkAndSweepExcludeModifiedIn(
		ctx,
		gc.namespace,
		gc.driver,
		time.Duration(gc.ExcludeModifiedIn),
		gc.DryRun,
	)
}

// +gengo:injectable
type GarbageCollector struct {
	agent.Agent

	Period            cron.Spec       `flags:",omitzero"`
	ExcludeModifiedIn strfmt.Duration `flags:",omitzero"`

	driver    driver.Driver     `inject:",opt"`
	namespace content.Namespace `inject:",opt"`
}

func (a *GarbageCollector) Disabled(ctx context.Context) bool {
	return a.driver == nil || a.namespace == nil || a.Period.Schedule() == nil
}

func (a *GarbageCollector) SetDefaults() {
	if a.Period.IsZero() {
		a.Period = "@midnight"
	}

	if a.ExcludeModifiedIn == 0 {
		a.ExcludeModifiedIn = strfmt.Duration(time.Hour)
	}
}

func (a *GarbageCollector) afterInit(ctx context.Context) error {
	if a.Disabled(ctx) {
		return nil
	}

	sfg := singleflight.Group[string]{}

	a.Host("Mark & Sweep", func(ctx context.Context) error {
		for range xiter.Merge(
			xiter.Of(time.Now()),
			a.Period.Times(ctx),
		) {
			a.Go(ctx, func(ctx context.Context) error {
				err, _ := sfg.Do("mark", func() error {
					defer sfg.Forget("mark")

					return a.MarkAndSweepExcludeModifiedIn(ctx, time.Duration(a.ExcludeModifiedIn))
				})

				return err
			})
		}

		return nil
	})

	return nil
}

func (a *GarbageCollector) MarkAndSweepExcludeModifiedIn(ctx context.Context, hour time.Duration) error {
	return MarkAndSweepExcludeModifiedIn(ctx, a.namespace, a.driver, hour, false)
}
