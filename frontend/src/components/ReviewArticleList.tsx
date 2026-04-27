import React, { useMemo, useState } from 'react';
import { CheckSquare, FileText, Image, Search, Square, Trash2 } from 'lucide-react';
import { REVIEW_TYPE_SUMMARY, type ReviewArticle } from '../services/reviewService';

interface ReviewArticleListProps {
  articles: ReviewArticle[];
  currentId?: string;
  selectedIds: string[];
  onSelectedIdsChange: (ids: string[]) => void;
  onOpen: (article: ReviewArticle) => void;
  onDelete: (article: ReviewArticle) => void;
  openingId?: string;
}

export const ReviewArticleList: React.FC<ReviewArticleListProps> = ({
  articles,
  currentId,
  selectedIds,
  onSelectedIdsChange,
  onOpen,
  onDelete,
  openingId,
}) => {
  const [query, setQuery] = useState('');
  const [startDate, setStartDate] = useState('');
  const [endDate, setEndDate] = useState('');

  const filtered = useMemo(() => {
    const normalizedQuery = query.trim().toLowerCase();
    return articles.filter(article => {
      if (startDate && article.date && article.date < startDate) return false;
      if (endDate && article.date && article.date > endDate) return false;
      if (!normalizedQuery) return true;
      const haystack = [
        article.title,
        article.summary,
        article.templateName,
        ...(article.tags ?? []),
        ...(article.stocks ?? []),
      ].join(' ').toLowerCase();
      return haystack.includes(normalizedQuery);
    });
  }, [articles, endDate, query, startDate]);

  const toggleSelected = (article: ReviewArticle) => {
    if (article.type === REVIEW_TYPE_SUMMARY) return;
    if (selectedIds.includes(article.id)) {
      onSelectedIdsChange(selectedIds.filter(id => id !== article.id));
      return;
    }
    onSelectedIdsChange([...selectedIds, article.id]);
  };

  const confirmDelete = (article: ReviewArticle) => {
    if (article.type === REVIEW_TYPE_SUMMARY) return;
    if (window.confirm(`确认删除「${article.title}」？图片文件会保留。`)) {
      onDelete(article);
    }
  };

  return (
    <div className="flex h-full flex-col">
      <div className="space-y-2 border-b fin-divider p-3">
        <div className="relative">
          <Search className="fin-text-tertiary pointer-events-none absolute left-3 top-2.5 h-4 w-4" />
          <input
            value={query}
            onChange={(event) => setQuery(event.target.value)}
            placeholder="搜索标题、摘要、标签、股票"
            className="fin-input w-full rounded-xl py-2 pl-9 pr-3 text-sm transition-colors"
          />
        </div>
        <div className="grid grid-cols-2 gap-2">
          <input
            type="date"
            value={startDate}
            onChange={(event) => setStartDate(event.target.value)}
            className="fin-input rounded-lg px-2 py-1.5 text-xs"
          />
          <input
            type="date"
            value={endDate}
            onChange={(event) => setEndDate(event.target.value)}
            className="fin-input rounded-lg px-2 py-1.5 text-xs"
          />
        </div>
      </div>

      <div className="min-h-0 flex-1 overflow-y-auto p-3">
        {filtered.length === 0 ? (
          <div className="fin-text-tertiary rounded-xl border border-dashed fin-divider p-5 text-center text-sm">
            没有匹配的复盘文章
          </div>
        ) : (
          <div className="space-y-2">
            {filtered.map(article => {
              const active = currentId === article.id;
              const isSummary = article.type === REVIEW_TYPE_SUMMARY;
              const selected = selectedIds.includes(article.id);
              return (
                <article
                  key={article.id}
                  className={`rounded-xl border p-3 transition-colors ${
                    active
                      ? 'border-accent/50 bg-accent/10'
                      : 'fin-panel-soft fin-divider hover:border-accent/40'
                  }`}
                >
                  <div className="flex items-start gap-2">
                    <button
                      type="button"
                      onClick={() => toggleSelected(article)}
                      disabled={isSummary}
                      className={`mt-0.5 rounded p-1 transition-colors ${
                        isSummary ? 'cursor-not-allowed fin-text-tertiary opacity-40' : 'fin-text-secondary fin-hover hover:text-accent-2'
                      }`}
                      title={isSummary ? '总复盘总结不可参与对比' : '选择用于对比'}
                    >
                      {selected ? <CheckSquare className="h-4 w-4" /> : <Square className="h-4 w-4" />}
                    </button>
                    <button type="button" onClick={() => onOpen(article)} className="min-w-0 flex-1 text-left">
                      <div className="flex items-center justify-between gap-2">
                        <span className="fin-text-primary truncate text-sm font-medium">{article.title}</span>
                        {openingId === article.id ? <span className="text-accent-2 text-xs">打开中</span> : null}
                      </div>
                      <div className="fin-text-tertiary mt-1 flex flex-wrap items-center gap-2 text-xs">
                        <FileText className="h-3.5 w-3.5" />
                        <span>{isSummary ? '总复盘总结' : article.date}</span>
                        {article.templateName ? <span>{article.templateName}</span> : null}
                        {article.imageCount > 0 ? (
                          <span className="inline-flex items-center gap-1">
                            <Image className="h-3.5 w-3.5" />
                            {article.imageCount}
                          </span>
                        ) : null}
                      </div>
                      {article.summary ? <p className="fin-text-secondary mt-2 line-clamp-2 text-xs leading-5">{article.summary}</p> : null}
                    </button>
                    <button
                      type="button"
                      onClick={() => confirmDelete(article)}
                      disabled={isSummary}
                      className={`rounded-lg p-1.5 transition-colors ${
                        isSummary ? 'cursor-not-allowed fin-text-tertiary opacity-40' : 'fin-text-tertiary hover:bg-red-500/10 hover:text-red-300'
                      }`}
                      title={isSummary ? '总复盘总结不可删除' : '删除每日复盘'}
                    >
                      <Trash2 className="h-4 w-4" />
                    </button>
                  </div>
                </article>
              );
            })}
          </div>
        )}
      </div>
    </div>
  );
};
