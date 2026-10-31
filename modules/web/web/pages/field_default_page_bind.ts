// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/** Resolve the FieldDefault host application for a list route. */
export function resolveFieldDefaultListApplication(route: {
  meta?: Record<string, unknown> | null;
  params?: Record<string, unknown> | null;
  path?: string;
}): string {
  const fromMeta = String(
    (route.meta as { fieldDefaultApplication?: string } | null)?.fieldDefaultApplication || ''
  ).trim();
  if (fromMeta) return fromMeta;
  const fromParam = String(route.params?.application || '').trim();
  if (fromParam) return fromParam;
  const m = String(route.path || '').match(/\/web\/field-defaults\/([^/]+)/);
  return String(m?.[1] || '').trim();
}

/** Resolve the FieldDefault host application for create/detail routes. */
export function resolveFieldDefaultDetailApplication(route: {
  params?: Record<string, unknown> | null;
}): string {
  return String(route.params?.application || '').trim();
}
