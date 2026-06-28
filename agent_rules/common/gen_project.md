# Project Rules Generation

## Purpose

- 维护 `agent_rules/project/project.md` 的通用生成规则；按项目根目录最小充分探测真实工程文件，避免把历史项目、临时文件或旧 `project.md` 当作事实来源。

## Read Order

1. 先读仓库根目录 `AGENTS.md`。
2. 读取 `agent_rules/env.md`，取得 `ai_tools` 的值。
3. 生成或重建 `agent_rules/project/project.md` 时，直接读取本文件。
4. 删除旧 `agent_rules/project/project.md`。
5. 按项目根目录最小充分探测真实项目文件。

生成或重建 `project.md` 时，禁止读取旧 `agent_rules/project/project.md`，也不进入普通执行链路中的 `agent_rules/common/common.md` 或技术栈规则。新 `project.md` 生成完成后，必须在项目根目录依次执行 Serena 初始化、Serena setup、`codegraph init`，并输出实际索引目录。

## Writing Project Rules

当新项目缺少 `agent_rules/project/project.md`，或需要重新生成项目专属规则时，先读取仓库根目录 `AGENTS.md`、`agent_rules/env.md` 和本文件，再按最小充分原则查看项目根目录下的真实工程文件。生成内容必须基于仓库内真实文件和用户已确认的信息，不要读取旧 `project.md`，不要把模板项目、历史项目或其他仓库的规则复制进来。

### Generation Sequence

1. 读取 `AGENTS.md`，识别当前任务为生成或重建 `project.md`。
2. 读取 `agent_rules/env.md`。
3. 读取 `agent_rules/common/gen_project.md`。
4. 删除旧 `agent_rules/project/project.md`；如果删除失败，必须停止生成并说明原因。
5. 只读取项目根目录下真实工程的根清单、构建配置、包管理文件、主入口文件、环境文件和现有测试目录；禁止全量读取源码。
6. 生成新的 `agent_rules/project/project.md`。
7. 在项目根目录执行 `serena init`，完成 Serena 全局初始化；如果命令不可用或执行失败，必须停止后续 Serena setup 并说明原因。
8. 初始化或更新 `.serena/project.yml`；如果 `.serena/project.yml` 不存在，执行 `serena project create .`，并让 Serena 基于真实工程推断语言；如果已存在，只按真实工程变化做最小必要更新。
9. 读取 `agent_rules/env.md` 的 `ai_tools`，按工具值执行 Serena setup：`codex` 执行 `serena setup codex`，`claude` 执行 `serena setup claude-code`；未知值必须停止 setup 并说明原因。
10. 在项目根目录执行 `codegraph init` 索引整个工程。
11. 输出本次实际索引目录。

### Project Meaning

- `Project Root` 指仓库当前根目录，也就是根目录 `AGENTS.md` 所在目录。
- `Project` 指当前用户真正要让 AI 理解、修改、检查和交付的正式工程范围；CodeGraph 可以索引整个工程，但修改范围仍受用户请求和 `project.md` 约束。
- `agent_rules/project/` 是项目专属规则与测试资料目录，不是业务源码目录。
- `agent_rules/project/project.md` 必须明确描述正式工程范围、只读参考范围、编译/检查方式、启动方式和验证方式。
- 历史归档、参考文档、截图、构建产物、临时脚本、测试记录和规则文件默认不属于可编辑工程源码；除非用户明确指定，否则只能作为只读参考或辅助资料。

### Project Discovery

- 生成或更新 `agent_rules/project/project.md` 前，必须基于项目根目录下的真实工程文件识别正式工程、只读参考目录、历史目录、构建产物和规则资料目录。
- 正式工程范围写入 `project.md` 后作为可编辑范围依据；如果存在多个合理解释，必须标记“待确认”或向用户确认，不得用单独配置文件维护目录限制。
- AI 必须基于真实工程文件自动识别每个工程的技术栈、启动方式、端口、接口前缀、静态检查方式、自动化测试方式和测试记录位置；无法确认的内容必须写成“待确认”，不得删除对应规则项。

### Tool Configuration Boundary

- `agent_rules/env.md` 只保存当前 AI 工具环境与范围配置，例如 `ai_tools: codex`、`operation_scope`；具体执行约束写在规则文件中。
- Serena 初始化和 setup 只服务代码理解、修改和安全删除，不决定可编辑目录；可编辑目录仍以 `agent_rules/env.md` 的 `operation_scope` 和 `agent_rules/project/project.md` 为准。
- 维护 `.serena/project.yml` 时，只能根据真实工程识别出的技术栈做最小必要更新，未知语言 key 必须标记“待确认”或停止说明原因，不得猜测。

