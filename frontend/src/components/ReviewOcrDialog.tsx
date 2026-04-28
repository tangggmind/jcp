import React, { useCallback, useEffect, useState } from 'react';
import { AlertCircle, Camera, Check, Copy, FileText, Image as ImageIcon, Loader2, RefreshCw, X } from 'lucide-react';
import { ClipboardSetText } from '../../wailsjs/runtime/runtime';
import { reviewService } from '../services/reviewService';

interface ReviewOcrDialogProps {
  isOpen: boolean;
  onClose: () => void;
}

export const ReviewOcrDialog: React.FC<ReviewOcrDialogProps> = ({ isOpen, onClose }) => {
  const [previewDataUrl, setPreviewDataUrl] = useState('');
  const [result, setResult] = useState('');
  const [error, setError] = useState('');
  const [capturing, setCapturing] = useState(false);
  const [parsing, setParsing] = useState(false);
  const [copied, setCopied] = useState(false);

  const startCapture = useCallback(async () => {
    setCapturing(true);
    setError('');
    setResult('');
    setCopied(false);
    setPreviewDataUrl('');
    try {
      const screenClip = await reviewService.captureScreenClip();
      setPreviewDataUrl(screenClip.dataBase64);
    } catch (err) {
      setError(err instanceof Error ? err.message : '截图失败');
    } finally {
      setCapturing(false);
    }
  }, []);

  useEffect(() => {
    if (!isOpen) return;
    setPreviewDataUrl('');
    setResult('');
    setCopied(false);
    setError('');
    void startCapture();
  }, [isOpen, startCapture]);

  useEffect(() => {
    if (!isOpen) return;
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        onClose();
      }
    };
    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [isOpen, onClose]);

  const runOcr = useCallback(async () => {
    if (!previewDataUrl) {
      setError('请先框选截图区域');
      return;
    }
    setParsing(true);
    setError('');
    try {
      const ocr = await reviewService.ocrImage({
        dataBase64: previewDataUrl,
        mimeType: 'image/png',
      });
      setResult(ocr.text);
      setCopied(false);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'OCR 解析失败');
    } finally {
      setParsing(false);
    }
  }, [previewDataUrl]);

  const copyResult = useCallback(async () => {
    if (!result) return;
    setError('');
    try {
      const ok = await ClipboardSetText(result);
      if (!ok) {
        throw new Error('写入剪贴板失败');
      }
      setCopied(true);
      window.setTimeout(() => setCopied(false), 1800);
    } catch (err) {
      setError(err instanceof Error ? err.message : '复制失败');
    }
  }, [result]);

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-[120] flex items-center justify-center bg-black/55 p-4 backdrop-blur-sm">
      <section className="fin-panel-strong flex h-[88vh] w-full max-w-6xl flex-col overflow-hidden rounded-2xl border fin-divider shadow-2xl">
        <header className="flex items-center justify-between border-b fin-divider px-5 py-4">
          <div className="flex items-center gap-3">
            <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-accent/10 text-accent-2">
              <Camera className="h-5 w-5" />
            </div>
            <div>
              <h3 className="fin-text-primary text-base font-semibold">AI OCR 识别</h3>
              <p className="fin-text-tertiary text-xs">使用 Windows 原生截图框选区域，再发送给当前默认 AI 解析。</p>
            </div>
          </div>
          <button
            type="button"
            onClick={onClose}
            className="fin-text-secondary fin-hover rounded-lg p-2 transition-colors"
          >
            <X className="h-5 w-5" />
          </button>
        </header>

        {error ? (
          <div className="mx-5 mt-4 flex items-center gap-2 rounded-xl border border-red-500/30 bg-red-500/10 px-4 py-3 text-sm text-red-200">
            <AlertCircle className="h-4 w-4" />
            {error}
          </div>
        ) : null}

        <main className="grid min-h-0 flex-1 grid-cols-1 gap-4 overflow-hidden p-5 lg:grid-cols-[1.1fr_0.9fr]">
          <div className="fin-panel-soft flex min-h-0 flex-col overflow-hidden rounded-2xl border fin-divider">
            <div className="flex items-center justify-between border-b fin-divider px-4 py-3">
              <div className="flex items-center gap-2 fin-text-primary text-sm font-semibold">
                <ImageIcon className="h-4 w-4" />
                截图预览
              </div>
              <button
                type="button"
                onClick={() => void startCapture()}
                disabled={capturing || parsing}
                className="inline-flex items-center gap-2 rounded-lg bg-accent/15 px-3 py-1.5 text-xs font-medium text-accent-2 transition-colors hover:bg-accent/25 disabled:cursor-not-allowed disabled:opacity-50"
              >
                {capturing ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <RefreshCw className="h-3.5 w-3.5" />}
                重新截图
              </button>
            </div>
            <div className="flex min-h-0 flex-1 items-center justify-center overflow-auto p-4">
              {capturing ? (
                <div className="fin-text-secondary flex items-center gap-2 text-sm">
                  <Loader2 className="h-4 w-4 animate-spin" />
                  正在启动系统截图，请直接在屏幕上框选区域...
                </div>
              ) : previewDataUrl ? (
                <img
                  src={previewDataUrl}
                  alt="OCR 截图预览"
                  className="max-h-full max-w-full rounded-xl border fin-divider object-contain shadow-lg"
                />
              ) : (
                <div className="fin-text-tertiary rounded-xl border border-dashed fin-divider px-6 py-10 text-center text-sm">
                  点击“重新截图”后，使用 Windows 截图框选要识别的区域。
                </div>
              )}
            </div>
          </div>

          <div className="fin-panel-soft flex min-h-0 flex-col overflow-hidden rounded-2xl border fin-divider">
            <div className="flex flex-wrap items-center justify-between gap-2 border-b fin-divider px-4 py-3">
              <div className="flex items-center gap-2 fin-text-primary text-sm font-semibold">
                <FileText className="h-4 w-4" />
                解析结果
              </div>
              <div className="flex items-center gap-2">
                <button
                  type="button"
                  onClick={() => void runOcr()}
                  disabled={parsing || capturing || !previewDataUrl}
                  className="inline-flex items-center gap-2 rounded-lg bg-accent px-3 py-1.5 text-xs font-medium text-white transition-colors hover:bg-accent-2 disabled:cursor-not-allowed disabled:opacity-50"
                >
                  {parsing ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : null}
                  OCR 解析
                </button>
                <button
                  type="button"
                  onClick={() => void copyResult()}
                  disabled={parsing || !result}
                  className="inline-flex items-center gap-2 rounded-lg bg-accent/15 px-3 py-1.5 text-xs font-medium text-accent-2 transition-colors hover:bg-accent/25 disabled:cursor-not-allowed disabled:opacity-50"
                >
                  {copied ? <Check className="h-3.5 w-3.5" /> : <Copy className="h-3.5 w-3.5" />}
                  {copied ? '已复制' : '复制结果'}
                </button>
                <button
                  type="button"
                  onClick={() => void startCapture()}
                  disabled={capturing || parsing}
                  className="inline-flex items-center gap-2 rounded-lg bg-accent/15 px-3 py-1.5 text-xs font-medium text-accent-2 transition-colors hover:bg-accent/25 disabled:cursor-not-allowed disabled:opacity-50"
                >
                  <RefreshCw className="h-3.5 w-3.5" />
                  重新截图
                </button>
              </div>
            </div>
            <div className="min-h-0 flex-1 overflow-auto p-4">
              {parsing ? (
                <div className="fin-text-secondary flex items-center gap-2 text-sm">
                  <Loader2 className="h-4 w-4 animate-spin" />
                  AI 正在解析图片...
                </div>
              ) : result ? (
                <pre className="fin-text-primary whitespace-pre-wrap break-words text-sm leading-6">{result}</pre>
              ) : (
                <div className="fin-text-tertiary rounded-xl border border-dashed fin-divider px-5 py-8 text-sm leading-6">
                  点击“ OCR 解析”后，AI 识别结果会显示在这里。
                </div>
              )}
            </div>
          </div>
        </main>
      </section>
    </div>
  );
};
