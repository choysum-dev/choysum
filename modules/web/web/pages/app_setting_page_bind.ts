// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/** Resolve the AppSetting host application for a list route. */
export function resolveAppSettingListApplication(route: {
  meta?: Record<string, unknown> | null;
  params?: Record<string, unknown> | null;
  path?: string;
}): string {
  const fromMeta = String(
    (route.meta as { appSettingApplication?: string } | null)?.appSettingApplication || ''
  ).trim();
  if (fromMeta) return fromMeta;
  const fromParam = String(route.params?.application || '').trim();
  if (fromParam) return fromParam;
  const m = String(route.path || '').match(/\/web\/app-settings\/([^/]+)/);
  return String(m?.[1] || '').trim();
}

/** Resolve the AppSetting host application for create/detail routes. */
export function resolveAppSettingDetailApplication(route: {
  params?: Record<string, unknown> | null;
}): string {
  return String(route.params?.application || '').trim();
}
