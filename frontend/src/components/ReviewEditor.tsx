import React from 'react';
import { Loader2, Save } from 'lucide-react';

interface ReviewEditorProps {
  content: string;
  disabled?: boolean;
  dirty: boolean;
  saving: boolean;
  uploading?: boolean;
  onChange: (content: string) => void;
  onSave: () => void;
  onImageFile?: (file: File) => Promise<string>;
  onNetworkImage?: (text: string) => Promise<string | null>;
}

export const ReviewEditor: React.FC<ReviewEditorProps> = ({
  content,
  disabled = false,
  dirty,
  saving,
  uploading = false,
  onChange,
  onSave,
  onImageFile,
  onNetworkImage,
}) => {
  const insertTextAtCursor = (textarea: HTMLTextAreaElement, text: string) => {
    const start = textarea.selectionStart;
    const end = textarea.selectionEnd;
    const next = content.slice(0, start) + text + content.slice(end);
    onChange(next);
    window.requestAnimationFrame(() => {
      textarea.selectionStart = start + text.length;
      textarea.selectionEnd = start + text.length;
      textarea.focus();
    });
  };

  const handleImageFiles = async (textarea: HTMLTextAreaElement, files: FileList | File[]) => {
    if (!onImageFile) return false;
    const imageFiles = Array.from(files).filter(file => file.type.startsWith('image/'));
    if (imageFiles.length === 0) return false;
    for (const file of imageFiles) {
      const markdown = await onImageFile(file);
      insertTextAtCursor(textarea, `\n${markdown}\n`);
    }
    return true;
  };

  const handleKeyDown = (event: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if ((event.ctrlKey || event.metaKey) && event.key.toLowerCase() === 's') {
      event.preventDefault();
      if (!saving && !disabled) {
        onSave();
      }
    }
  };

  const handlePaste = async (event: React.ClipboardEvent<HTMLTextAreaElement>) => {
    const files = event.clipboardData.files;
    if (files && files.length > 0) {
      event.preventDefault();
      await handleImageFiles(event.currentTarget, files);
      return;
    }
    const text = event.clipboardData.getData('text/plain');
    if (!text || !onNetworkImage) return;
    const markdown = await onNetworkImage(text);
    if (markdown) {
      event.preventDefault();
      insertTextAtCursor(event.currentTarget, markdown);
    }
  };

  const handleDrop = async (event: React.DragEvent<HTMLTextAreaElement>) => {
    const files = event.dataTransfer.files;
    if (!files || files.length === 0) return;
    event.preventDefault();
    await handleImageFiles(event.currentTarget, files);
  };

  return (
    <div className="fin-panel-strong flex h-full min-h-0 flex-col rounded-2xl border fin-divider">
      <div className="flex items-center justify-between border-b fin-divider px-3 py-2">
        <span className="fin-text-tertiary text-xs">{uploading ? '图片上传中...' : dirty ? '未保存修改' : '内容已同步'}</span>
        <button
          type="button"
          onClick={onSave}
          disabled={saving || disabled}
          className="inline-flex items-center gap-2 rounded-lg bg-accent px-3 py-1.5 text-xs font-medium text-white transition-colors hover:bg-accent-2 disabled:cursor-not-allowed disabled:opacity-50"
        >
          {saving ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <Save className="h-3.5 w-3.5" />}
          保存
        </button>
      </div>
      <textarea
        value={content}
        disabled={disabled}
        onChange={(event) => onChange(event.target.value)}
        onKeyDown={handleKeyDown}
        onPaste={(event) => void handlePaste(event)}
        onDrop={(event) => void handleDrop(event)}
        onDragOver={(event) => event.preventDefault()}
        className="fin-input min-h-0 flex-1 resize-none rounded-b-2xl border-0 p-4 font-mono text-sm leading-6 disabled:cursor-not-allowed disabled:opacity-60"
        spellCheck={false}
      />
    </div>
  );
};
