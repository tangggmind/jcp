import React, { useCallback, useEffect, useRef, useState } from 'react';
import { AlertCircle, Camera, Check, Copy, FileText, Image as ImageIcon, Loader2, RefreshCw, X } from 'lucide-react';
import { ClipboardSetText } from '../../wailsjs/runtime/runtime';
import { reviewService, type ReviewScreenCaptureResult } from '../services/reviewService';

interface ReviewOcrDialogProps {
  isOpen: boolean;
  onClose: () => void;
}

interface ClientRectState {
  left: number;
  top: number;
  width: number;
  height: number;
}

interface Point {
  x: number;
  y: number;
}

type OcrPhase = 'preview' | 'selecting';

export const ReviewOcrDialog: React.FC<ReviewOcrDialogProps> = ({ isOpen, onClose }) => {
  const imageRef = useRef<HTMLImageElement | null>(null);
  const dragStartRef = useRef<Point | null>(null);
  const [phase, setPhase] = useState<OcrPhase>('preview');
  const [screen, setScreen] = useState<ReviewScreenCaptureResult | null>(null);
  const [selection, setSelection] = useState<ClientRectState | null>(null);
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
    setSelection(null);
    try {
      const nextScreen = await reviewService.captureScreen();
      setScreen(nextScreen);
      setPhase('selecting');
    } catch (err) {
      setError(err instanceof Error ? err.message : '截图失败');
      setPhase('preview');
    } finally {
      setCapturing(false);
    }
  }, []);

  useEffect(() => {
    if (!isOpen) return;
    setPhase('preview');
    setScreen(null);
    setPreviewDataUrl('');
    setResult('');
    setCopied(false);
    setError('');
    setSelection(null);
    void startCapture();
  }, [isOpen, startCapture]);

  useEffect(() => {
    if (!isOpen) return;
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        if (phase === 'selecting' && previewDataUrl) {
          setPhase('preview');
          setSelection(null);
          return;
        }
        onClose();
      }
    };
    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [isOpen, onClose, phase, previewDataUrl]);

  const getImageRelativePoint = (event: React.PointerEvent): Point | null => {
    const image = imageRef.current;
    if (!image) return null;
    const bounds = getRenderedImageRect(image);
    if (
      event.clientX < bounds.left ||
      event.clientX > bounds.left + bounds.width ||
      event.clientY < bounds.top ||
      event.clientY > bounds.top + bounds.height
    ) {
      return null;
    }
    return { x: event.clientX, y: event.clientY };
  };

  const handlePointerDown = (event: React.PointerEvent<HTMLDivElement>) => {
    if (phase !== 'selecting') return;
    const point = getImageRelativePoint(event);
    if (!point) return;
    event.currentTarget.setPointerCapture(event.pointerId);
    dragStartRef.current = point;
    setSelection({ left: point.x, top: point.y, width: 0, height: 0 });
  };

  const handlePointerMove = (event: React.PointerEvent<HTMLDivElement>) => {
    if (phase !== 'selecting' || !dragStartRef.current) return;
    const image = imageRef.current;
    if (!image) return;
    const bounds = getRenderedImageRect(image);
    const currentX = clamp(event.clientX, bounds.left, bounds.left + bounds.width);
    const currentY = clamp(event.clientY, bounds.top, bounds.top + bounds.height);
    const start = dragStartRef.current;
    setSelection(normalizeClientRect(start.x, start.y, currentX, currentY));
  };

  const handlePointerUp = async (event: React.PointerEvent<HTMLDivElement>) => {
    if (event.currentTarget.hasPointerCapture(event.pointerId)) {
      event.currentTarget.releasePointerCapture(event.pointerId);
    }
    if (phase !== 'selecting' || !dragStartRef.current || !screen) {
      dragStartRef.current = null;
      return;
    }
    const image = imageRef.current;
    if (!image) {
      dragStartRef.current = null;
      return;
    }
    const bounds = getRenderedImageRect(image);
    const finalSelection = normalizeClientRect(
      dragStartRef.current.x,
      dragStartRef.current.y,
      clamp(event.clientX, bounds.left, bounds.left + bounds.width),
      clamp(event.clientY, bounds.top, bounds.top + bounds.height),
    );
    dragStartRef.current = null;

    if (finalSelection.width < 8 || finalSelection.height < 8) {
      setSelection(null);
      return;
    }

    try {
      const dataUrl = await cropSelection(screen.dataBase64, finalSelection, imageRef.current);
      setPreviewDataUrl(dataUrl);
      setPhase('preview');
      setSelection(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : '裁剪截图失败');
    }
  };

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

  if (phase === 'selecting' && screen?.dataBase64) {
    return (
      <div
        className="fixed inset-0 z-[140] cursor-crosshair select-none bg-black"
        onPointerDown={handlePointerDown}
        onPointerMove={handlePointerMove}
        onPointerUp={(event) => void handlePointerUp(event)}
      >
        <div className="pointer-events-none fixed left-1/2 top-4 z-[142] -translate-x-1/2 rounded-2xl border border-white/20 bg-black/70 px-4 py-2 text-center shadow-2xl backdrop-blur">
          <div className="text-sm font-semibold text-white">按住鼠标拖拽框选 OCR 区域</div>
          <div className="mt-1 text-xs text-white/60">松开鼠标后进入预览，Esc 取消</div>
        </div>
        <button
          type="button"
          onClick={onClose}
          className="fixed right-4 top-4 z-[143] rounded-xl bg-black/60 p-2 text-white/80 transition-colors hover:bg-white/15 hover:text-white"
        >
          <X className="h-5 w-5" />
        </button>
        <img
          ref={imageRef}
          src={screen.dataBase64}
          className="h-full w-full object-contain"
          draggable={false}
          alt="全屏截图"
        />
        {selection ? (
          <div
            className="pointer-events-none fixed z-[142] border-2 border-cyan-300 bg-cyan-300/10 shadow-[0_0_0_9999px_rgba(0,0,0,0.45)]"
            style={{
              left: selection.left,
              top: selection.top,
              width: selection.width,
              height: selection.height,
            }}
          />
        ) : null}
      </div>
    );
  }

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
              <p className="fin-text-tertiary text-xs">框选截图后，点击 OCR 解析将图片发送给当前默认 AI。</p>
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
                  正在最小化并截图...
                </div>
              ) : previewDataUrl ? (
                <img
                  src={previewDataUrl}
                  alt="OCR 截图预览"
                  className="max-h-full max-w-full rounded-xl border fin-divider object-contain shadow-lg"
                />
              ) : (
                <div className="fin-text-tertiary rounded-xl border border-dashed fin-divider px-6 py-10 text-center text-sm">
                  截图完成后在这里预览框选区域。
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

function clamp(value: number, min: number, max: number): number {
  return Math.max(min, Math.min(max, value));
}

function normalizeClientRect(startX: number, startY: number, endX: number, endY: number): ClientRectState {
  return {
    left: Math.min(startX, endX),
    top: Math.min(startY, endY),
    width: Math.abs(endX - startX),
    height: Math.abs(endY - startY),
  };
}

function cropSelection(sourceDataUrl: string, selection: ClientRectState, imageElement: HTMLImageElement | null): Promise<string> {
  return new Promise((resolve, reject) => {
    if (!imageElement) {
      reject(new Error('截图预览未加载'));
      return;
    }
    const imageBounds = getRenderedImageRect(imageElement);
    const scaleX = imageElement.naturalWidth / imageBounds.width;
    const scaleY = imageElement.naturalHeight / imageBounds.height;
    const sx = Math.max(0, Math.round((selection.left - imageBounds.left) * scaleX));
    const sy = Math.max(0, Math.round((selection.top - imageBounds.top) * scaleY));
    const sw = Math.max(1, Math.round(selection.width * scaleX));
    const sh = Math.max(1, Math.round(selection.height * scaleY));

    const img = new Image();
    img.onload = () => {
      const canvas = document.createElement('canvas');
      canvas.width = sw;
      canvas.height = sh;
      const ctx = canvas.getContext('2d');
      if (!ctx) {
        reject(new Error('浏览器不支持 Canvas 截图裁剪'));
        return;
      }
      ctx.drawImage(img, sx, sy, sw, sh, 0, 0, sw, sh);
      resolve(canvas.toDataURL('image/png'));
    };
    img.onerror = () => reject(new Error('截图图片加载失败'));
    img.src = sourceDataUrl;
  });
}

function getRenderedImageRect(imageElement: HTMLImageElement): ClientRectState {
  const bounds = imageElement.getBoundingClientRect();
  const naturalWidth = imageElement.naturalWidth;
  const naturalHeight = imageElement.naturalHeight;
  if (naturalWidth <= 0 || naturalHeight <= 0 || bounds.width <= 0 || bounds.height <= 0) {
    return {
      left: bounds.left,
      top: bounds.top,
      width: bounds.width,
      height: bounds.height,
    };
  }

  const imageRatio = naturalWidth / naturalHeight;
  const boxRatio = bounds.width / bounds.height;
  let renderedWidth = bounds.width;
  let renderedHeight = bounds.height;

  if (boxRatio > imageRatio) {
    renderedWidth = bounds.height * imageRatio;
  } else {
    renderedHeight = bounds.width / imageRatio;
  }

  return {
    left: bounds.left + (bounds.width - renderedWidth) / 2,
    top: bounds.top + (bounds.height - renderedHeight) / 2,
    width: renderedWidth,
    height: renderedHeight,
  };
}
