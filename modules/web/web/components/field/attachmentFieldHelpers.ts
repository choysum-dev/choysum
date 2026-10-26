// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { normalizeOptionalString } from '@/core/service/utils/normalization';

export type AttachmentLike = Record<string, any>;

export function isAttachmentObject(raw: unknown): raw is AttachmentLike {
  return !!raw && typeof raw === 'object' && !Array.isArray(raw);
}

export function resolveDescriptor(raw: unknown): AttachmentLike | undefined {
  if (!isAttachmentObject(raw)) return undefined;
  const descriptor = raw.descriptor;
  return descriptor && typeof descriptor === 'object' && !Array.isArray(descriptor) ? (descriptor as AttachmentLike) : undefined;
}

export function resolveBindingId(raw: unknown): string | undefined {
  if (!isAttachmentObject(raw)) return undefined;
  const descriptor = resolveDescriptor(raw);
  return normalizeOptionalString(raw.attachmentBindingId ?? raw.bindingId ?? raw.Id ?? descriptor?.id);
}

export function resolveObjectId(raw: unknown): string | undefined {
  if (!isAttachmentObject(raw)) return undefined;
  const fromAttachmentObjectId = normalizeOptionalString(raw.attachmentObjectId);
  if (fromAttachmentObjectId) return fromAttachmentObjectId;
  return normalizeOptionalString(raw.objectId);
}

export function resolveFileName(raw: unknown): string | undefined {
  if (typeof raw === 'string') return normalizeOptionalString(raw);
  if (!isAttachmentObject(raw)) return undefined;
  const descriptor = resolveDescriptor(raw);
  return normalizeOptionalString(raw.fileName ?? raw.displayName ?? raw.name ?? raw.originalFileName ?? descriptor?.fileName);
}

export function resolveMimeType(raw: unknown): string | undefined {
  if (!isAttachmentObject(raw)) return undefined;
  const descriptor = resolveDescriptor(raw);
  return normalizeOptionalString(raw.mimeType ?? raw.contentType ?? raw.clientContentType ?? raw.proposedContentType ?? descriptor?.mimeType);
}

export function resolveSizeBytes(raw: unknown): number | undefined {
  if (!isAttachmentObject(raw)) return undefined;
  const descriptor = resolveDescriptor(raw);
  const source = raw.sizeBytes ?? raw.size ?? raw.file?.size ?? descriptor?.sizeBytes;
  const size = typeof source === 'number' ? source : Number(source);
  return Number.isFinite(size) && size >= 0 ? size : undefined;
}

/** Prefer download/url; fall back to preview/thumbnail for linkable hosts. */
export function resolveDownloadUrl(raw: unknown): string | undefined {
  if (!isAttachmentObject(raw)) return undefined;
  const descriptor = resolveDescriptor(raw);
  const fromDownloadUrl = normalizeOptionalString(raw.downloadUrl);
  if (fromDownloadUrl) return fromDownloadUrl;
  const fromUrl = normalizeOptionalString(raw.url);
  if (fromUrl) return fromUrl;
  const fromPreviewUrl = normalizeOptionalString(raw.previewUrl);
  if (fromPreviewUrl) return fromPreviewUrl;
  const fromThumbnailUrl = normalizeOptionalString(raw.thumbnailUrl);
  if (fromThumbnailUrl) return fromThumbnailUrl;
  const fromDescriptorDownload = normalizeOptionalString(descriptor?.downloadUrl);
  if (fromDescriptorDownload) return fromDescriptorDownload;
  return normalizeOptionalString(descriptor?.previewUrl);
}

/** Prefer preview/thumbnail for image media; fall back to download/url. */
export function resolvePreviewUrl(raw: unknown): string | undefined {
  if (!isAttachmentObject(raw)) return undefined;
  const descriptor = resolveDescriptor(raw);
  const fromPreviewUrl = normalizeOptionalString(raw.previewUrl);
  if (fromPreviewUrl) return fromPreviewUrl;
  const fromUrl = normalizeOptionalString(raw.url);
  if (fromUrl) return fromUrl;
  const fromThumbnailUrl = normalizeOptionalString(raw.thumbnailUrl);
  if (fromThumbnailUrl) return fromThumbnailUrl;
  const fromDownloadUrl = normalizeOptionalString(raw.downloadUrl);
  if (fromDownloadUrl) return fromDownloadUrl;
  const fromDescriptorPreview = normalizeOptionalString(descriptor?.previewUrl);
  if (fromDescriptorPreview) return fromDescriptorPreview;
  return normalizeOptionalString(descriptor?.downloadUrl);
}

