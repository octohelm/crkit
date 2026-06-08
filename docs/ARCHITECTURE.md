# ARCHITECTURE

crkit 是一个兼容 OCI Distribution Spec V2 的制品注册中心，支持容器镜像和任意类型制品的存储与分发。

## 系统分层

```
┌──────────────────────────────────────────────┐
│                 CLI (internal/cmd/crkit)      │
│          serve / gc / upload-purger           │
└──────────────────┬───────────────────────────┘
                   │
┌──────────────────▼───────────────────────────┐
│           Registry HTTP (pkg/registryhttp)    │
│      OCI Distribution Spec V2 兼容 API        │
└──────────────────┬───────────────────────────┘
                   │
┌──────────────────▼───────────────────────────┐
│            Content (pkg/content)              │
│   Namespace → Repository → {Manifest, Tag, Blob} │
│       实现: fs / remote / proxy               │
└──────────────────┬───────────────────────────┘
                   │
┌──────────────────▼───────────────────────────┐
│             Driver (pkg/driver)               │
│      文件系统抽象: fs（本地）/ s3（对象存储）    │
└──────────────────────────────────────────────┘
```

### 横向能力

```
┌──────────────┐  ┌──────────────┐  ┌───────────────┐
│  pkg/oci      │  │ pkg/artifact  │  │  运维工具       │
│  镜像操作      │  │  制品打包分发   │  │  GC / Purger  │
└──────────────┘  └──────────────┘  └───────────────┘
```

## 各层说明

### Driver — 文件系统抽象

提供统一的文件操作接口：读、写、追加、删除、移动、遍历。支持两种后端：

- **fs** — 本地文件系统
- **s3** — S3 兼容对象存储

### Content — 注册中心领域模型

实现 OCI Distribution Spec 定义的领域概念：

| 概念 | 对应接口 | 职责 |
|---|---|---|
| Namespace | `content.Namespace` | 仓库的逻辑分组，按名称定位 Repository |
| Repository | `content.Repository` | 单个制品存储单元，包含三个子服务 |
| ManifestService | `content.ManifestService` | 清单的增删查 |
| TagService | `content.TagService` | 标签的增删查 |
| BlobStore | `content.BlobStore` | 数据块的上传、下载、删除、分块续传 |

三种 Namespace 实现模式：

- **fs** — 基于 Driver 的本地存储，支持 GC 和上传清理
- **remote** — 直连远程 Registry（OCI Distribution Spec 客户端）
- **proxy** — 本地缓存 + 远程 fallback（写时缓存、读时回源）

### Registry HTTP — OCI Distribution Spec API

对外暴露符合 OCI Distribution Spec V2 的 HTTP API，覆盖：

- **Manifest** — GET / HEAD / PUT / DELETE `/{name}/manifests/{reference}`
- **Blob** — GET / HEAD / DELETE `/{name}/blobs/{digest}`
- **Blob Upload** — POST / GET / PATCH / PUT / DELETE `/{name}/blobs/uploads[/{id}]`
- **Tag** — GET `/{name}/tags/list`
- **Catalog** — GET `/_catalog`

API 层按 courier 三层架构拆分，契约与实现分离：

- `pkg/apis/registry/v2` — 模型、错误、校验
- `pkg/endpoints/registry/v2` — HTTP 契约（路径、参数）
- `pkg/registryhttp/apis/registry` — 实现组装

### OCI 镜像操作（pkg/oci）

提供对 OCI 镜像和 Index 的读写、转换、传输能力：

- 格式解析：OCI Image Manifest / OCI Index、Docker Manifest / Manifest List
- 镜像变异（mutate）
- 远程拉取/推送（remote）
- tar 打包/解包

### 制品打包（pkg/artifact）

基于 OCI Artifact 规范，将任意内容打包为可在 Registry 中分发的制品：

- **executable** — 可执行文件制品
- **kubepkg** — KubePkg 制品

### CLI 入口（internal/cmd/crkit）

- **serve** — 启动 Registry HTTP 服务
- **gc** — 垃圾回收：清理未被任何 Manifest 引用的孤立 Blob
- **upload-purger** — 清理超时的分块上传

## 请求链路

以 `GET /v2/{name}/manifests/latest` 为例：

```
HTTP Request
  → Router (registryhttp/apis.R)
    → GetManifest.Output()
      → Namespace.Repository(ctx, name)
        → Repository.Manifests(ctx)
          → ManifestService.Get(ctx, dgst)
            → Driver.Reader(ctx, path)
```

契约层（apis + endpoints）定义"是什么"，实现层（registryhttp + content）完成"怎么做"。
