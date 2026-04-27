import React, { useEffect, useMemo, useState } from 'react';
import { reviewService } from '../services/reviewService';

interface ReviewPreviewProps {
  content: string;
}

export const ReviewPreview: React.FC<ReviewPreviewProps> = ({ content }) => {
  const [imageMap, setImageMap] = useState<Record<string, string>>({});
  const previewContent = useMemo(() => stripYamlFrontMatter(content), [content]);
  const localImages = useMemo(() => findLocalImages(previewContent), [previewContent]);

  useEffect(() => {
    let cancelled = false;
    const loadImages = async () => {
      const next: Record<string, string> = {};
      for (const path of localImages) {
        const filePath = path.replace(/^\.\.\//, '');
        try {
          next[path] = await reviewService.getAssetBase64(filePath);
        } catch {
          next[path] = '';
        }
      }
      if (!cancelled) setImageMap(next);
    };
    void loadImages();
    return () => {
      cancelled = true;
    };
  }, [localImages]);

  const renderedContent = useMemo(() => replaceLocalImages(previewContent, imageMap), [previewContent, imageMap]);

  if (!previewContent.trim()) {
    return (
      <div className="fin-panel-soft fin-text-tertiary flex h-full items-center justify-center rounded-2xl border border-dashed fin-divider text-sm">
        暂无可预览内容
      </div>
    );
  }

  return (
    <div className="fin-panel-strong fin-scrollbar h-full overflow-auto rounded-2xl border fin-divider p-5 text-left">
      <div className="review-preview-content max-w-none">
        {renderMarkdown(renderedContent)}
      </div>
    </div>
  );
};

function stripYamlFrontMatter(content: string): string {
  const normalized = content.replace(/^\uFEFF/, '');
  if (!normalized.startsWith('---')) return normalized;
  const match = normalized.match(/^---[ \t]*\r?\n[\s\S]*?\r?\n---[ \t]*(?:\r?\n|$)/);
  return match ? normalized.slice(match[0].length).trimStart() : normalized;
}

function findLocalImages(content: string): string[] {
  return Array.from(content.matchAll(/!\[[^\]]*]\(((?:\.\.\/)?pictures\/[^)\s]+)\)/g)).map(match => match[1]);
}

function replaceLocalImages(content: string, imageMap: Record<string, string>): string {
  return content.replace(/!\[([^\]]*)]\(((?:\.\.\/)?pictures\/[^)\s]+)\)/g, (_match, alt: string, path: string) => {
    if (imageMap[path]) {
      return `![${alt || '图片'}](${imageMap[path]})`;
    }
    const label = alt || '本地图片';
    return `\n> ${imagePlaceholderText(label, path)}\n`;
  });
}

function imagePlaceholderText(label: string, path: string): string {
  return `缺图 ${label}：${path}（图片不存在或读取失败）`;
}

function renderMarkdown(content: string): React.ReactNode[] {
  const lines = content.split(/\r?\n/);
  const nodes: React.ReactNode[] = [];
  let index = 0;

  while (index < lines.length) {
    const line = lines[index];
    const trimmed = line.trim();

    if (!trimmed) {
      index += 1;
      continue;
    }

    const image = trimmed.match(/^!\[([^\]]*)]\(([^)]+)\)$/);
    if (image) {
      nodes.push(
        <figure key={`img-${index}`} className="my-4">
          <img className="max-h-[560px] max-w-full rounded-xl border fin-divider object-contain" src={image[2]} alt={image[1] || '图片'} />
          {image[1] ? <figcaption className="fin-text-tertiary mt-2 text-center text-xs">{image[1]}</figcaption> : null}
        </figure>,
      );
      index += 1;
      continue;
    }

    const heading = trimmed.match(/^(#{1,6})\s+(.+)$/);
    if (heading) {
      const level = heading[1].length;
      const className = headingClassName(level);
      const children = renderInline(heading[2], `h-${index}`);
      if (level === 1) nodes.push(<h1 key={`h-${index}`} className={className}>{children}</h1>);
      else if (level === 2) nodes.push(<h2 key={`h-${index}`} className={className}>{children}</h2>);
      else if (level === 3) nodes.push(<h3 key={`h-${index}`} className={className}>{children}</h3>);
      else nodes.push(<h4 key={`h-${index}`} className={className}>{children}</h4>);
      index += 1;
      continue;
    }

    if (/^---+$/.test(trimmed)) {
      nodes.push(<hr key={`hr-${index}`} className="my-5 border-0 border-t fin-divider" />);
      index += 1;
      continue;
    }

    if (trimmed.startsWith('>')) {
      const quoteLines: string[] = [];
      while (index < lines.length && lines[index].trim().startsWith('>')) {
        quoteLines.push(lines[index].trim().replace(/^>\s?/, ''));
        index += 1;
      }
      nodes.push(
        <blockquote key={`quote-${index}`} className="fin-text-secondary my-4 border-l-4 border-accent/50 bg-accent/10 px-4 py-3 text-sm leading-7">
          {renderInline(quoteLines.join(' '), `quote-${index}`)}
        </blockquote>,
      );
      continue;
    }

    if (/^\|.+\|$/.test(trimmed) && index + 1 < lines.length && /^\|?\s*:?-{3,}:?\s*(\|\s*:?-{3,}:?\s*)+\|?$/.test(lines[index + 1].trim())) {
      const tableLines: string[] = [];
      while (index < lines.length && /^\|.+\|$/.test(lines[index].trim())) {
        tableLines.push(lines[index].trim());
        index += 1;
      }
      nodes.push(renderTable(tableLines, `table-${index}`));
      continue;
    }

    if (/^[-*+]\s+/.test(trimmed) || /^\d+\.\s+/.test(trimmed)) {
      const ordered = /^\d+\.\s+/.test(trimmed);
      const items: string[] = [];
      const pattern = ordered ? /^\d+\.\s+/ : /^[-*+]\s+/;
      while (index < lines.length && pattern.test(lines[index].trim())) {
        items.push(lines[index].trim().replace(pattern, ''));
        index += 1;
      }
      const ListTag = ordered ? 'ol' : 'ul';
      nodes.push(
        <ListTag key={`list-${index}`} className={`fin-text-primary my-3 space-y-1 pl-5 text-sm leading-7 ${ordered ? 'list-decimal' : 'list-disc'}`}>
          {items.map((item, itemIndex) => <li key={itemIndex}>{renderInline(item, `li-${index}-${itemIndex}`)}</li>)}
        </ListTag>,
      );
      continue;
    }

    const paragraphLines: string[] = [];
    while (index < lines.length && lines[index].trim()) {
      const current = lines[index].trim();
      if (
        /^#{1,6}\s+/.test(current) ||
        current.startsWith('>') ||
        /^[-*+]\s+/.test(current) ||
        /^\d+\.\s+/.test(current) ||
        /^!\[([^\]]*)]\(([^)]+)\)$/.test(current) ||
        (/^\|.+\|$/.test(current) && index + 1 < lines.length && /^\|?\s*:?-{3,}:?\s*(\|\s*:?-{3,}:?\s*)+\|?$/.test(lines[index + 1].trim()))
      ) {
        break;
      }
      paragraphLines.push(current);
      index += 1;
    }

    if (paragraphLines.length > 0) {
      nodes.push(
        <p key={`p-${index}`} className="fin-text-primary my-3 text-sm leading-7">
          {renderInline(paragraphLines.join(' '), `p-${index}`)}
        </p>,
      );
    } else {
      index += 1;
    }
  }

  return nodes;
}

