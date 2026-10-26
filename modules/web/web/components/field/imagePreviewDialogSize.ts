// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/** Max preview box relative to the viewport (CSS px). */
export const IMAGE_PREVIEW_MAX_WIDTH_VW = 0.92;
export const IMAGE_PREVIEW_MAX_HEIGHT_VH = 0.85;
export const IMAGE_PREVIEW_MAX_WIDTH_PX = 56 * 16; // 56rem
export const IMAGE_PREVIEW_MAX_HEIGHT_PX = 48 * 16; // 48rem

/**
 * Scale intrinsic image size into the preview dialog box while preserving
 * aspect ratio and staying within viewport / rem caps.
 */
export function resolveImagePreviewDialogSize(
  naturalWidth: number,
  naturalHeight: number,
  viewportWidth: number,
  viewportHeight: number
): { width: number; height: number } {
  const nw = Number(naturalWidth);
  const nh = Number(naturalHeight);
  if (!Number.isFinite(nw) || !Number.isFinite(nh) || nw <= 0 || nh <= 0) {
    return { width: 0, height: 0 };
  }

  const maxW = Math.max(1, Math.min(viewportWidth * IMAGE_PREVIEW_MAX_WIDTH_VW, IMAGE_PREVIEW_MAX_WIDTH_PX));
  const maxH = Math.max(1, Math.min(viewportHeight * IMAGE_PREVIEW_MAX_HEIGHT_VH, IMAGE_PREVIEW_MAX_HEIGHT_PX));
  const scale = Math.min(maxW / nw, maxH / nh, 1);
  return {
    width: Math.max(1, Math.round(nw * scale)),
    height: Math.max(1, Math.round(nh * scale)),
  };
}
