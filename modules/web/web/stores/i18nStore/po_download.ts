// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Download a module-scoped PO export from the host gateway.
 * Requires lang, application, and module (P4).
 */
export async function downloadTerminologyPo(opts: {
  lang: string;
  application: string;
  module: string;
  accessToken?: string | null;
  fetchImpl?: typeof fetch;
}): Promise<Blob> {
  const lang = opts.lang.trim();
  const application = opts.application.trim();
  const moduleName = opts.module.trim();
  if (!lang) throw new Error('lang is required');
  if (!application) throw new Error('application is required');
  if (!moduleName) throw new Error('module is required');

  const params = new URLSearchParams({
    lang,
    application,
    module: moduleName,
  });
  const headers: Record<string, string> = {};
  const token = (opts.accessToken || '').trim();
  if (token) {
    headers.Authorization = `Bearer ${token}`;
  }
  const fetchFn = opts.fetchImpl || fetch;
  const res = await fetchFn(`/web/i18n/po?${params.toString()}`, { headers });
  if (!res.ok) {
    let detail = res.statusText;
    try {
      const body = await res.json();
      if (body?.error) detail = String(body.error);
    } catch {
      /* ignore */
    }
    throw new Error(detail || `PO download failed (${res.status})`);
  }
  return res.blob();
}
