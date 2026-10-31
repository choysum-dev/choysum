// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/** Resolve the PropertyDefinition host application for a list route. */
export function resolvePropertyDefinitionListApplication(route: {
  meta?: Record<string, unknown> | null;
  params?: Record<string, unknown> | null;
  path?: string;
}): string {
  const fromMeta = String(
    (route.meta as { propertyDefinitionApplication?: string } | null)?.propertyDefinitionApplication || ''
  ).trim();
  if (fromMeta) return fromMeta;
  const fromParam = String(route.params?.application || '').trim();
  if (fromParam) return fromParam;
  const m = String(route.path || '').match(/\/web\/property-definitions\/([^/]+)/);
  return String(m?.[1] || '').trim();
}

/** Resolve the PropertyDefinition host application for create/detail routes. */
export function resolvePropertyDefinitionDetailApplication(route: {
  params?: Record<string, unknown> | null;
}): string {
  return String(route.params?.application || '').trim();
}
