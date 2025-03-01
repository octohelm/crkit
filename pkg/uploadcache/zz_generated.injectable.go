/*
Package uploadcache GENERATED BY gengo:injectable
DON'T EDIT THIS FILE
*/
package uploadcache

import (
	context "context"
)

func (p *MemUploadCache) InjectContext(ctx context.Context) context.Context {
	return UploadCacheInjectContext(ctx, p)
}

func (v *MemUploadCache) Init(ctx context.Context) error {
	if err := v.beforeInit(ctx); err != nil {
		return err
	}
	if err := v.Agent.Init(ctx); err != nil {
		return err
	}

	return nil
}

type contextUploadCache struct{}

func UploadCacheFromContext(ctx context.Context) (UploadCache, bool) {
	if v, ok := ctx.Value(contextUploadCache{}).(UploadCache); ok {
		return v, true
	}
	return nil, false
}

func UploadCacheInjectContext(ctx context.Context, tpe UploadCache) context.Context {
	return context.WithValue(ctx, contextUploadCache{}, tpe)
}
