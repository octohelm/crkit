/*
Package garbagecollector GENERATED BY gengo:injectable
DON'T EDIT THIS FILE
*/
package garbagecollector

import (
	context "context"

	content "github.com/octohelm/crkit/pkg/content"
	fsdriver "github.com/octohelm/crkit/pkg/content/fs/driver"
)

func (v *Executor) Init(ctx context.Context) error {
	if value, ok := fsdriver.DriverFromContext(ctx); ok {
		v.driver = value
	}
	if value, ok := content.NamespaceFromContext(ctx); ok {
		v.namespace = value
	}

	return nil
}

func (v *GarbageCollector) Init(ctx context.Context) error {
	if value, ok := fsdriver.DriverFromContext(ctx); ok {
		v.driver = value
	}
	if value, ok := content.NamespaceFromContext(ctx); ok {
		v.namespace = value
	}
	if err := v.Agent.Init(ctx); err != nil {
		return err
	}

	if err := v.afterInit(ctx); err != nil {
		return err
	}

	return nil
}
