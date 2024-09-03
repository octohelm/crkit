package memory

import (
	"testing"

	"github.com/distribution/distribution/v3/registry/storage/cache/cachecheck"
)

func TestInMemoryBlobInfoCache(t *testing.T) {
	cachecheck.CheckBlobDescriptorCache(t, NewInMemoryBlobDescriptorCacheProvider(UnlimitedSize))
}
