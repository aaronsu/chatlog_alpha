# Tools Agent Rules

## CodeGraph Initialization

- CodeGraph 不限制索引目录；初始化在项目根目录执行，覆盖整个工程。
- `agent_rules/project/project.md` 中描述的正式可编辑源码范围只限制分析和修改行为，不限制 CodeGraph 的索引范围。
- 生成或重建 `agent_rules/project/project.md` 后，必须在项目根目录执行：

```bash
codegraph init
```

- 执行 CodeGraph 初始化后，必须输出实际索引目录；如果命令不可用或执行失败，必须说明失败原因，不能声称索引已初始化。

## Scope

- 本文件专门维护工具使用约束，适用于代码理解、架构追踪、影响分析和准备修改代码的场景。

## CodeGraph

- 代码理解和架构追踪优先使用 CodeGraph：
  - `codegraph_context`
  - `codegraph_trace`
  - `codegraph_callers` / `codegraph_callees`
  - `codegraph_impact`

- 如果当前目标工程没有初始化过 CodeGraph，则必须先在项目根目录执行：

```bash
codegraph init
```

## Serena

- Serena 负责代码修改、重命名、安全删除和修改后的局部检查；初始化流程在 `agent_rules/common/gen_project.md` 中维护。
- 只有 Serena 工具不可用、项目配置损坏、切换 AI 客户端或明确要求重建 Serena 接入时，才执行 Serena 重建；重建顺序固定为先 `serena init`，再按 `agent_rules/env.md` 的 `ai_tools` 执行 `serena setup codex` 或 `serena setup claude-code`。
- 修改已有代码时优先使用 Serena：
  - `find_symbol`
  - `find_referencing_symbols`
  - `replace_symbol_body`
  - `rename_symbol`
  - `get_diagnostics_for_file`
- 删除符号、类、方法或代码文件时，优先使用 Serena 的安全删除能力。
- 修改后只检查修改部分和直接引用修改部分；需要构建、安装、打包、部署、数据库写入或生产操作时，必须先获得用户明确授权。
