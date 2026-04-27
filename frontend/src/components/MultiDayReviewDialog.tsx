import React, { useCallback, useEffect, useState } from 'react';
import { AlertCircle, Loader2, RefreshCw, X } from 'lucide-react';
import { reviewService, REVIEW_TYPE_DAILY, type ReviewArticleDetail } from '../services/reviewService';
import { ReviewPreview } from './ReviewPreview';

interface MultiDayReviewDialogProps {
  isOpen: boolean;
  onClose: () => void;
}

export const MultiDayReviewDialog: React.FC<MultiDayReviewDialogProps> = ({ isOpen, onClose }) => {
  const [articles, setArticles] = useState<ReviewArticleDetail[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  const loadRecentArticles = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const result = await reviewService.listArticles({
        type: REVIEW_TYPE_DAILY,
        page: 1,
        pageSize: 5,
      });
      const details = await Promise.all(result.items.map(article => reviewService.getArticle(article.id)));
      setArticles(details);
    } catch (err) {
      setError(err instanceof Error ? err.message : '加载多日复盘失败');
      setArticles([]);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    if (isOpen) {
      void loadRecentArticles();
    }
  }, [isOpen, loadRecentArticles]);

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-[95] bg-black/45 p-3 backdrop-blur-sm sm:p-5">
      <section className="fin-panel-strong flex h-full flex-col overflow-hidden rounded-2xl border fin-divider shadow-2xl">
        <header className="flex items-center justify-between gap-3 border-b fin-divider px-5 py-4">
          <div>
            <h2 className="fin-text-primary text-base font-semibold">多日复盘</h2>
            <p className="fin-text-tertiary text-xs">
              并列展示最近 {articles.length || 0} 篇每日复盘，最多 5 篇，仅用于阅读对比。
            </p>
          </div>
          <div className="flex items-center gap-2">
            <button
              type="button"
              onClick={() => void loadRecentArticles()}
              disabled={loading}
              className="fin-text-secondary fin-hover inline-flex items-center gap-2 rounded-lg px-3 py-2 text-xs transition-colors disabled:opacity-50"
              title="刷新多日复盘"
            >
              <RefreshCw className={`h-4 w-4 ${loading ? 'animate-spin' : ''}`} />
              刷新
            </button>
            <button
              type="button"
              onClick={onClose}
              className="fin-text-secondary fin-hover rounded-lg p-2 transition-colors"
              title="关闭多日复盘"
            >
              <X className="h-5 w-5" />
            </button>
          </div>
        </header>

        {error ? (
          <div className="m-4 flex items-center gap-2 rounded-xl border border-red-500/30 bg-red-500/10 px-4 py-3 text-sm text-red-200">
            <AlertCircle className="h-4 w-4" />
            {error}
          </div>
        ) : null}

        <main className="min-h-0 flex-1 overflow-hidden p-4">
          {loading && articles.length === 0 ? (
            <div className="fin-panel-soft fin-text-secondary flex h-full items-center justify-center gap-2 rounded-2xl border fin-divider text-sm">
              <Loader2 className="h-4 w-4 animate-spin" />
              加载最近复盘...
            </div>
          ) : articles.length === 0 ? (
            <div className="fin-panel-soft fin-text-tertiary flex h-full items-center justify-center rounded-2xl border border-dashed fin-divider text-sm">
              暂无每日复盘文章
            </div>
          ) : (
            <div className="fin-scrollbar h-full overflow-x-auto overflow-y-hidden pb-2">
              <div className="grid h-full grid-flow-col auto-cols-[minmax(340px,1fr)] gap-4">
                {articles.map(detail => (
                  <article key={detail.article.id} className="fin-panel flex min-h-0 flex-col overflow-hidden rounded-2xl border fin-divider">
                    <header className="border-b fin-divider px-4 py-3 text-left">
                      <div className="flex items-start justify-between gap-3">
                        <div className="min-w-0">
                          <h3 className="fin-text-primary truncate text-sm font-semibold">{detail.article.title}</h3>
                          <p className="fin-text-tertiary mt-1 text-xs">{detail.article.date}</p>
                        </div>
                        <span className="shrink-0 rounded-full bg-accent/10 px-2 py-1 text-xs text-accent-2">
                          {detail.article.emotion || '未记录情绪'}
                        </span>
                      </div>
                      <div className="fin-text-tertiary mt-2 flex flex-wrap gap-2 text-xs">
                        <span>盈亏：{formatProfitLoss(detail.article.profitLoss)}</span>
                        <span>纪律：{detail.article.disciplineScore ?? '-'}</span>
                        {detail.article.stocks.length > 0 ? <span>股票：{detail.article.stocks.join('、')}</span> : null}
                      </div>
                    </header>
                    <div className="min-h-0 flex-1 p-3">
                      <ReviewPreview content={detail.content} />
                    </div>
                  </article>
                ))}
              </div>
            </div>
          )}
        </main>
      </section>
    </div>
  );
};

function formatProfitLoss(value: number): string {
  if (!Number.isFinite(value) || value === 0) return '-';
  return `${value > 0 ? '+' : ''}${value.toFixed(2)}`;
}
