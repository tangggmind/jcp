import React, { useEffect, useState } from 'react';
import { Copy, Save, Trash2, X } from 'lucide-react';
import { reviewService, type ReviewTemplate } from '../services/reviewService';

interface ReviewTemplateDialogProps {
  isOpen: boolean;
  templates: ReviewTemplate[];
  onClose: () => void;
  onChanged: () => void;
}

export const ReviewTemplateDialog: React.FC<ReviewTemplateDialogProps> = ({ isOpen, templates, onClose, onChanged }) => {
  const [editing, setEditing] = useState<ReviewTemplate | null>(null);
  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [content, setContent] = useState('');
  const [isDefault, setIsDefault] = useState(false);
  const [error, setError] = useState('');
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    if (isOpen && !editing && templates.length > 0) {
      selectTemplate(templates[0]);
    }
  }, [editing, isOpen, templates]);

  if (!isOpen) return null;

  const selectTemplate = (template: ReviewTemplate) => {
    setEditing(template);
    setName(template.name);
    setDescription(template.description);
    setContent(template.content);
    setIsDefault(template.isDefault);
    setError('');
  };

  const newTemplate = () => {
    setEditing(null);
    setName('');
    setDescription('');
    setContent('# {{title}}\n\n## 今日市场\n\n');
    setIsDefault(false);
    setError('');
  };

  const copyTemplate = (template: ReviewTemplate) => {
    setEditing(null);
    setName(`${template.name} 副本`);
    setDescription(template.description);
    setContent(template.content);
    setIsDefault(false);
    setError('');
  };

  const save = async () => {
    setSaving(true);
    setError('');
    try {
      const saved = await reviewService.saveTemplate({
        id: editing?.isBuiltin ? '' : editing?.id || '',
        name,
        description,
        content,
        isDefault,
      });
      if (!saved.id) {
        throw new Error(saved.name || '保存模板失败');
      }
      onChanged();
      selectTemplate(saved);
    } catch (err) {
      setError(err instanceof Error ? err.message : '保存模板失败');
    } finally {
      setSaving(false);
    }
  };

  const remove = async (template: ReviewTemplate) => {
    if (template.isBuiltin) return;
    if (!window.confirm(`确认删除模板「${template.name}」？`)) return;
    try {
      await reviewService.deleteTemplate(template.id);
      newTemplate();
      onChanged();
    } catch (err) {
      setError(err instanceof Error ? err.message : '删除模板失败');
    }
  };

  return (
    <div className="fixed inset-0 z-[110] flex items-center justify-center bg-black/45 p-4 backdrop-blur-sm">
      <div className="fin-panel-strong grid h-[82vh] w-full max-w-5xl grid-cols-[280px_1fr] overflow-hidden rounded-2xl border fin-divider shadow-2xl">
        <aside className="border-r fin-divider">
          <div className="flex items-center justify-between border-b fin-divider p-3">
            <span className="fin-text-primary text-sm font-semibold">模板</span>
            <button onClick={newTemplate} className="rounded-lg bg-accent px-3 py-1.5 text-xs font-medium text-white hover:bg-accent-2">新建</button>
          </div>
          <div className="space-y-2 overflow-y-auto p-3">
            {templates.map(template => (
              <button
                key={template.id}
                type="button"
                onClick={() => selectTemplate(template)}
                className={`w-full rounded-xl border p-3 text-left text-sm transition-colors ${editing?.id === template.id ? 'border-accent/50 bg-accent/10' : 'fin-panel-soft fin-divider hover:border-accent/40'}`}
              >
                <div className="fin-text-primary font-medium">{template.name}</div>
                <div className="fin-text-tertiary mt-1 text-xs">
                  {template.isBuiltin ? '内置' : '自定义'}{template.isDefault ? ' · 默认' : ''}
                </div>
              </button>
            ))}
          </div>
        </aside>

        <section className="flex min-h-0 flex-col">
          <header className="flex items-center justify-between border-b fin-divider p-4">
            <div>
              <h3 className="fin-text-primary text-base font-semibold">模板管理</h3>
              <p className="fin-text-tertiary text-xs">内置模板可复制，不可直接删除。</p>
            </div>
            <button onClick={onClose} className="fin-text-secondary fin-hover rounded-lg p-2">
              <X className="h-4 w-4" />
            </button>
          </header>

          <div className="min-h-0 flex-1 space-y-3 overflow-y-auto p-4">
            {error ? <div className="rounded-xl border border-red-500/30 bg-red-500/10 px-3 py-2 text-sm text-red-200">{error}</div> : null}
            <label className="block">
              <div className="flex items-center gap-3">
                <span className="fin-text-secondary w-20 shrink-0 text-right text-xs font-medium">模板名称</span>
                <input value={name} onChange={event => setName(event.target.value)} placeholder="例如：每日交易复盘" className="fin-input min-w-0 flex-1 rounded-xl px-3 py-2 text-sm" />
              </div>
              <span className="fin-text-tertiary ml-[92px] mt-1 block text-xs">用于在新建每日复盘时识别模板。</span>
            </label>
            <label className="block">
              <div className="flex items-center gap-3">
                <span className="fin-text-secondary w-20 shrink-0 text-right text-xs font-medium">模板描述</span>
                <input value={description} onChange={event => setDescription(event.target.value)} placeholder="说明该模板适用的复盘场景" className="fin-input min-w-0 flex-1 rounded-xl px-3 py-2 text-sm" />
              </div>
              <span className="fin-text-tertiary ml-[92px] mt-1 block text-xs">会显示在模板选择区域，帮助区分不同模板。</span>
            </label>
            <label className="fin-text-secondary inline-flex items-center gap-2 text-sm">
              <input type="checkbox" checked={isDefault} onChange={event => setIsDefault(event.target.checked)} />
              设为默认模板
            </label>
            <label className="block">
              <span className="fin-text-secondary text-xs font-medium">模板正文（Markdown）</span>
              <textarea value={content} onChange={event => setContent(event.target.value)} className="fin-input mt-1 h-80 w-full resize-none rounded-xl p-3 font-mono text-sm leading-6" />
              <span className="fin-text-tertiary mt-1 block text-xs">支持变量占位符，例如 {'{{title}}'}、{'{{date}}'}。</span>
            </label>
          </div>

          <footer className="flex justify-between border-t fin-divider p-4">
            <div className="flex gap-2">
              {editing ? (
                <button type="button" onClick={() => copyTemplate(editing)} className="fin-text-secondary fin-hover inline-flex items-center gap-2 rounded-xl border fin-divider px-3 py-2 text-sm">
                  <Copy className="h-4 w-4" />
                  复制
                </button>
              ) : null}
              {editing && !editing.isBuiltin ? (
                <button type="button" onClick={() => void remove(editing)} className="inline-flex items-center gap-2 rounded-xl border border-red-500/30 px-3 py-2 text-sm text-red-300 hover:bg-red-500/10">
                  <Trash2 className="h-4 w-4" />
                  删除
                </button>
              ) : null}
            </div>
            <button type="button" onClick={() => void save()} disabled={saving} className="inline-flex items-center gap-2 rounded-xl bg-accent px-4 py-2 text-sm font-medium text-white hover:bg-accent-2 disabled:opacity-50">
              <Save className="h-4 w-4" />
              保存
            </button>
          </footer>
        </section>
      </div>
    </div>
  );
};