### Required Output

- 生成位置固定为 `agent_rules/project/project.md`；所有路径使用相对项目根目录路径。
- 必须写明正式可编辑工程、只读参考范围、历史 / 临时 / 产物目录、关键模块和业务边界。
- 每个正式工程必须写明技术栈、工作目录、启动命令、端口、环境文件、静态 / 语法检查命令、验证方法和是否需要用户授权。
- 无法从真实文件确认的目录、端口、命令、服务用途或测试入口必须写成“待确认”，不得猜测。
- 必须生成 `## Test Methods`，维护测试数据原则、测试记录索引、运行态 / 自动化测试入口、接口基线、鉴权方式和固定测试基线。
- 测试记录写入 `agent_rules/project/tests/*_tests_flows.md`；脚本放 `tests/scripts/`；产物放 `tests/artifacts/`；新增脚本名必须能看出测试对象或场景。
- Python 正式工程必须记录专属 conda 环境名、Python 版本和 `conda run -n <env_name> ...` 形式的启动 / 检查命令；未知项写“待确认”。
- 禁止写入真实账号、真实密码、生产 token、私钥、数据库连接串等敏感信息。

### Suggested Sections

- `Scope`、`Project Structure`、`Hard Rules`、`Local Baseline`、`Local Commands`、`Test Accounts`、`Test Methods`。

### Discovery Rules

- 只读取最小充分文件：根清单、构建配置、包管理文件、主入口文件、环境文件和现有测试目录。
- 优先识别正式源码目录，不把历史备份、压缩包、截图、构建产物、临时脚本默认纳入可修改范围。
- 技术栈判断结合 `Stack Markers` 和项目实际入口；仓库中存在某技术栈，不代表当前任务允许修改该技术栈。
- 如果正式项目或可修改范围不清，先在 `project.md` 标记待确认，或向用户确认后再落规则。

## Detection Priority

1. 用户明确指定的项目类型、模块、目录或文件。
2. 即将修改文件所在目录的最近标记文件。
3. 仓库根目录标记文件。

- 多个技术栈同时命中时：优先以即将修改文件所在目录为准；Android 与 Java 同时命中时按 Android 处理；跨技术栈任务读取全部命中规则；同一问题同时满足多份规则时从严执行。

## Stack Markers

### Android

- 命中任一特征即视为 Android：`settings.gradle` / `settings.gradle.kts`；`build.gradle` / `build.gradle.kts`；`src/main/AndroidManifest.xml`；`src/main/res/`；位于 Android module 的 `*.kt` / `*.java`。

### Vue

- 命中任一组合即视为 Vue：`package.json` + `vite.config.*`；`package.json` + `vue.config.*`；`package.json` + `nuxt.config.*`；`src/**/*.vue`。

### Python

- 命中任一特征即视为 Python：`pyproject.toml`；`requirements.txt` / `requirements-dev.txt`；`setup.py` / `setup.cfg`；`Pipfile`；成体系的 `*.py` 模块或包目录。
- 命中 Python 后，所有 Python 项目必须使用项目专属 conda 环境，禁止直接使用或切换系统 Python、base 环境或其他项目环境。
- 创建或使用 Python 环境前，必须明确记录 conda 环境名称与 Python 版本；命令优先使用 `conda run -n <env_name> ...` 精确指定环境，不通过修改全局 PATH 或系统默认解释器来切换。
- 项目专属 conda 环境名称、Python 版本、安装命令和启动命令必须写入 `agent_rules/project/project.md`；如果当前项目尚未确认这些信息，必须在 `project.md` 标记为“待确认”，不要猜测。

### Java

- 命中任一特征即视为 Java：`pom.xml`；`mvnw`；`src/main/java`；`src/test/java`；非 Android 模块下成体系的 `*.java` 文件。

## Notes

- 本文件维护 `project.md` 生成规则；主入口仍是仓库根目录 `AGENTS.md`。
- 当前工程的专属规则与测试资料统一放在 `agent_rules/project/`，其中项目规则入口为 `agent_rules/project/project.md`，测试资料目录为 `agent_rules/project/tests/`。
- 仓库中存在某技术栈，不等于当前任务必须读取该技术栈规则；规则应跟随当前目标文件、模块和直接调用链，而不是被仓库中“存在什么技术栈”所扩散。
- 是否使用 `planning-with-files` 仍由仓库根入口规则决定。