export function hasAttachment(raw: unknown): boolean {
  if (raw == null) return false;
  if (typeof raw === 'string') return !!normalizeOptionalString(raw);
  if (isAttachmentObject(raw)) {
    const kind = normalizeOptionalString(raw.kind)?.toLowerCase();
    if (kind === 'noop' || kind === 'clear') return false;
    return !!(
      resolveBindingId(raw) ||
      resolveFileName(raw) ||
      resolveObjectId(raw) ||
      resolvePreviewUrl(raw) ||
      resolveDownloadUrl(raw) ||
      raw.file
    );
  }
  return true;
}

export function formatAttachmentSize(sizeBytes: number | undefined): string | undefined {
  if (sizeBytes == null) return undefined;
  if (sizeBytes < 1024) return `${sizeBytes} B`;
  const units = ['KB', 'MB', 'GB', 'TB'];
  let value = sizeBytes / 1024;
  let unitIndex = 0;
  while (value >= 1024 && unitIndex < units.length - 1) {
    value /= 1024;
    unitIndex += 1;
  }
  const digits = value >= 10 ? 0 : 1;
  return `${value.toFixed(digits)} ${units[unitIndex]}`;
}

export function toAttachmentMetaText(raw: unknown): string | undefined {
  const mimeType = resolveMimeType(raw);
  const sizeText = formatAttachmentSize(resolveSizeBytes(raw));
  const summary = [mimeType, sizeText].filter(Boolean).join(' · ');
  return summary || undefined;
}

/** Compact file row: size only (mime belongs in tooltip). */
export function toAttachmentCompactMetaText(raw: unknown): string | undefined {
  return formatAttachmentSize(resolveSizeBytes(raw));
}

/** Full meta for title/tooltip on compact file rows. */
export function toAttachmentTooltipText(raw: unknown): string | undefined {
  return toAttachmentMetaText(raw) || resolveFileName(raw);
}

/** Enforce accept tokens beyond the native input hint (drag/drop and *.ext). */
export function isAcceptAllowed(file: File, accept: string): boolean {
  const name = String(file.name || '').toLowerCase();
  const type = String(file.type || '').toLowerCase();
  const tokens = accept
    .split(',')
    .map(t => t.trim().toLowerCase())
    .filter(Boolean);
  const restrictive = tokens.filter(t => t !== '*' && t !== '*/*');
  if (!restrictive.length) return true;
  return restrictive.some(token => {
    if (token.startsWith('*.')) return name.endsWith(token.slice(1));
    if (token.startsWith('.')) return name.endsWith(token);
    if (token.endsWith('/*')) return type !== '' && type.startsWith(token.slice(0, -1));
    return type === token;
  });
}

export function toAttachmentDisplayText(raw: unknown, emptyObjectLabel: string): string {
  if (raw == null) return '';
  const fileName = resolveFileName(raw);
  if (fileName) return fileName;
  const bindingId = resolveBindingId(raw);
  if (bindingId) return bindingId;
  if (typeof raw === 'object' && raw && !Array.isArray(raw)) {
    return emptyObjectLabel;
  }
  return String(raw);
}

export function isTableRenderMode(renderMode: unknown): boolean {
  return renderMode === 'table';
}

/** Default form avatar box edge length in CSS pixels (Tailwind size-28). */
export const DEFAULT_IMAGE_AVATAR_SIZE_PX = 112;

export type ImageAvatarFit = 'cover' | 'contain';
export type ImageAvatarRadius = 'none' | 'sm' | 'md' | 'lg' | 'xl';

function cssLengthFromProp(value: number | string | undefined): string | undefined {
  if (value === undefined || value === null) return undefined;
  if (typeof value === 'number') {
    if (!Number.isFinite(value) || value <= 0) return undefined;
    return `${value}px`;
  }
  const trimmed = String(value).trim();
  if (!trimmed) return undefined;
  if (/^\d+(\.\d+)?$/.test(trimmed)) return `${trimmed}px`;
  return trimmed;
}

/** Parse px-compatible lengths for avatar math; non-px CSS lengths fall back to default. */
export function cssLengthToPx(value: number | string | undefined, fallbackPx: number): number {
  if (typeof value === 'number') {
    if (!Number.isFinite(value) || value <= 0) return fallbackPx;
    return value;
  }
  const trimmed = String(value ?? '').trim();
  if (!trimmed) return fallbackPx;
  if (/^\d+(\.\d+)?$/.test(trimmed)) return Number(trimmed);
  const px = trimmed.match(/^(\d+(?:\.\d+)?)px$/i);
  if (px) return Number(px[1]);
  return fallbackPx;
}

