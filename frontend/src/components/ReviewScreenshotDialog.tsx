import React, { useCallback, useEffect, useRef, useState } from 'react';
import { AlertCircle, Loader2, X } from 'lucide-react';
import { reviewService, type ReviewScreenCaptureResult } from '../services/reviewService';

interface ReviewScreenshotDialogProps {
  isOpen: boolean;
  onClose: () => void;
  onCaptured: (dataUrl: string) => void;
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

export const ReviewScreenshotDialog: React.FC<ReviewScreenshotDialogProps> = ({ isOpen, onClose, onCaptured }) => {
  const imageRef = useRef<HTMLImageElement | null>(null);
  const dragStartRef = useRef<Point | null>(null);
  const [screen, setScreen] = useState<ReviewScreenCaptureResult | null>(null);
  const [selection, setSelection] = useState<ClientRectState | null>(null);
  const [error, setError] = useState('');
  const [capturing, setCapturing] = useState(false);
  const [cropping, setCropping] = useState(false);

  const startCapture = useCallback(async () => {
    setCapturing(true);
    setError('');
    setScreen(null);
    setSelection(null);
    try {
      setScreen(await reviewService.captureScreen());
    } catch (err) {
      setError(err instanceof Error ? err.message : '截图失败');
    } finally {
      setCapturing(false);
    }
  }, []);

  useEffect(() => {
    if (!isOpen) return;
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
    if (!screen?.dataBase64 || capturing || cropping) return;
    const point = getImageRelativePoint(event);
    if (!point) return;
    event.currentTarget.setPointerCapture(event.pointerId);
    dragStartRef.current = point;
    setSelection({ left: point.x, top: point.y, width: 0, height: 0 });
  };

  const handlePointerMove = (event: React.PointerEvent<HTMLDivElement>) => {
    if (!dragStartRef.current || !imageRef.current || cropping) return;
    const bounds = getRenderedImageRect(imageRef.current);
    const currentX = clamp(event.clientX, bounds.left, bounds.left + bounds.width);
    const currentY = clamp(event.clientY, bounds.top, bounds.top + bounds.height);
    const start = dragStartRef.current;
    setSelection(normalizeClientRect(start.x, start.y, currentX, currentY));
  };

  const handlePointerUp = async (event: React.PointerEvent<HTMLDivElement>) => {
    if (event.currentTarget.hasPointerCapture(event.pointerId)) {
      event.currentTarget.releasePointerCapture(event.pointerId);
    }
    if (!dragStartRef.current || !screen?.dataBase64 || !imageRef.current) {
      dragStartRef.current = null;
      return;
    }
    const bounds = getRenderedImageRect(imageRef.current);
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

    setCropping(true);
    setError('');
    try {
      const dataUrl = await cropSelection(screen.dataBase64, finalSelection, imageRef.current);
      onCaptured(dataUrl);
    } catch (err) {
      setError(err instanceof Error ? err.message : '裁剪截图失败');
    } finally {
      setCropping(false);
      setSelection(null);
    }
  };

  if (!isOpen) return null;

  return (
    <div
      className="fixed inset-0 z-[140] cursor-crosshair select-none bg-black"
      onPointerDown={handlePointerDown}
      onPointerMove={handlePointerMove}
      onPointerUp={(event) => void handlePointerUp(event)}
    >
      <div className="pointer-events-none fixed left-1/2 top-4 z-[142] -translate-x-1/2 rounded-2xl border border-white/20 bg-black/70 px-4 py-2 text-center shadow-2xl backdrop-blur">
        <div className="text-sm font-semibold text-white">{capturing ? '正在隐藏窗口并截图...' : cropping ? '正在裁剪并插入截图...' : '按住鼠标拖拽框选截图区域'}</div>
        <div className="mt-1 text-xs text-white/60">松开鼠标后自动保存到复盘图片目录，Esc 取消</div>
      </div>
      <button
        type="button"
        onClick={onClose}
        className="fixed right-4 top-4 z-[143] rounded-xl bg-black/60 p-2 text-white/80 transition-colors hover:bg-white/15 hover:text-white"
      >
        <X className="h-5 w-5" />
      </button>

      {error ? (
        <div className="fixed left-1/2 top-20 z-[143] flex -translate-x-1/2 items-center gap-2 rounded-xl border border-red-300/40 bg-red-950/80 px-4 py-3 text-sm text-red-100 shadow-2xl">
          <AlertCircle className="h-4 w-4" />
          {error}
        </div>
      ) : null}

      {capturing ? (
        <div className="flex h-full w-full items-center justify-center text-sm text-white/70">
          <Loader2 className="mr-2 h-4 w-4 animate-spin" />
          正在截取屏幕...
        </div>
      ) : screen?.dataBase64 ? (
        <img
          ref={imageRef}
          src={screen.dataBase64}
          className="h-full w-full object-contain"
          draggable={false}
          alt="全屏截图"
        />
      ) : (
        <div className="flex h-full w-full items-center justify-center text-sm text-white/70">截图失败，请关闭后重试</div>
      )}

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
