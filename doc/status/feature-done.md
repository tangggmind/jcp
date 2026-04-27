# 每日复盘模块 · 已完成功能记录（feature-done）

---

## 记录说明

每条记录仅在 Issue 达到完成条件后追加，字段必须完整。

| 完成时间 | Issue 编号 | 功能摘要 | 测试结果 | 风险说明 |
|----------|------------|----------|----------|----------|
| 2026-04-27 12:20:22 +08:00 | E1-S1-I1 | 新增复盘模块常量、文章、模板、图片、列表、保存、对比请求响应模型 | `go test ./internal/models` 通过 | 已完成本地模型契约验证，尚未进行 Wails 类型生成 |
| 2026-04-27 12:21:36 +08:00 | E1-S1-I2 | 新增 ReviewService 初始化、复盘目录创建、review.db 打开、SQLite 表和索引迁移、Close 关闭能力 | `go test ./internal/services -run ReviewService` 通过 | 已完成本地迁移验证，App 层容错接入尚未开发 |
| 2026-04-27 12:22:40 +08:00 | E1-S2-I1 | 初始化 `review_templates/default-daily.md` 和默认模板元数据，支持文件/记录缺失时修复且不覆盖用户模板正文 | `go test ./internal/services -run 'DefaultDailyTemplate|ReviewService'` 通过 | 模板 CRUD 与前端模板选择尚未开发 |
| 2026-04-27 12:23:50 +08:00 | E1-S2-I2 | 初始化 `articles/复盘总结.md` 和 `summary_review` 索引记录，支持缺少 Front Matter 时按固定文件名推断为总结文章 | `go test ./internal/services -run 'SummaryArticle|DefaultDailyTemplate|ReviewService'` 通过 | 总结文章删除保护与编辑保存仍待后续 Issue 完成 |
| 2026-04-27 12:25:00 +08:00 | E1-S3-I1 | 新增 Markdown Front Matter 解析、容错摘要生成和本地图片引用扫描工具 | `go test ./internal/services -run 'ParseReviewMarkdown|ScanLocalReviewImageRefs'` 通过 | 保存时重写 Front Matter 与索引重建将在后续 Issue 中接入 |
| 2026-04-27 12:25:58 +08:00 | E1-S3-I2 | 新增复盘相对路径安全解析和临时文件 rename 原子写入工具 | `go test ./internal/services -run 'ResolveReviewSafePath|AtomicWriteReviewFile'` 通过 | Windows 覆盖替换已按本地测试验证，后续保存接口需统一调用 |
| 2026-04-27 12:27:39 +08:00 | E2-S1-I1 | 实现指定自然日每日复盘创建、默认模板变量渲染、Front Matter 写入、文件落盘和索引插入 | `go test ./internal/services -run 'CreateDailyReview'` 通过 | 指定模板不存在时已回退默认模板并返回 warning，前端展示待接入 |
| 2026-04-27 12:29:17 +08:00 | E2-S1-I2 | 实现文章详情读取、列表总复盘置顶、每日复盘倒序、关键词搜索、日期范围筛选和分页上限 | `go test ./internal/services -run 'GetArticle|ListArticles'` 通过 | tags/stocks 复杂检索当前按 JSON 解码后内存过滤，数据量增大后可优化 SQL |
| 2026-04-27 12:30:27 +08:00 | E2-S2-I1 | 实现文章保存、合法 Front Matter 重写、标题推断、图片数量同步、原子写入和索引更新 | `go test ./internal/services -run 'SaveArticle'` 通过 | 索引更新失败会返回 warning，后续重建索引入口可恢复 |
| 2026-04-27 12:31:25 +08:00 | E2-S2-I2 | 实现每日复盘删除、总复盘总结删除保护、Markdown 文件删除、索引删除和图片资产关联解除 | `go test ./internal/services -run 'DeleteArticle'` 通过 | 图片文件保留，图片清理器不属于首期范围 |
| 2026-04-27 12:32:32 +08:00 | E2-S3-I1 | 实现文章索引重建，支持扫描 Markdown 恢复总结/每日复盘、同步外部 Front Matter 修改并跳过非法文件名 | `go test ./internal/services -run 'RebuildIndex'` 通过 | 损坏 Front Matter 会按容错结果入库，完整修复在后续保存时完成 |
| 2026-04-27 12:33:43 +08:00 | E2-S3-I2 | 在 App 中挂载 ReviewService，暴露文章列表、详情、创建、保存、删除、总结和重建索引 Wails API，支持 nil 服务容错 | `go test . -run 'ReviewAPI'` 通过 | Wails 绑定生成尚未执行，前端服务暂用动态绑定访问 |
| 2026-04-27 12:34:50 +08:00 | E3-S1-I1 | 新增 `frontend/src/services/reviewService.ts`，定义复盘文章、模板、图片、对比类型并封装 Wails API 错误处理 | `npm run build` 通过 | 模板、图片、对比 API 后端实现将在后续 Epic 接入 |
| 2026-04-27 12:36:13 +08:00 | E3-S1-I2 | 在主界面顶部新增“每日复盘”入口并挂载全屏 `ReviewDialog` 容器，关闭后保留原页面状态 | `npm run build` 通过 | 工作台内部列表和编辑状态将在后续 Issue 完成 |
| 2026-04-27 12:37:43 +08:00 | E3-S2-I1 | 实现 `ReviewDialog` 工作台状态骨架，打开时加载文章和模板，支持打开文章、模式切换、加载状态和错误提示 | `npm run build` 通过 | 编辑器、预览和完整列表组件将在后续 Issue 拆分完善 |
| 2026-04-27 12:38:47 +08:00 | E3-S2-I2 | 新增 `ReviewArticleList`，支持文章搜索、日期筛选、当前高亮、每日复盘多选、删除确认，并禁用总结文章对比和删除 | `npm run build` 通过 | 筛选当前为前端本地过滤，后续可按规模切换为后端查询 |
| 2026-04-27 12:39:41 +08:00 | E3-S2-I3 | 在工作台中实现新建每日复盘弹窗，支持日期、默认模板选择、创建后自动打开和重复日期打开已有文章提示 | `npm run build` 通过 | 模板列表为空时禁用创建，模板管理能力后续接入 |
| 2026-04-27 12:40:32 +08:00 | E3-S3-I1 | 新增 `ReviewEditor` textarea 编辑器，支持 dirty 状态、保存按钮、Ctrl/Cmd+S 快捷保存和保存失败保留正文 | `npm run build` 通过 | 图片粘贴/拖拽事件入口将在 E4 接入 |
| 2026-04-27 12:41:53 +08:00 | E3-S3-I2 | 新增 `ReviewPreview`，使用 markstream 渲染 Markdown，并支持编辑、预览、分栏模式切换及本地图片占位 | `npm run build` 通过 | 本地图片实际加载依赖 E4 安全图片 API |
| 2026-04-27 12:43:01 +08:00 | E3-S3-I3 | 实现 30 秒防抖自动保存、关闭前保存/放弃确认、切换文章未保存提示和保存中防并发保护，总结文章可编辑保存 | `npm run build` 通过 | 未保存提示使用浏览器确认框，后续可替换为自定义三按钮弹窗 |
| 2026-04-27 12:44:28 +08:00 | E4-S1-I1 | 实现粘贴图片 base64 保存，支持 png/jpeg/webp/gif 校验、10MB 限制、hash 文件名、日期/summary 目录和资产索引 | `go test ./internal/services -run 'SavePastedImage'` 通过 | 同 hash 图片复用同一路径，图片清理器不在本期 |
| 2026-04-27 12:46:11 +08:00 | E4-S1-I2 | 实现网络图片下载，限制 http/https、15 秒超时、HTTP 状态、10MB 体积、Content-Type 和实际内容探测校验 | `go test ./internal/services -run 'DownloadImage|SavePastedImage'` 通过 | 下载失败不会写入半文件，前端仍需保留原 URL |
| 2026-04-27 12:47:16 +08:00 | E4-S1-I3 | 暴露 `SaveReviewPastedImage`、`DownloadReviewImage`、`GetReviewAssetBase64`，并实现 pictures-only data URL 安全读取 | `go test ./internal/services -run 'GetReviewAssetBase64|DownloadImage|SavePastedImage'` 和 `go test . -run 'ReviewAPI'` 通过 | `GetReviewAssetBase64` 返回字符串，错误以文本返回供前端占位 |
| 2026-04-27 12:48:17 +08:00 | E4-S2-I1 | 在 `ReviewEditor` 中实现剪贴板和拖拽图片读取、base64 转换、上传调用和光标位置插入 Markdown 图片语法 | `npm run build` 通过 | 上传失败会展示错误并保留原正文 |
| 2026-04-27 12:49:11 +08:00 | E4-S2-I2 | 实现粘贴网络图片 URL/Markdown 图片语法识别、下载后替换为本地 Markdown 路径，并在预览中加载本地图片 data URL | `npm run build` 通过 | 网络下载失败时保留原始文本并显示错误 |
| 2026-04-27 12:50:23 +08:00 | E5-S1-I1 | 实现模板新建、编辑、删除、默认唯一切换、模板文件落盘、安全 ID 和内置/默认模板删除保护 | `go test ./internal/services -run 'Template'` 通过 | 模板 ID 使用安全短 UUID，避免路径注入 |
| 2026-04-27 12:51:06 +08:00 | E5-S1-I2 | 暴露 `GetReviewTemplates`、`SaveReviewTemplate`、`DeleteReviewTemplate` App API，并按 success/错误文本处理删除结果 | `go test . -run 'ReviewAPI'` 通过 | 保存错误暂通过空 ID 和 Name 错误文本返回，前端会按异常展示 |
| 2026-04-27 12:52:09 +08:00 | E5-S2-I1 | 新增 `ReviewTemplateDialog`，支持模板列表、新建、编辑、复制、设默认和删除自定义模板，内置模板只允许复制 | `npm run build` 通过 | 未保存切换当前直接切换，后续可升级为自定义确认体验 |
| 2026-04-27 12:52:56 +08:00 | E5-S2-I2 | 新建复盘流程已传入 TemplateID，模板描述展示在新建弹窗，内置模板可复制为自定义模板，后端保留未知变量 | `npm run build` 通过 | 自定义模板创建文章通过后端模板渲染路径完成 |
| 2026-04-27 12:54:17 +08:00 | E6-S1-I1 | 实现 `Compare` 基础对比，支持 ArticleIDs 和日期范围，排除总结文章，按日期排序并校验至少 2 篇每日复盘 | `go test ./internal/services -run 'CompareReviewArticles'` 通过 | 段落和频率统计在下一 Issue 补齐 |
| 2026-04-27 12:55:44 +08:00 | E6-S1-I2 | 对比结果补充标准 Markdown 段落提取、300 字截断、标签频率和股票频率统计 | `go test ./internal/services -run 'CompareReviewArticles|ExtractsSectionsAndStats'` 通过 | 段落提取支持首期标准二级标题集合 |
| 2026-04-27 12:56:48 +08:00 | E6-S2-I1 | 新增 `ReviewCompare` 前端对比视图，支持多选后查看横向对比表、标签/股票统计、关键段落和点击日期打开文章 | `go test . -run 'ReviewAPI'` 和 `npm run build` 通过 | 首期不包含导出和 AI 总结 |
| 2026-04-27 12:57:37 +08:00 | E6-S2-I2 | 在复盘工作台新增重建索引入口，二次确认后调用 RebuildReviewIndex 并刷新文章/模板列表，失败显示错误 | `npm run build` 通过 | 重建索引仅扫描 Markdown，不删除文章文件 |
| 2026-04-27 12:58:52 +08:00 | E6-S3-I1 | 完成 Wails 绑定生成、Go 全量测试和前端生产构建，新增复盘模块首版回归验收通过 | `wails generate module`、`go test ./...`、`npm run build` 通过 | Vite 报告 chunk size 警告，不影响构建结果；未执行真实桌面手工点击回归 |
