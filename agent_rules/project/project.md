# Project Rules

## Project Summary

- 项目名称：`chatlog_alpha`
- Go module：`github.com/sjzar/chatlog`
- 项目类型：Go 本地工具，提供微信 4.x 聊天记录本地查询能力，入口包括 CLI、HTTP 和 MCP。
- 主要平台：README 标明支持 `macOS` 与 `Windows`，其中 Windows 当前需优先在测试环境验证。
- 默认本地服务入口：README 标明根页面为 `http://127.0.0.1:5030/`。

## Runtime And Environment

- Go 版本：`go.mod` 声明 `go 1.24.0`。
- CGO：构建命令使用 `CGO_ENABLED=1`，且依赖 `github.com/mattn/go-sqlite3`。
- Node：朋友圈官方 WASM 解密路径需要 `node`；缺失时程序回退到内置 Go 实现。
- Conda 环境名称：待确认。
- Python 版本：待确认。
- Python/Conda 安装命令：待确认。本仓库未发现 `environment.yml`、`requirements*.txt`、`pyproject.toml`。

## Project Layout

- `main.go`：程序入口，调用 `cmd/chatlog.Execute()`。
- `cmd/chatlog/`：Cobra CLI 命令入口，包含 root、log、HTTP CLI 等命令。
- `internal/chatlog/`：主应用编排。
- `internal/wechatdb/`：微信数据库访问与 repository。
- `internal/model/`：聊天记录、联系人、群聊、媒体、朋友圈等模型。
- `pkg/`：通用包，包括版本、进程、文件监听/复制、时间/字符串/系统工具、媒体格式处理。
- `skills/chatlog-http-cli/`：本项目提供的 chatlog HTTP CLI Skill 文档。
- `agent_rules/`：本仓库规则目录；当前项目专属规则入口为本文件。

## Commands

- 运行：`go run .`
- 构建当前平台：`make build`
- 直接构建：`go build -trimpath -o bin/chatlog main.go`
- 测试：`make test` 或 `go test ./... -cover`
- 依赖整理：`make tidy` 或 `go mod tidy`
- Lint：`make lint` 或 `golangci-lint run ./...`
- 全量 Makefile 流程：`make all` 会依次执行 `clean`、`lint`、`tidy`、`test`、`build`。

## Execution Boundaries

- 默认可操作范围来自 `agent_rules/env.md`：`./`。
- 未经用户明确授权，不执行 Git、构建、安装、打包、部署、数据库写入或生产操作。
- `go mod tidy` 会改动依赖文件，`make clean` 会删除 `bin/`，`make build`/`go build` 属于构建；执行前必须获得用户明确授权。
- 可执行静态检查、语法检查，以及“修改部分”和“直接引用修改部分”的联动检查；检查范围保持最小。
- 不主动运行会访问真实微信数据、密钥、数据库或本地生产资料的命令；如需运行，必须先说明影响并获得用户确认。

## Development Rules

- 只处理用户明确指出的范围；禁止无关重构、格式化、目录清理、额外功能或顺手优化。
- 保持最小必要 diff；禁止无语义变化的删后重加、整段重写或等价改动。
- 代码风格跟随现有 Go 代码：标准 Go 格式、Cobra 命令结构、`internal`/`pkg` 边界。
- 修改共享函数前先检查调用方，优先在共同入口修复根因。
- 新增注释只写业务约束、复杂逻辑或非显然取舍。

## Platform Notes

- macOS 扫描 Key 可能需要 root 权限和系统权限配置；不要在未授权情况下执行相关操作。
- Windows 读取微信进程内存可能需要管理员权限；README 标明 Windows 尚未完成实机测试。
- 本项目处理本地聊天记录、密钥和媒体数据，默认不外传、不上传、不写入外部服务。

## Rule Routing Notes

- 生成或重建本文件时，只能遵循特殊初始化流程：根 `AGENTS.md` -> `agent_rules/env.md` -> `agent_rules/common/gen_project.md` -> 正式工程真实文件。
- 普通任务不要读取旧版生成来源，也不要把模板项目、历史项目或其他仓库规则复制到本项目。
- 当前仓库主要技术栈为 Go；仓库内未发现 Python、Java、Android、Vue 项目标记文件。
