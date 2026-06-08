package v1

import (
	"iter"

	specv1 "github.com/opencontainers/image-spec/specs-go/v1"
)

// Descriptor 描述清单内容的元数据
type Descriptor = specv1.Descriptor

// Manifest 清单接口，兼容 OCI 和 Docker 格式
type Manifest interface {
	// Type 返回清单的媒体类型
	Type() string
	// References 遍历清单引用的所有 Descriptor
	References() iter.Seq[Descriptor]
}