function renderTable(lines: string[], key: string): React.ReactNode {
  const rows = lines
    .filter((_, index) => index !== 1)
    .map(line => line.replace(/^\|/, '').replace(/\|$/, '').split('|').map(cell => cell.trim()));
  const [head = [], ...body] = rows;

  return (
    <div key={key} className="my-4 overflow-x-auto">
      <table className="w-full min-w-[640px] border-collapse text-sm">
        <thead>
          <tr className="border-b fin-divider">
            {head.map((cell, index) => (
              <th key={index} className="bg-accent/10 px-3 py-2 text-left font-semibold fin-text-primary">{renderInline(cell, `${key}-h-${index}`)}</th>
            ))}
          </tr>
        </thead>
        <tbody>
          {body.map((row, rowIndex) => (
            <tr key={rowIndex} className="border-b fin-divider">
              {row.map((cell, cellIndex) => (
                <td key={cellIndex} className="px-3 py-2 align-top fin-text-secondary">{renderInline(cell, `${key}-${rowIndex}-${cellIndex}`)}</td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function renderInline(text: string, keyPrefix: string): React.ReactNode[] {
  const nodes: React.ReactNode[] = [];
  const pattern = /(`[^`]+`|\*\*[^*]+\*\*|\*[^*]+\*|\[[^\]]+]\([^)]+\))/g;
  let cursor = 0;
  let match: RegExpExecArray | null;

  while ((match = pattern.exec(text)) !== null) {
    if (match.index > cursor) {
      nodes.push(text.slice(cursor, match.index));
    }
    const token = match[0];
    const key = `${keyPrefix}-${match.index}`;
    if (token.startsWith('`')) {
      nodes.push(<code key={key} className="rounded bg-accent/10 px-1.5 py-0.5 font-mono text-[0.9em] text-accent-2">{token.slice(1, -1)}</code>);
    } else if (token.startsWith('**')) {
      nodes.push(<strong key={key} className="font-semibold fin-text-primary">{token.slice(2, -2)}</strong>);
    } else if (token.startsWith('*')) {
      nodes.push(<em key={key} className="text-accent-2">{token.slice(1, -1)}</em>);
    } else {
      const link = token.match(/^\[([^\]]+)]\(([^)]+)\)$/);
      if (link) {
        nodes.push(
          <a key={key} href={link[2]} target="_blank" rel="noreferrer" className="text-accent-2 underline underline-offset-2">
            {link[1]}
          </a>,
        );
      }
    }
    cursor = match.index + token.length;
  }

  if (cursor < text.length) {
    nodes.push(text.slice(cursor));
  }
  return nodes;
}

function headingClassName(level: number): string {
  const base = 'fin-text-primary font-semibold';
  if (level === 1) return `${base} mb-4 mt-1 text-2xl`;
  if (level === 2) return `${base} mb-3 mt-6 text-xl`;
  if (level === 3) return `${base} mb-2 mt-5 text-lg`;
  return `${base} mb-2 mt-4 text-base`;
}
