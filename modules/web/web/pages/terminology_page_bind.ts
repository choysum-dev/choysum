// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/** Resolve the TranslationTerm host application for a list route. */
export function resolveTerminologyListApplication(route: {
  meta?: Record<string, unknown> | null;
  params?: Record<string, unknown> | null;
  path?: string;
}): string {
  const fromMeta = String((route.meta as { terminologyApplication?: string } | null)?.terminologyApplication || '').trim();
  if (fromMeta) return fromMeta;
  const fromParam = String(route.params?.application || '').trim();
  if (fromParam) return fromParam;
  const m = String(route.path || '').match(/\/web\/terminology\/([^/]+)/);
  return String(m?.[1] || '').trim();
}

/** Resolve the TranslationTerm host application for a detail route. */
export function resolveTerminologyDetailApplication(route: {
  params?: Record<string, unknown> | null;
}): string {
  return String(route.params?.application || '').trim();
}
