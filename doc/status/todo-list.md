# 每日复盘模块 · 开发 Issue 进度清单（todo-list）

> **说明：** 本清单基于 `doc/epic/` 下各 Epic 文件正文显式展开的 Issue 生成。
> **目录：** Epic 拆分文件位于 `doc/epic/`，状态文件位于 `doc/status/`。
> **口径：** PRD 负责业务范围，详细开发说明书负责开发执行与 Issue 编号。
> 完成一个 Issue 后**必须**更新状态与测试结果，再进入下一个。
> Issue 编号规则固定为：`E1-S1-I1`。

---

## 状态约定

| 状态 | 含义 |
|------|------|
| `Todo` | 待开发，条件未就绪或排队中 |
| `In Progress` | 正在开发 |
| `Test Passed` | 开发完成，本地测试全部通过 |
| `Done` | 已合并主干，联调验证通过 |
| `Skip` | 本版本跳过（标注原因） |

---

## 开发前更新规则

1. 开发开始前先将对应 Issue 状态改为 `In Progress`。
2. 本地验证通过后更新“测试”列，并将状态改为 `Test Passed`。
3. 合并主干或联调通过后，才能改为 `Done`。
4. 如需拆分新增任务，先更新对应 `doc/epic/*.md`，再补录到本表。
5. 如遇阻塞，不得继续下一个 Issue，必须先记录阻塞原因。

---

## 进度表

| Issue | 描述 | 状态 | 依赖 | 测试 | 备注 |
|------|------|------|------|------|------|
| E1-S1-I1 | 定义复盘模型与请求响应结构 | Todo | 无 | 待补充 | 后续 Wails 绑定依赖这些 JSON 字段 |
| E1-S1-I2 | 初始化 ReviewService 目录和 SQLite 迁移 | Todo | E1-S1-I1 | 待补充 | 使用纯 Go SQLite 驱动 |
| E1-S2-I1 | 初始化默认每日复盘模板 | Todo | E1-S1-I2 | 待补充 | 默认模板必须始终存在 |
| E1-S2-I2 | 初始化总复盘总结文章 | Todo | E1-S1-I2 | 待补充 | 总复盘总结不可按普通文章删除 |
| E1-S3-I1 | 实现 Front Matter、摘要和图片引用解析工具 | Todo | E1-S1-I1 | 待补充 | Front Matter 解析失败需容错 |
| E1-S3-I2 | 实现路径安全校验和原子写入工具 | Todo | E1-S1-I2 | 待补充 | 所有文件写入必须防路径穿越 |
| E2-S1-I1 | 创建指定日期每日复盘 | Todo | E1-S2-I1, E1-S3-I2 | 待补充 | 自然日/交易日规则按首期自然日处理 |
| E2-S1-I2 | 获取文章详情、列表搜索和筛选 | Todo | E2-S1-I1, E1-S3-I1 | 待补充 | 总复盘总结置顶 |
| E2-S2-I1 | 保存文章并同步 Front Matter 和索引 | Todo | E2-S1-I2, E1-S3-I1, E1-S3-I2 | 待补充 | Markdown 文件是真源 |
| E2-S2-I2 | 删除每日复盘并处理孤立图片引用 | Todo | E2-S1-I2 | 待补充 | 删除文章不删除 pictures 图片 |
| E2-S3-I1 | 重建文章索引 | Todo | E2-S2-I1 | 待补充 | 删除 review.db 后应可恢复列表 |
| E2-S3-I2 | 暴露文章管理 Wails API | Todo | E2-S3-I1 | 待补充 | 服务初始化失败不得导致 API panic |
| E3-S1-I1 | 新增前端复盘服务封装和类型 | Todo | E2-S3-I2 | 待补充 | 需要 Wails 绑定可用 |
| E3-S1-I2 | 集成主界面每日复盘入口和 Dialog 容器 | Todo | E3-S1-I1 | 待补充 | 首期采用全屏弹窗 |
| E3-S2-I1 | 实现 ReviewDialog 工作台状态骨架 | Todo | E3-S1-I2 | 待补充 | 使用组件内状态管理 |
| E3-S2-I2 | 实现 ReviewArticleList 列表、筛选、多选和删除入口 | Todo | E3-S2-I1 | 待补充 | 总复盘总结不可对比和删除 |
| E3-S2-I3 | 实现新建每日复盘弹窗流程 | Todo | E3-S2-I2 | 待补充 | 重复日期提示打开已有或取消 |
| E3-S3-I1 | 实现 ReviewEditor 文本编辑和保存快捷键 | Todo | E3-S2-I1 | 待补充 | textarea 首期实现 |
| E3-S3-I2 | 实现 ReviewPreview 和编辑/预览/分栏模式 | Todo | E3-S3-I1 | 待补充 | 本地图片完整预览依赖 E4 |
| E3-S3-I3 | 实现自动保存、未保存离开提示和总结文章编辑 | Todo | E3-S3-I2 | 待补充 | 自动保存需避免并发提交 |
| E4-S1-I1 | 实现粘贴图片保存接口 | Todo | E1-S3-I2, E2-S1-I2 | 待补充 | 默认单图最大 10 MB |
| E4-S1-I2 | 实现网络图片下载接口 | Todo | E4-S1-I1 | 待补充 | 只允许 http/https |
| E4-S1-I3 | 暴露图片和安全预览 Wails API | Todo | E4-S1-I2 | 待补充 | base64 API 只能读 pictures |
| E4-S2-I1 | 实现编辑器粘贴和拖拽图片上传 | Todo | E4-S1-I3, E3-S3-I1 | 待补充 | 失败时不丢失正文 |
| E4-S2-I2 | 实现网络图片识别、替换和预览加载 | Todo | E4-S2-I1, E3-S3-I2 | 待补充 | 下载失败保留原 URL |
| E5-S1-I1 | 实现模板 CRUD 和默认模板规则 | Todo | E1-S2-I1, E1-S3-I2 | 待补充 | 内置模板不可删除 |
| E5-S1-I2 | 暴露模板 Wails API | Todo | E5-S1-I1 | 待补充 | 删除类接口非 success 视为失败 |
| E5-S2-I1 | 实现 ReviewTemplateDialog 模板管理界面 | Todo | E5-S1-I2, E3-S2-I1 | 待补充 | 内置模板可复制不可删除 |
| E5-S2-I2 | 集成模板选择、变量渲染和模板复制 | Todo | E5-S2-I1, E3-S2-I3, E2-S1-I1 | 待补充 | 未知模板变量保留原样 |
| E6-S1-I1 | 实现 CompareReviewArticles 后端基础对比 | Todo | E2-S1-I2 | 待补充 | 最少 2 篇每日复盘 |
| E6-S1-I2 | 实现 Markdown 标准段落提取和频率统计 | Todo | E6-S1-I1, E1-S3-I1 | 待补充 | 段落超过 300 字截断 |
| E6-S2-I1 | 实现 ReviewCompare 前端对比展示 | Todo | E6-S1-I2, E3-S2-I2 | 待补充 | 仅首期横向表格和摘要 |
| E6-S2-I2 | 实现索引重建入口和结果刷新 | Todo | E2-S3-I1, E3-S2-I1 | 待补充 | 重建索引不删除文章文件 |
| E6-S3-I1 | Wails 绑定生成、构建和回归验收 | Todo | E3-S3-I3, E4-S2-I2, E5-S2-I2, E6-S2-I2 | 待补充 | 环境缺依赖时记录阻塞，不标记 Done |
