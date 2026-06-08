# Go 工具链入口
mod go 'tool/go/justfile'

# crkit 项目特有命令
mod crkit 'internal/cmd/crkit/justfile'

[group('meta')]
default:
    @just --list --list-submodules
