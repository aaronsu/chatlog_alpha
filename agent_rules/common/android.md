# Android Agent Rules

## Scope

- 任务命中 Android 标记时，除仓库根目录 `AGENTS.md` 和 `agent_rules/common/common.md` 外，还必须读取本文件。

## Edit Rules

- 仅进行编码、静态检查、语法检查和引用联动检查；不执行任何 Gradle / Android Gradle 编译、构建、打包、测试命令，包括 `gradle`、`gradlew`、`./gradlew`、`gradlew.bat`。
- 修改 XML 时，尺寸必须使用 `@dimen/dp_x` 或 `@dimen/t_dp_x`，禁止直接写 `14dp`、`14sp` 等裸值。
- 除非任务明确要求，保持资源名、id、binding 变量名、包名、清单声明稳定。
- 布局改动后，检查对应 `Activity`、`Fragment`、`Dialog`、`Adapter`、`ViewHolder`、Binding / 数据绑定代码；资源改动后，检查 `layout`、`drawable`、`string`、`color`、`dimen`、`manifest` 引用。

## Static Checks

- 检查 Kotlin / Java 导入、资源 id、binding 字段、include 布局、DataBinding 变量是否一致。
- 检查 `Fragment` / `Activity` / `Dialog` 生命周期中的新增代码位置是否合理。
- 检查 Adapter、ViewHolder、listener、callback、Manifest、路由、intent extra、Bundle key 的调用链和引用是否闭合。

## Figma MCP

- Android 任务使用 `figma-mcp` 时，必须优先且仅使用 `get_design_context` 获取设计样式信息，禁止使用 `get_screenshot` 代替设计读取。
- 若 `figma-mcp` 的 `get_design_context` 调用超时、失败或无法返回有效设计上下文，必须立即停止当前实现任务并反馈用户；禁止在未拿到设计上下文时自行推断、补全或自由发挥样式。
- Figma 画板宽度为 360px 时，Android 默认按 `1px ≈ 1dp` 落地；禁止仅凭实机截图主观整体缩放尺寸。实现前必须读取对应节点的 Figma 尺寸，分别确认聊天列表、输入框预览、文件卡片等具体节点尺寸；不同场景禁止互相套用尺寸。只有当 Figma 画板宽度不是 360px，或目标 Android 页面设计基准不是 360dp 时，才按比例换算。
