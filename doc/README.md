# 每日复盘模块 · Openclaw 开发协作规范

## 1. 项目输入

固定读取目录下以下文件作为源文档：
- `doc/产品需求说明书.md`
- `doc/产品功能开发说明书.md`

协作文档目录结构：
- `doc/README.md`
- `doc/epic/`
- `doc/status/todo-list.md`
- `doc/status/feature-done.md`

源文档口径：
- `产品需求说明书.md` 是业务范围来源。
- `产品功能开发说明书.md` 是开发执行来源。
- 若二者存在节奏差异，按更小可交付单元拆分，并在 Epic 风险中记录。

当前已识别的范围节奏差异：
- PRD 的里程碑规划把图片能力列为第二阶段、多日对比列为第三阶段。
- 功能开发说明书把图片保存、网络图片下载和基础多日对比列为第一版必须完成。
- 本拆分保留这些能力为独立 Epic，按依赖顺序推进，不在早期 Issue 中隐式扩展。

## 2. Epic 文件列表

| Epic ID | 文件 | 范围摘要 |
|---------|------|----------|
| E1 | `doc/epic/E1-review-foundation.md` | 后端模型、目录、SQLite、默认模板、总复盘总结、Front Matter 与路径安全基础 |
| E2 | `doc/epic/E2-article-lifecycle.md` | 文章创建、读取、保存、删除、列表搜索筛选、重建索引和文章 Wails API |
| E3 | `doc/epic/E3-review-workbench-ui.md` | 前端复盘入口、工作台、文章列表、Markdown 编辑预览、保存和总结文章编辑 |
| E4 | `doc/epic/E4-review-assets.md` | 粘贴图片、拖拽图片、网络图片下载、本地图片预览和资产索引 |
| E5 | `doc/epic/E5-review-templates.md` | 模板 CRUD、默认模板规则、模板管理 UI、模板选择和复制 |
| E6 | `doc/epic/E6-review-compare-and-acceptance.md` | 多日对比、段落提取、频率统计、索引重建入口和首版验收 |

> 仅在处理某个 Issue 时读取对应 Epic 文件，避免加载全部需求上下文。

## 3. 核心开发原则

1. 严格采用 TDD：先测试，后实现。
2. 一次只允许开发一个 Issue。
3. 每次开发前必须先检查依赖与验收条件。
4. 每次开发后必须更新 `doc/status/todo-list.md` 与 `doc/status/feature-done.md`。
5. 不得跳过阻塞项，不得默认补全缺失需求。
6. 不得跨 Epic 隐式扩展范围。
7. 开发当前 Issue 时，仅加载相关 Epic 文件与状态文件。

## 4. 编号规则

- Epic：`E{n}`
- Story：`E{n}-S{n}`
- Issue：`E{n}-S{n}-I{n}`
- Issue 状态只能使用：`Todo`、`In Progress`、`Test Passed`、`Done`、`Skip`

## 5. 开发顺序

当收到 `openclaw 开始开发` 指令时，按以下顺序执行：

1. 读取 `doc/status/todo-list.md`
2. 找到首个状态为 `Todo` 或 `In Progress` 的 Issue
3. 打开该 Issue 所属 Epic 文件
4. 复述 Issue 目标、依赖和验收标准
5. 先编写测试设计与测试用例
6. 再进行最小实现
7. 验证测试结果
8. 更新状态文档

## 6. 阻塞处理

出现以下任一情况必须停止并反馈：
- 需求缺失
- 接口/字段定义不明确
- 外部依赖不可用
- 验收标准冲突
- 上游 Issue 未完成
- 源文档要求与现有代码结构明显冲突

不得把阻塞项标记为 `Done`。如果只能完成本地验证，应先标记为 `Test Passed`，等待合并或联调确认后再改为 `Done`。

## 7. 测试要求

每个 Issue 至少包含：
- 功能测试
- 边界测试

测试必须映射到验收标准。后端优先补充 Go 单元测试，前端优先补充可执行构建检查和必要的组件/手工测试记录。

## 8. 数据和范围纪律

- Markdown 文件是文章内容真源，SQLite 只保存索引、元数据、模板索引和图片引用关系。
- 所有文章和图片路径必须限制在 `articles`、`pictures`、`review_templates` 目录内。
- 删除文章默认不删除图片，只解除或标记图片关联。
- 总复盘总结文章不可删除。
- 非首期能力不进入当前 Issue：AI 草稿、AI 多日总结、交易记录自动盈亏、行情自动填充、富文本编辑器、图片清理器、导出能力。
