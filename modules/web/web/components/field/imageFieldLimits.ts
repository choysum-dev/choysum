// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { DEFAULT_GLOBAL_MAX_UPLOAD_BYTES } from '@/core/service/orm/upload_limits';

export type ImageFieldLimits = {
  maxUploadBytes?: number;
  maxWidth?: number;
  maxHeight?: number;
};

export type ImageFieldLimitMessages = {
  fileTooLarge: (formattedLimit: string) => string;
  widthTooLarge: (maxWidth: number) => string;
  heightTooLarge: (maxHeight: number) => string;
};

export function resolveImageFieldLimits(meta: ImageFieldLimits | null | undefined): ImageFieldLimits {
  const out: ImageFieldLimits = {};
  if (typeof meta?.maxUploadBytes === 'number' && meta.maxUploadBytes > 0) {
    out.maxUploadBytes = Math.min(meta.maxUploadBytes, DEFAULT_GLOBAL_MAX_UPLOAD_BYTES);
  }
  if (typeof meta?.maxWidth === 'number' && meta.maxWidth > 0) {
    out.maxWidth = meta.maxWidth;
  }
  if (typeof meta?.maxHeight === 'number' && meta.maxHeight > 0) {
    out.maxHeight = meta.maxHeight;
  }
  return out;
}

export function formatImageByteLimit(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${Math.round(bytes / 1024)} KB`;
  return `${Math.round(bytes / (1024 * 1024))} MB`;
}

export async function readImageNaturalDimensions(file: Blob): Promise<{ width: number; height: number } | undefined> {
  if (typeof createImageBitmap === 'function') {
    try {
      const bitmap = await createImageBitmap(file);
      const width = bitmap.width;
      const height = bitmap.height;
      if (typeof bitmap.close === 'function') {
        bitmap.close();
      }
      return { width, height };
    } catch {
      return undefined;
    }
  }

  if (typeof Image === 'undefined') {
    return undefined;
  }

  const objectUrl = typeof URL !== 'undefined' && typeof URL.createObjectURL === 'function' ? URL.createObjectURL(file) : undefined;
  if (!objectUrl) {
    return undefined;
  }

  try {
    return await new Promise(resolve => {
      const img = new Image();
      img.onload = () => {
        resolve({ width: img.naturalWidth, height: img.naturalHeight });
      };
      img.onerror = () => resolve(undefined);
      img.src = objectUrl;
    });
  } finally {
    if (typeof URL.revokeObjectURL === 'function') {
      URL.revokeObjectURL(objectUrl);
    }
  }
}

export type ImageFieldValidationResult =
  | { ok: true }
  | { ok: false; reason: 'fileTooLarge' | 'widthTooLarge' | 'heightTooLarge'; detail: string };

export async function validateImageFieldFile(
  file: Blob,
  limitsInput: ImageFieldLimits | null | undefined,
  readDimensions: (file: Blob) => Promise<{ width: number; height: number } | undefined> = readImageNaturalDimensions
): Promise<ImageFieldValidationResult> {
  const limits = resolveImageFieldLimits(limitsInput);
  if (limits.maxUploadBytes !== undefined && file.size > limits.maxUploadBytes) {
    return { ok: false, reason: 'fileTooLarge', detail: formatImageByteLimit(limits.maxUploadBytes) };
  }

  if (limits.maxWidth === undefined && limits.maxHeight === undefined) {
    return { ok: true };
  }

  const dimensions = await readDimensions(file);
  if (!dimensions) {
    return { ok: true };
  }

  if (limits.maxWidth !== undefined && dimensions.width > limits.maxWidth) {
    return { ok: false, reason: 'widthTooLarge', detail: String(limits.maxWidth) };
  }
  if (limits.maxHeight !== undefined && dimensions.height > limits.maxHeight) {
    return { ok: false, reason: 'heightTooLarge', detail: String(limits.maxHeight) };
  }

  return { ok: true };
}
