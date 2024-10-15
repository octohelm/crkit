/*
Package api GENERATED BY gengo:injectable 
DON'T EDIT THIS FILE
*/
package api

import (
	context "context"

	content "github.com/octohelm/crkit/pkg/content"
)

func (p *NamespaceProvider) InjectContext(ctx context.Context) context.Context {
	return content.NamespaceInjectContext(ctx, p)
}

func (v *NamespaceProvider) Init(ctx context.Context) error {
	if err := v.beforeInit(ctx); err != nil {
		return err
	}
	if err := v.Content.Init(ctx); err != nil {
		return err
	}
	if err := v.afterInit(ctx); err != nil {
		return err
	}

	return nil
}