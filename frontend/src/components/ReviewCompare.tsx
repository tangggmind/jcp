import React from 'react';
import type { CompareReviewResult } from '../services/reviewService';

interface ReviewCompareProps {
  result: CompareReviewResult;
  onOpenArticle: (articleId: string) => void;
}

export const ReviewCompare: React.FC<ReviewCompareProps> = ({ result, onOpenArticle }) => {
  return (
    <div className="fin-panel-strong fin-scrollbar h-full overflow-auto rounded-2xl border fin-divider p-4 text-left">
      <div className="mb-4 grid gap-3 md:grid-cols-2">
        <StatPanel title="高频标签" items={result.tagStats} />
        <StatPanel title="高频股票" items={result.stockStats} />
      </div>
      <div className="overflow-x-auto">
        <table className="w-full min-w-[900px] border-collapse text-sm">
          <thead className="fin-text-tertiary text-left text-xs">
            <tr className="border-b fin-divider">
              <th className="p-2">日期</th>
              <th className="p-2">标题</th>
              <th className="p-2">摘要</th>
              <th className="p-2">标签</th>
              <th className="p-2">股票</th>
              <th className="p-2">盈亏</th>
              <th className="p-2">情绪</th>
              <th className="p-2">纪律</th>
            </tr>
          </thead>
          <tbody>
            {result.items.map(item => (
              <tr key={item.articleId} className="fin-text-secondary border-b fin-divider align-top">
                <td className="p-2">
                  <button onClick={() => onOpenArticle(item.articleId)} className="text-accent-2 hover:underline">{item.date}</button>
                </td>
                <td className="fin-text-primary p-2">{item.title}</td>
                <td className="fin-text-secondary max-w-xs p-2 text-xs leading-5">{item.summary}</td>
                <td className="p-2">{item.tags.join('、')}</td>
                <td className="p-2">{item.stocks.join('、')}</td>
                <td className="p-2">{item.profitLoss}</td>
                <td className="p-2">{item.emotion || '-'}</td>
                <td className="p-2">{item.disciplineScore ?? '-'}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      <div className="mt-4 grid gap-3 md:grid-cols-2">
        {result.items.map(item => (
          <section key={item.articleId} className="fin-panel-soft rounded-xl border fin-divider p-3">
            <h4 className="fin-text-primary mb-2 text-sm font-semibold">{item.date} 关键段落</h4>
            {Object.entries(item.sections).length === 0 ? (
              <p className="fin-text-tertiary text-xs">暂无标准段落</p>
            ) : (
              Object.entries(item.sections).map(([name, text]) => (
                <div key={name} className="mb-2">
                  <div className="text-accent-2 text-xs">{name}</div>
                  <p className="fin-text-secondary whitespace-pre-wrap text-xs leading-5">{text || '-'}</p>
                </div>
              ))
            )}
          </section>
        ))}
      </div>
    </div>
  );
};

const StatPanel: React.FC<{ title: string; items: { name: string; count: number }[] }> = ({ title, items }) => (
  <div className="fin-panel-soft rounded-xl border fin-divider p-3">
    <div className="fin-text-primary mb-2 text-sm font-semibold">{title}</div>
    <div className="flex flex-wrap gap-2">
      {items.length === 0 ? <span className="fin-text-tertiary text-xs">暂无统计</span> : items.map(item => (
        <span key={item.name} className="rounded-full bg-accent/10 px-2 py-1 text-xs text-accent-2">{item.name} × {item.count}</span>
      ))}
    </div>
  </div>
);
