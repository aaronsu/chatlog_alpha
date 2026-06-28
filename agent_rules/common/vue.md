# Vue Agent Rules

## Scope

- 任务命中 Vue 标记时，除仓库根目录 `AGENTS.md` 和 `agent_rules/common/common.md` 外，还必须读取本文件。

## Edit Rules

- 仅进行编码、静态检查、语法检查和引用联动检查；若用户要求 UI 测试，优先使用 Playwright。
- 不执行 `npm`、`pnpm`、`yarn` 的安装、编译、构建、打包、测试命令。
- 保持现有组件拆分、命名风格、状态管理方式，不做与任务无关的 `template` / `style` 重排。
- 修改 `template` 后，检查对应 `script`、`style`、props、emits、slots、指令、样式选择器；修改路由、状态管理或 API 调用后，检查直接引用它们的页面、组件、composable、store。

## Static Checks

- 检查 `import` 路径、组件注册、props、emits、事件名是否一致。
- 检查 `ref`、`reactive`、`computed`、`watch`、`defineProps`、`defineEmits` 的名称和使用是否匹配。
- 检查 `v-if` / `v-for` / `key` / `slot` / `class` / `style` 绑定是否存在明显错误；若使用 TypeScript，再检查类型引用、接口字段、返回值结构。

## Style Rule

- 优先保持最小必要修改，避免只为格式或个人偏好重排大段 `template` / `style`。
