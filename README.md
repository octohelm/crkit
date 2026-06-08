# Crkit

Container Registry Kit — 兼容 OCI Distribution Spec V2 的制品注册中心。

支持容器镜像和任意 OCI Artifact 格式制品的存储、代理与分发。

## 快速开始

```bash
# 启动本地 Registry（默认监听 :5000）
crkit serve registry

# 指定端口
crkit serve registry --addr=:5070
```

启动后可通过标准 OCI 工具访问：

```bash
docker pull localhost:5000/library/alpine
skopeo copy alpine:latest docker://localhost:5000/library/alpine:latest
```

## 存储模式

### 本地存储（默认）

基于文件系统存储，无需外部依赖：

```bash
# 本地磁盘
crkit serve registry

# S3 对象存储
crkit serve registry --content-backend=s3://bucket/prefix
```

### 代理缓存

启动本地缓存层，拉取时自动回源到远程 Registry：

```bash
crkit serve registry \
  --remote-endpoint=https://docker.io \
  --remote-username=user \
  --remote-password=pass \
  --addr=:5070
```

### 直连代理

不缓存，直接代理所有请求到远程 Registry：

```bash
crkit serve registry \
  --remote-endpoint=https://docker.io \
  --no-cache \
  --addr=:5070
```

## CLI 命令

| 命令 | 作用 |
|---|---|
| `serve registry` | 启动 Registry HTTP 服务 |
| `gc` | 垃圾回收：清理未被引用的孤立 Blob |
| `upload-purger` | 清理超时未完成的分块上传 |

## API

遵循 [OCI Distribution Spec V2](https://github.com/opencontainers/distribution-spec/blob/main/spec.md)：

| 操作 | 路径 |
|---|---|
| 列出仓库 | `GET /v2/_catalog` |
| 列出标签 | `GET /v2/{name}/tags/list` |
| 获取/推送清单 | `GET` / `PUT` `/v2/{name}/manifests/{reference}` |
| 下载/上传 Blob | `GET` `/v2/{name}/blobs/{digest}` |
| 分块上传 Blob | `POST` `/v2/{name}/blobs/uploads/` |

## 开发

```bash
# 构建
just build

# 运行测试
just go test

# 代码生成
just go gen
```

更多细节见 [架构文档](docs/ARCHITECTURE.md) 和 [编码规范](docs/CODING_GUIDELINE.md)。
