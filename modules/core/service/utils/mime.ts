// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Check whether a given MIME type supports inline browser rendering.
 */
export function inlineMimeAllowed(mimeType: string): boolean {
  const normalized = mimeType.toLowerCase();
  return (
    normalized === 'image/png' || normalized === 'image/jpeg' || normalized === 'image/webp' || normalized === 'application/pdf' || normalized === 'text/plain'
  );
}

/**
 * Map a MIME type to a canonical file extension (including leading dot).
 */
export function mimeSuffix(mimeType: string): string {
  const normalized = mimeType.toLowerCase();
  switch (normalized) {
    case 'image/png':
      return '.png';
    case 'image/jpeg':
      return '.jpg';
    case 'image/webp':
      return '.webp';
    case 'application/pdf':
      return '.pdf';
    case 'text/plain':
      return '.txt';
    default:
      return '';
  }
}

/**
 * Check whether a content type is present in an allow-list.
 *
 * Returns true when the allow-list is empty (no restrictions).
 */
export function isMimeTypeAllowed(contentType: string | undefined, allowedMimeTypes: string[]): boolean {
  if (allowedMimeTypes.length === 0) {
    return true;
  }
  if (!contentType) {
    return false;
  }
  return allowedMimeTypes.includes(contentType);
}
