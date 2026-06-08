# CONTEXT

## 术语表

### Registry（注册中心）

crkit 提供的制品注册中心服务，兼容 OCI Distribution Spec V2。不仅支持容器镜像的存储与分发，还支持以 OCI Artifact 格式打包和分发任意类型的制品（如可执行文件、KubePkg 等）。

> 英文保留 "Registry"，中文使用"注册中心"。

### Namespace（命名空间）

仓库的逻辑分组容器。通过 Namespace 定位到具体的 Repository。crkit 支持三种实现模式：

- **直连（remote）** — 直接操作远程 Registry
- **本地（fs）** — 基于文件系统存储（本地磁盘或 S3）
- **代理（proxy）** — 本地缓存 + 远程 fallback

### Repository（仓库）

单个制品（镜像）的逻辑存储单元，通过 `reference.Named` 标识（如 `library/alpine`）。每个 Repository 包含三个子服务：ManifestService（清单管理）、TagService（标签管理）、BlobStore（数据块存储）。

### Manifest（清单）

描述制品内容和结构的元数据。兼容 OCI Image Manifest / OCI Index 和 Docker Manifest / Manifest List 四种格式。

### Blob（数据块）

制品中的原始数据单元（如镜像的 config、layer）。支持分块上传（chunked upload），由 BlobStore 管理生命周期。

### Tag（标签）

指向特定 Manifest 的人类可读引用。可变——同一个 Tag 可以重新指向不同的 Manifest。

### Image（镜像）

OCI Image，由 Config（配置对象）和 Layers（层）组成，两者均以 Blob 形式存储在 Registry 中。

### Index（索引）

OCI Image Index，指向一组不同平台的 Image 或其他 Index，用于多架构（multi-arch）镜像分发。

### Artifact（制品）

可通过 `pkg/artifact/` 打包为 OCI Index 并在 Registry 中分发的任意内容类型。当前支持的制品类型包括：

- **executable** — 可执行文件制品，`artifactType = application/vnd.executable+index`
- **kubepkg** — KubePkg 制品，`artifactType = application/vnd.kubepkg+index`

> 英文保留 "Artifact"，中文统一使用"制品"。

### Driver（驱动）

底层文件存储的抽象接口，提供 WalkDir / Stat / Reader / Writer / Delete / Move 等文件操作。当前实现：

- **fs** — 本地文件系统
- **s3** — S3 兼容的对象存储

### Garbage Collector（垃圾回收）

扫描 Repository 中的所有 Blob，删除未被任何 Manifest 引用的孤立 Blob，回收存储空间。

### Upload Purger（上传清理）

清理超时未完成的 Blob 分块上传（chunked upload），释放临时资源。

