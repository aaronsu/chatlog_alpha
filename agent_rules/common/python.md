# Python Agent Rules

## Scope

- 任务命中 Python 标记时，除仓库根目录 `AGENTS.md` 和 `agent_rules/common/common.md` 外，还必须读取本文件。

## Edit Rules

- 仅进行编码、静态检查、语法检查和引用联动检查。
- 未经允许，不执行 `pip`、`poetry`、`uv`、`conda` 等安装或环境变更命令，也不执行与任务无关的测试、打包、发布命令。
- 保持现有模块边界、入口方式、配置方式，不做无关重构。
- 修改模块后，检查直接导入方、直接调用方、配置入口、命令入口；修改数据结构、函数签名、异常逻辑后，检查直接使用这些接口的代码。

## Conda Environment

- Python conda 环境创建与使用规则跟随 `agent_rules/common/gen_project.md` 的 Python 节点。
- 当前项目的 conda 环境名称、Python 版本、依赖安装命令和启动命令以 `agent_rules/project/project.md` 为准。

## Static Checks

- 检查 `import` 路径与模块名、函数签名、默认参数、关键字参数是否与调用方一致。
- 检查类属性、实例属性、返回值字段、`typing`、`dataclass`、`pydantic`、`Enum`、异常分支是否存在明显不一致。
- 检查文件路径、环境变量、配置键名是否与读取方保持一致。

## Syntax Note

- 默认以静态阅读检查为主；若后续任务明确需要解释器级语法检查，也必须避免安装依赖、避免运行测试、避免改变环境。