/**
 * Resolve form image avatar box CSS size for cover (fixed crop box).
 * Defaults to 112×112; a single side makes a square; both sides may differ.
 */
export function resolveImageAvatarBoxStyle(
  width?: number | string,
  height?: number | string
): { width: string; height: string } {
  const w = cssLengthFromProp(width);
  const h = cssLengthFromProp(height);
  if (w && h) return { width: w, height: h };
  if (w) return { width: w, height: w };
  if (h) return { width: h, height: h };
  const fallback = `${DEFAULT_IMAGE_AVATAR_SIZE_PX}px`;
  return { width: fallback, height: fallback };
}

/**
 * Contain box: scale natural pixels into max width/height while preserving ratio.
 * When natural size is unknown, use the max box (square unless both max sides set).
 */
export function resolveImageAvatarContainBoxStyle(opts: {
  width?: number | string;
  height?: number | string;
  naturalWidth?: number;
  naturalHeight?: number;
}): { width: string; height: string } {
  const maxW = cssLengthToPx(opts.width, DEFAULT_IMAGE_AVATAR_SIZE_PX);
  const maxH =
    opts.height !== undefined && opts.height !== null && String(opts.height).trim() !== ''
      ? cssLengthToPx(opts.height, maxW)
      : opts.width !== undefined && opts.width !== null && String(opts.width).trim() !== ''
        ? maxW
        : DEFAULT_IMAGE_AVATAR_SIZE_PX;

  const nw = Number(opts.naturalWidth);
  const nh = Number(opts.naturalHeight);
  if (Number.isFinite(nw) && Number.isFinite(nh) && nw > 0 && nh > 0) {
    const scale = Math.min(maxW / nw, maxH / nh, 1);
    return {
      width: `${Math.max(1, Math.round(nw * scale))}px`,
      height: `${Math.max(1, Math.round(nh * scale))}px`,
    };
  }

  return { width: `${maxW}px`, height: `${maxH}px` };
}

export function resolveImageAvatarRadiusClass(radius?: ImageAvatarRadius | string): string {
  switch (radius) {
    case 'none':
      return 'rounded-none';
    case 'sm':
      return 'rounded-sm';
    case 'lg':
      return 'rounded-lg';
    case 'xl':
      return 'rounded-xl';
    case 'md':
    default:
      return 'rounded-md';
  }
}

/** Pending local pick waiting for form Prepare/PUT/Finalize. */
export function hasPendingLocalFile(raw: unknown): boolean {
  return isAttachmentObject(raw) && !!raw.file;
}

export type BinaryMediaIconKind =
  | 'image'
  | 'pdf'
  | 'spreadsheet'
  | 'archive'
  | 'video'
  | 'audio'
  | 'code'
  | 'file';

/** Mime/extension → icon kind for Binary field media (not ImageField behavior). */
export function resolveBinaryMediaIconKind(raw: unknown): BinaryMediaIconKind {
  const mime = (resolveMimeType(raw) || '').toLowerCase();
  const name = (resolveFileName(raw) || '').toLowerCase();

  if (mime.startsWith('image/') || /\.(png|jpe?g|gif|webp|bmp|avif|svg)$/i.test(name)) return 'image';
  if (mime === 'application/pdf' || name.endsWith('.pdf')) return 'pdf';
  if (
    mime.includes('spreadsheet') ||
    mime.includes('excel') ||
    mime === 'text/csv' ||
    /\.(xlsx?|csv|ods)$/i.test(name)
  ) {
    return 'spreadsheet';
  }
  if (
    mime.includes('zip') ||
    mime.includes('compressed') ||
    mime.includes('tar') ||
    mime.includes('gzip') ||
    /\.(zip|rar|7z|tar|gz|tgz)$/i.test(name)
  ) {
    return 'archive';
  }
  if (mime.startsWith('video/') || /\.(mp4|mov|webm|mkv|avi)$/i.test(name)) return 'video';
  if (mime.startsWith('audio/') || /\.(mp3|wav|ogg|m4a|flac)$/i.test(name)) return 'audio';
  if (
    mime.includes('json') ||
    mime.includes('javascript') ||
    mime.includes('typescript') ||
    mime.startsWith('text/') ||
    /\.(json|js|ts|tsx|jsx|vue|py|go|rs|java|c|cpp|h|md|ya?ml|xml|html?|css)$/i.test(name)
  ) {
    return 'code';
  }
  return 'file';
}
