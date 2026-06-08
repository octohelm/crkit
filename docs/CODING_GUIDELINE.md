# CODING GUIDELINE

## 代码组织

### 目录结构

```
pkg/
├── apis/{domain}/v{major}      # 契约模型：类型定义、校验、错误码
├── endpoints/{domain}/v{major}  # HTTP 契约：路径、参数位置、返回声明
├── registryhttp/               # Registry HTTP 服务
│   └── apis/registry/          #   端点实现（Output 方法体 + 注入依赖）
├── content/                    # 领域接口 + 存储实现
│   ├── fs/                     #   本地文件系统实现
│   ├── remote/                 #   直连远程 Registry
│   └── proxy/                  #   代理缓存
├── driver/                     # 文件系统驱动
│   ├── fs/                     #   本地磁盘
│   └── s3/                     #   S3 对象存储
├── oci/                        # OCI 镜像操作工具
└── artifact/                   # 制品打包与分发

internal/
├── cmd/crkit/                  # CLI 入口
└── pkg/                        # 内部工具包
```

### 文件命名

- 按领域概念命名，如 `manifest.go`、`blob.go`，不使用 `types.go`、`common.go` 等泛名
- 端点文件按资源分组，不按 HTTP method 拆分（如 `manifest.go` 包含该资源的所有动词）
- 生成文件统一前缀 `zz_generated.`，后缀 `.go`

## 代码风格

- Go 标准格式化（gofmt / goimports）
- 导出类型和函数使用大写，注释使用英文或中文（保持一致）
- 包内类型引用尽量短名，跨包引用使用有意义的别名

### 导入别名

当同包名冲突时使用别名区分：

```go
import (
    apisv2 "github.com/octohelm/crkit/pkg/apis/registry/v2"
    endpointsv2 "github.com/octohelm/crkit/pkg/endpoints/registry/v2"
)
```

## 向后兼容

当类型从实现包迁移到契约层，在原位置保留 `type alias` 避免下游 break：

```go
// pkg/content/named.go
import apisv2 "github.com/octohelm/crkit/pkg/apis/registry/v2"

type Name = apisv2.Name
```

下游无需改动即可继续使用原有引用路径。

## 代码生成

### 触发时机

修改以下内容后必须运行 `just go gen`：

- 新增/修改 `+gengo:injectable` 标记的 struct
- 新增/修改 `+gengo:operator` 标记的 package
- 新增/修改 `+gengo:injectable:provider` 标记的接口

### 生成标签说明

| 标签 | 作用 |
|---|---|
| `+gengo:injectable` | struct 打此标签后，生成 `Init(ctx)` 方法，自动从 context 解析注入依赖 |
| `+gengo:injectable:provider` | 接口打此标签后，生成 `FromContext` / `InjectContext` 辅助函数 |
| `+gengo:operator` | package 打此标签后，扫描包内类型生成路由注册代码 |
| `+gengo:operator:register=R` | 同时指定注册到的 Router 变量名 |

### 产物

生成文件（`zz_generated.*.go`）为只读产物，**不手写、不手改**。

## 测试

### 运行

```sh
just go test          # 全部测试
just go test ./pkg/content/fs  # 指定包
```

### 编写规范

- 测试文件命名 `*_test.go`，与被测文件同目录
- 使用标准库 `testing` 包，断言优先使用 `github.com/octohelm/x/testing/v2`
- 集成测试使用 `pkg/content/testutil` 中定义的共享测试套件

## 文档

### 必须维护的文档

| 文件 | 内容 | 触发条件 |
|---|---|---|
| `CONTEXT.md` | 领域术语表 | 新增或变更领域概念时立即更新 |
| `docs/ARCHITECTURE.md` | 系统架构 | 新增模块或改变分层关系时更新 |
| `docs/CODING_GUIDELINE.md` | 编码规范（本文档） | 规范变更时更新 |
| `docs/adr/` | 架构决策记录 | 满足 ADR 条件时新增 |

### 文档语言

- 文档正文使用中文，专有名词保留英文
- 术语严格使用 `CONTEXT.md` 中定义的词汇
