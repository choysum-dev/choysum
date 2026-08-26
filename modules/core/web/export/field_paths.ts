// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Normalizes list-registry field paths (dot-separated relation segments) to record export paths (slash-separated).
 */
export function normalizeExportFieldPath(path: string): string {
  const trimmed = String(path ?? '').trim();
  if (!trimmed) {
    return '';
  }
  return trimmed.replace(/\./g, '/');
}

/** Normalizes and deduplicates export field paths while preserving order. */
export function normalizeExportFieldPaths(paths: string[] | undefined | null): string[] {
  const out: string[] = [];
  const seen = new Set<string>();
  for (const path of paths ?? []) {
    const normalized = normalizeExportFieldPath(path);
    if (!normalized || normalized === 'Id' || seen.has(normalized)) {
      continue;
    }
    seen.add(normalized);
    out.push(normalized);
  }
  return out;
}
