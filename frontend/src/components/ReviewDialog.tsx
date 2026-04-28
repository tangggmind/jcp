import React, { useCallback, useEffect, useState } from 'react';
import { AlertCircle, BookOpen, Camera, Loader2, Plus, RefreshCw, Settings, SplitSquareHorizontal, X } from 'lucide-react';
import type { Stock } from '../types';
import { reviewService, REVIEW_TYPE_SUMMARY, type ReviewArticle, type ReviewArticleDetail, type ReviewTemplate } from '../services/reviewService';
import { ReviewArticleList } from './ReviewArticleList';
import { ReviewEditor } from './ReviewEditor';
import { ReviewPreview } from './ReviewPreview';
import { ReviewTemplateDialog } from './ReviewTemplateDialog';
import { ReviewCompare } from './ReviewCompare';
import { ReviewOcrDialog } from './ReviewOcrDialog';
import type { CompareReviewResult } from '../services/reviewService';

interface ReviewDialogProps {
  isOpen: boolean;
  onClose: () => void;
  selectedStock?: Stock | null;
}

export const ReviewDialog: React.FC<ReviewDialogProps> = ({ isOpen, onClose, selectedStock }) => {
  const [articles, setArticles] = useState<ReviewArticle[]>([]);
  const [templates, setTemplates] = useState<ReviewTemplate[]>([]);
  const [current, setCurrent] = useState<ReviewArticleDetail | null>(null);
  const [content, setContent] = useState('');
  const [dirty, setDirty] = useState(false);
  const [mode, setMode] = useState<'edit' | 'preview' | 'split' | 'compare'>('preview');
  const [selectedCompareIds, setSelectedCompareIds] = useState<string[]>([]);
  const [loading, setLoading] = useState(false);
  const [openingId, setOpeningId] = useState('');
  const [error, setError] = useState('');
  const [showCreate, setShowCreate] = useState(false);
  const [createDate, setCreateDate] = useState(() => new Date().toISOString().slice(0, 10));
  const [createTemplateId, setCreateTemplateId] = useState('');
  const [creating, setCreating] = useState(false);
  const [saving, setSaving] = useState(false);
  const [uploadingImage, setUploadingImage] = useState(false);
  const [showTemplates, setShowTemplates] = useState(false);
  const [showOcr, setShowOcr] = useState(false);
  const [compareResult, setCompareResult] = useState<CompareReviewResult | null>(null);
  const selectedCreateTemplate = templates.find(template => template.id === createTemplateId);

  const loadWorkbench = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const [articleResult, templateResult] = await Promise.all([
        reviewService.listArticles({ pageSize: 200 }),
        reviewService.listTemplates(),
      ]);
      setArticles(articleResult.items);
      setTemplates(templateResult);
      setCreateTemplateId(prev => prev || templateResult.find(template => template.isDefault)?.id || templateResult[0]?.id || '');
      if (!current && articleResult.items.length > 0) {
        const detail = await reviewService.getArticle(articleResult.items[0].id);
        setCurrent(detail);
        setContent(detail.content);
        setDirty(false);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : '加载复盘工作台失败');
    } finally {
      setLoading(false);
    }
  }, []);

  const openArticle = useCallback(async (article: ReviewArticle) => {
    if (dirty && !window.confirm('当前文章有未保存修改，确认放弃并切换文章？')) {
      return;
    }
    setOpeningId(article.id);
    setError('');
    try {
      const detail = await reviewService.getArticle(article.id);
      setCurrent(detail);
      setContent(detail.content);
      setDirty(false);
    } catch (err) {
      setError(err instanceof Error ? err.message : '打开复盘文章失败');
    } finally {
      setOpeningId('');
    }
  }, [dirty]);

  const deleteArticle = useCallback(async (article: ReviewArticle) => {
    if (article.type === REVIEW_TYPE_SUMMARY) return;
    setError('');
    try {
      await reviewService.deleteArticle(article.id);
      setArticles(prev => prev.filter(item => item.id !== article.id));
      setSelectedCompareIds(prev => prev.filter(id => id !== article.id));
      if (current?.article.id === article.id) {
        setCurrent(null);
        setContent('');
        setDirty(false);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : '删除复盘文章失败');
    }
  }, [current]);

  const createArticle = useCallback(async () => {
    setCreating(true);
    setError('');
    try {
      const detail = await reviewService.createDailyReview({
        date: createDate,
        templateId: createTemplateId || undefined,
        stocks: selectedStock?.symbol ? [selectedStock.symbol] : [],
      });
      setCurrent(detail);
      setContent(detail.content);
      setDirty(false);
      setShowCreate(false);
      const result = await reviewService.listArticles({ pageSize: 200 });
      setArticles(result.items);
    } catch (err) {
      const message = err instanceof Error ? err.message : '创建每日复盘失败';
      const existing = articles.find(article => article.date === createDate);
      if (message.includes('该日期复盘已存在') && existing && window.confirm('该日期复盘已存在，是否打开已有文章？')) {
        setShowCreate(false);
        await openArticle(existing);
      } else {
        setError(message);
      }
    } finally {
      setCreating(false);
    }
  }, [articles, createDate, createTemplateId, openArticle, selectedStock]);

  const saveCurrentArticle = useCallback(async () => {
    if (!current || saving) return;
    setSaving(true);
    setError('');
    try {
      const detail = await reviewService.saveArticle({
        id: current.article.id,
        title: current.article.title,
        content,
      });
      setCurrent(detail);
      setContent(detail.content);
      setDirty(false);
      setArticles(prev => prev.map(article => article.id === detail.article.id ? detail.article : article));
    } catch (err) {
      setError(err instanceof Error ? err.message : '保存复盘文章失败');
    } finally {
      setSaving(false);
    }
  }, [content, current, saving]);

  const uploadImageFile = useCallback(async (file: File) => {
    if (!current) {
      throw new Error('请先打开复盘文章');
    }
    setUploadingImage(true);
    setError('');
    try {
      const dataBase64 = await fileToBase64(file);
      const result = await reviewService.savePastedImage({
        articleId: current.article.id,
        date: current.article.date,
        fileName: file.name,
        mimeType: file.type,
        dataBase64,
      });
      return result.markdownText;
    } catch (err) {
      setError(err instanceof Error ? err.message : '上传图片失败');
      throw err;
    } finally {
      setUploadingImage(false);
    }
  }, [current]);

  const downloadNetworkImage = useCallback(async (text: string) => {
    if (!current) return null;
    const trimmed = text.trim();
    const markdownMatch = trimmed.match(/^!\[([^\]]*)]\((https?:\/\/[^)\s]+)\)$/i);
    const directMatch = trimmed.match(/^https?:\/\/\S+\.(?:png|jpe?g|webp|gif)(?:\?\S*)?$/i);
    const url = markdownMatch?.[2] || directMatch?.[0];
    if (!url) return null;
    setUploadingImage(true);
    setError('');
    try {
      const result = await reviewService.downloadImage({
        articleId: current.article.id,
        date: current.article.date,
        url,
      });
      if (markdownMatch) {
        return `![${markdownMatch[1] || '图片'}](${result.markdownPath})`;
      }
      return result.markdownText;
    } catch (err) {
      setError(err instanceof Error ? err.message : '下载网络图片失败');
      return null;
    } finally {
      setUploadingImage(false);
    }
  }, [current]);

  const insertScreenshotImage = useCallback(async () => {
    if (!current) {
      setError('请先打开复盘文章');
      return null;
    }

    setUploadingImage(true);
    setError('');
    try {
      const screenshot = await reviewService.captureScreenClip();
      const result = await reviewService.savePastedImage({
        articleId: current.article.id,
        date: current.article.date,
        fileName: `screenshot-${Date.now()}.png`,
        mimeType: 'image/png',
        dataBase64: screenshot.dataBase64,
      });
      return result.markdownText;
    } catch (err) {
      setError(err instanceof Error ? err.message : '截图插入失败');
      return null;
    } finally {
      setUploadingImage(false);
    }
  }, [current]);

  const runCompare = useCallback(async () => {
    if (selectedCompareIds.length < 2) {
      setError('至少选择 2 篇每日复盘');
      return;
    }
    setError('');
    try {
      const result = await reviewService.compareArticles({ articleIds: selectedCompareIds });
      setCompareResult(result);
      setMode('compare');
    } catch (err) {
      setError(err instanceof Error ? err.message : '对比失败');
    }
  }, [selectedCompareIds]);

  const rebuildIndex = useCallback(async () => {
    if (!window.confirm('重建索引会扫描 articles 目录并刷新列表，不会删除 Markdown 文件。确认继续？')) return;
    setError('');
    try {
      await reviewService.rebuildIndex();
      await loadWorkbench();
    } catch (err) {
      setError(err instanceof Error ? err.message : '重建索引失败');
    }
  }, [loadWorkbench]);

  const handleClose = useCallback(async () => {
    if (!dirty) {
      onClose();
      return;
    }
    if (window.confirm('当前文章有未保存修改，是否先保存再关闭？')) {
      await saveCurrentArticle();
      onClose();
      return;
    }
    if (window.confirm('放弃未保存修改并关闭复盘工作台？')) {
      onClose();
    }
  }, [dirty, onClose, saveCurrentArticle]);

  useEffect(() => {
    if (isOpen) {
      setMode('preview');
      void loadWorkbench();
    }
  }, [isOpen, loadWorkbench]);

  useEffect(() => {
    if (!isOpen || !dirty || !current || saving) return;
    const timer = window.setTimeout(() => {
      void saveCurrentArticle();
    }, 30000);
    return () => window.clearTimeout(timer);
  }, [current, dirty, isOpen, saveCurrentArticle, saving]);

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-[90] bg-black/45 backdrop-blur-sm">
      <div className="h-full w-full p-3 sm:p-5">
        <section className="fin-panel-strong h-full overflow-hidden rounded-2xl border fin-divider shadow-2xl">
          <header className="flex items-center justify-between border-b fin-divider px-5 py-4">
            <div className="flex items-center gap-3">
              <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-accent/10 text-accent-2">
                <BookOpen className="h-5 w-5" />
              </div>
              <div>
                <h2 className="fin-text-primary text-base font-semibold">每日复盘工作台</h2>
                <p className="fin-text-tertiary text-xs">
                  {selectedStock ? `当前股票：${selectedStock.name} ${selectedStock.symbol}` : '未选择股票，可直接管理复盘文章'}
                </p>
              </div>
            </div>
            <button
              type="button"
              onClick={() => void handleClose()}
              className="fin-text-secondary fin-hover rounded-lg p-2 transition-colors"
              title="关闭复盘工作台"
            >
              <X className="h-5 w-5" />
            </button>
          </header>

          <main className="grid h-[calc(100%-73px)] grid-cols-1 overflow-hidden lg:grid-cols-[320px_1fr]">
            <aside className="fin-panel-soft border-b fin-divider lg:border-b-0 lg:border-r">
              <div className="flex items-center justify-between border-b fin-divider px-4 py-3">
                <div>
                  <div className="fin-text-primary text-sm font-semibold">文章列表</div>
                  <div className="fin-text-tertiary text-xs">{articles.length} 篇文章 · {templates.length} 个模板</div>
                </div>
                <button
                  type="button"
                  onClick={() => void loadWorkbench()}
                  className="fin-text-secondary fin-hover rounded-lg p-2 transition-colors"
                  title="刷新复盘列表"
                >
                  <RefreshCw className={`h-4 w-4 ${loading ? 'animate-spin' : ''}`} />
                </button>
                <button
                  type="button"
                  onClick={() => setShowCreate(true)}
                  className="fin-text-secondary fin-hover rounded-lg p-2 transition-colors"
                  title="新建每日复盘"
                >
                  <Plus className="h-4 w-4" />
                </button>
                <button
                  type="button"
                  onClick={() => setShowTemplates(true)}
                  className="fin-text-secondary fin-hover rounded-lg p-2 transition-colors"
                  title="模板管理"
                >
                  <Settings className="h-4 w-4" />
                </button>
                <button
                  type="button"
                  onClick={() => void rebuildIndex()}
                  className="fin-text-secondary fin-hover rounded-lg px-2 py-2 text-xs transition-colors"
                  title="重建索引"
                >
                  重建
                </button>
              </div>

              <div className="h-[calc(100%-57px)] overflow-hidden">
                {loading && articles.length === 0 ? (
                  <div className="fin-text-secondary flex items-center justify-center gap-2 py-10 text-sm">
                    <Loader2 className="h-4 w-4 animate-spin" />
                    加载中...
                  </div>
                ) : articles.length === 0 ? (
                  <div className="fin-text-tertiary rounded-xl border border-dashed fin-divider p-5 text-center text-sm">
                    暂无复盘文章
                  </div>
                ) : (
                  <ReviewArticleList
                    articles={articles}
                    currentId={current?.article.id}
                    selectedIds={selectedCompareIds}
                    onSelectedIdsChange={setSelectedCompareIds}
                    onOpen={(article) => void openArticle(article)}
                    onDelete={(article) => void deleteArticle(article)}
                    openingId={openingId}
                  />
                )}
              </div>
            </aside>

            <section className="flex min-h-0 flex-col">
              <div className="flex flex-wrap items-center justify-between gap-3 border-b fin-divider px-4 py-3">
                <div>
                  <div className="fin-text-primary text-sm font-semibold">{current?.article.title || '请选择复盘文章'}</div>
                <div className="fin-text-tertiary text-xs">
                  {dirty ? '未保存' : '已同步'} · 模式：{mode} · 对比选择 {selectedCompareIds.length} 篇
                </div>
              </div>
              <button
                type="button"
                onClick={() => setShowOcr(true)}
                className="inline-flex items-center gap-2 rounded-xl bg-accent/15 px-3 py-2 text-xs font-medium text-accent-2 transition-colors hover:bg-accent/25"
              >
                <Camera className="h-3.5 w-3.5" />
                AI OCR
              </button>
              <button
                type="button"
                onClick={() => void runCompare()}
                disabled={selectedCompareIds.length < 2}
                className="rounded-xl bg-accent px-3 py-2 text-xs font-medium text-white disabled:cursor-not-allowed disabled:opacity-40"
              >
                对比
              </button>
              <div className="fin-panel-soft flex items-center gap-1 rounded-xl border fin-divider p-1">
                  {(['edit', 'preview', 'split', 'compare'] as const).map(item => (
                    <button
                      key={item}
                      type="button"
                      onClick={() => {
                        setMode(item);
                        if (item !== 'compare') {
                          setSelectedCompareIds([]);
                        }
                      }}
                      className={`rounded-lg px-3 py-1.5 text-xs transition-colors ${
                        mode === item ? 'bg-accent/20 text-accent-2' : 'fin-text-secondary fin-hover'
                      }`}
                    >
                      {item === 'split' ? <SplitSquareHorizontal className="inline h-3.5 w-3.5" /> : item}
                    </button>
                  ))}
                </div>
              </div>

              {error ? (
                <div className="m-4 flex items-center gap-2 rounded-xl border border-red-500/30 bg-red-500/10 px-4 py-3 text-sm text-red-200">
                  <AlertCircle className="h-4 w-4" />
                  {error}
                </div>
              ) : null}

              <div className="min-h-0 flex-1 overflow-hidden p-4">
                {!current ? (
                  <div className="fin-panel-soft fin-text-tertiary flex h-full items-center justify-center rounded-2xl border border-dashed fin-divider text-sm">
                    选择左侧文章开始编辑或预览
                  </div>
                ) : mode === 'compare' ? (
                  compareResult ? (
                    <ReviewCompare result={compareResult} onOpenArticle={(articleId) => {
                      const article = articles.find(item => item.id === articleId);
                      if (article) void openArticle(article);
                    }} />
                  ) : (
                    <div className="fin-panel-soft fin-text-tertiary flex h-full items-center justify-center rounded-2xl border border-dashed fin-divider text-sm">
                      勾选至少两篇每日复盘后点击“对比”。
                    </div>
                  )
                ) : mode === 'preview' ? (
                  <ReviewPreview content={content} />
                ) : mode === 'split' ? (
                  <div className="grid h-full min-h-0 grid-cols-1 gap-4 xl:grid-cols-2">
                    <ReviewEditor
                      content={content}
                      dirty={dirty}
                      saving={saving}
                      uploading={uploadingImage}
                      disabled={!current}
                      onChange={(value) => {
                        setContent(value);
                        setDirty(true);
                      }}
                      onSave={() => void saveCurrentArticle()}
                      onImageFile={uploadImageFile}
                      onNetworkImage={downloadNetworkImage}
                      onScreenshot={insertScreenshotImage}
                    />
                    <ReviewPreview content={content} />
                  </div>
                ) : (
                  <ReviewEditor
                    content={content}
                    dirty={dirty}
                    saving={saving}
                    uploading={uploadingImage}
                    disabled={!current}
                    onChange={(value) => {
                      setContent(value);
                      setDirty(true);
                    }}
                    onSave={() => void saveCurrentArticle()}
                    onImageFile={uploadImageFile}
                    onNetworkImage={downloadNetworkImage}
                    onScreenshot={insertScreenshotImage}
                  />
                )}
              </div>
            </section>
          </main>
        </section>
      </div>

      {showCreate ? (
        <div className="fixed inset-0 z-[100] flex items-center justify-center bg-black/45 p-4 backdrop-blur-sm">
          <div className="fin-panel-strong w-full max-w-md rounded-2xl border fin-divider p-5 shadow-2xl">
            <div className="mb-4 flex items-center justify-between">
              <div>
                <h3 className="fin-text-primary text-base font-semibold">新建每日复盘</h3>
                <p className="fin-text-tertiary text-xs">选择日期和模板，创建后自动打开文章。</p>
              </div>
              <button
                type="button"
                onClick={() => setShowCreate(false)}
                className="fin-text-secondary fin-hover rounded-lg p-2"
              >
                <X className="h-4 w-4" />
              </button>
            </div>

            <div className="space-y-3">
              <label className="block">
                <span className="fin-text-secondary text-xs">复盘日期</span>
                <input
                  type="date"
                  value={createDate}
                  onChange={(event) => setCreateDate(event.target.value)}
                  className="fin-input mt-1 w-full rounded-xl px-3 py-2 text-sm"
                />
              </label>
              <label className="block">
                <span className="fin-text-secondary text-xs">模板</span>
                <select
                  value={createTemplateId}
                  onChange={(event) => setCreateTemplateId(event.target.value)}
                  disabled={templates.length === 0}
                  className="fin-input mt-1 w-full rounded-xl px-3 py-2 text-sm"
                >
                  {templates.length === 0 ? (
                    <option value="">模板加载失败或为空</option>
                  ) : (
                    templates.map(template => (
                      <option key={template.id} value={template.id}>
                        {template.name}{template.isDefault ? '（默认）' : ''}
                      </option>
                    ))
                  )}
                </select>
                {selectedCreateTemplate?.description ? (
                  <p className="fin-text-tertiary mt-1 text-xs leading-5">{selectedCreateTemplate.description}</p>
                ) : null}
              </label>
            </div>

            <div className="mt-5 flex justify-end gap-2">
              <button
                type="button"
                onClick={() => setShowCreate(false)}
                className="fin-text-secondary fin-hover rounded-xl px-4 py-2 text-sm"
              >
                取消
              </button>
              <button
                type="button"
                onClick={() => void createArticle()}
                disabled={creating || !createDate || templates.length === 0}
                className="inline-flex items-center gap-2 rounded-xl bg-accent px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-accent-2 disabled:cursor-not-allowed disabled:opacity-50"
              >
                {creating ? <Loader2 className="h-4 w-4 animate-spin" /> : null}
                创建
              </button>
            </div>
          </div>
        </div>
      ) : null}
      <ReviewTemplateDialog
        isOpen={showTemplates}
        templates={templates}
        onClose={() => setShowTemplates(false)}
        onChanged={() => void loadWorkbench()}
      />
      <ReviewOcrDialog isOpen={showOcr} onClose={() => setShowOcr(false)} />
    </div>
  );
};

function fileToBase64(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = () => {
      const result = typeof reader.result === 'string' ? reader.result : '';
      resolve(result);
    };
    reader.onerror = () => reject(reader.error || new Error('读取图片失败'));
    reader.readAsDataURL(file);
  });
}
