// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Host Gateway terminology catalog API (GET|PATCH /web/i18n/terms).
 */

import { getTokenProvider } from '@/core/web/rpc/providers';

export type TermItem = {
  application: string;
  module: string;
  scope: string;
  src: string;
  value: string;
  kind: string;
  source?: string;
  status?: string;
};

export type TermsListResponse = {
  lang: string;
  items: TermItem[];
  total: number;
  limit: number;
  offset: number;
  truncated?: boolean;
};

export type TermsListQuery = {
  lang: string;
  application?: string;
  module?: string;
  q?: string;
  limit?: number;
  offset?: number;
};

export type TermsPatchBody =
  | (TermItem & { lang: string })
  | {
      lang: string;
      items: TermItem[];
    };

export type FetchTermsOptions = {
  fetchImpl?: typeof fetch;
  baseUrl?: string;
  /** Override Authorization bearer (tests). */
  accessToken?: string;
};

async function resolveAccessToken(override?: string): Promise<string> {
  const explicit = String(override || '').trim();
  if (explicit) {
    return explicit;
  }
  const tokenProvider = getTokenProvider();
  if (!tokenProvider) {
    return '';
  }
  try {
    const needRefresh = await tokenProvider.shouldRefreshToken?.();
    if (needRefresh) {
      await tokenProvider.refreshToken();
    }
  } catch {
    // best-effort
  }
  try {
    return String((await tokenProvider.getToken()) || '').trim();
  } catch {
    return '';
  }
}

async function authorizedFetch(
  url: string,
  init: RequestInit,
  options?: FetchTermsOptions
): Promise<Response> {
  const fetchFn = options?.fetchImpl ?? globalThis.fetch;
  if (typeof fetchFn !== 'function') {
    throw new Error('fetch is not available');
  }
  const headers = new Headers(init.headers || {});
  if (!headers.has('Accept')) {
    headers.set('Accept', 'application/json');
  }
  if (!headers.has('Authorization')) {
    const token = await resolveAccessToken(options?.accessToken);
    if (token) {
      headers.set('Authorization', `Bearer ${token}`);
    }
  }
  return fetchFn(url, {
    ...init,
    headers,
    credentials: init.credentials ?? 'same-origin',
  });
}

/**
 * List terminology rows via Gateway (authenticated).
 */
export async function fetchTerms(query: TermsListQuery, options?: FetchTermsOptions): Promise<TermsListResponse> {
  const lang = String(query.lang || '').trim();
  if (!lang) {
    throw new Error('lang is required');
  }
  const params = new URLSearchParams({ lang });
  const application = String(query.application || '').trim();
  if (application) {
    params.set('application', application);
  }
  const moduleName = String(query.module || '').trim();
  if (moduleName) {
    params.set('module', moduleName);
  }
  const q = String(query.q || '').trim();
  if (q) {
    params.set('q', q);
  }
  if (query.limit != null) {
    params.set('limit', String(query.limit));
  }
  if (query.offset != null) {
    params.set('offset', String(query.offset));
  }

  const base = String(options?.baseUrl ?? '').replace(/\/$/, '');
  const url = `${base}/web/i18n/terms?${params.toString()}`;
  const res = await authorizedFetch(url, { method: 'GET' }, options);
  if (!res.ok) {
    throw new Error(`terms gateway HTTP ${res.status}`);
  }
  const body = (await res.json()) as Partial<TermsListResponse>;
  return {
    lang: String(body.lang || lang),
    items: Array.isArray(body.items) ? (body.items as TermItem[]) : [],
    total: Number(body.total || 0),
    limit: Number(body.limit || 0),
    offset: Number(body.offset || 0),
    truncated: body.truncated === true,
  };
}

/**
 * Patch terminology rows via Gateway (authenticated).
 */
export async function patchTerms(body: TermsPatchBody, options?: FetchTermsOptions): Promise<TermsListResponse> {
  const lang = String((body as any).lang || '').trim();
  if (!lang) {
    throw new Error('lang is required');
  }
  const base = String(options?.baseUrl ?? '').replace(/\/$/, '');
  const url = `${base}/web/i18n/terms`;
  const res = await authorizedFetch(
    url,
    {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    },
    options
  );
  if (!res.ok) {
    throw new Error(`terms gateway HTTP ${res.status}`);
  }
  const payload = (await res.json()) as Partial<TermsListResponse>;
  return {
    lang: String(payload.lang || lang),
    items: Array.isArray(payload.items) ? (payload.items as TermItem[]) : [],
    total: Number(payload.total || (payload.items as any[])?.length || 0),
    limit: Number(payload.limit || 0),
    offset: Number(payload.offset || 0),
  };
}
