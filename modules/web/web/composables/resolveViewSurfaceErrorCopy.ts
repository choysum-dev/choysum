// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { getActivePinia } from 'pinia';
import { isTransportOutage } from '@/core/web/rpc/transportOutage';
import { createTranslate } from '@/web/web/i18n';
import { useConnectivityStore } from '../stores/connectivityStore';

export type ViewSurfaceErrorCopy = {
  title: string;
  detail: string;
};

export type ResolveViewSurfaceErrorCopyOpts = {
  /** Used when the failure is not a session/transport outage. */
  fallbackTitle?: string;
  /** Override detail for non-outage failures (default: Error.message). */
  fallbackDetail?: string;
};

const { _t } = createTranslate('web', {
  path: 'web/composables/resolveViewSurfaceErrorCopy',
  location: 'resolveViewSurfaceErrorCopy',
});

function isSessionUnreachable(): boolean {
  try {
    if (!getActivePinia()) return false;
    return useConnectivityStore().status === 'unreachable';
  } catch {
    return false;
  }
}

function errorDetail(error: unknown, fallback?: string): string {
  if (typeof fallback === 'string' && fallback.trim()) return fallback.trim();
  if (error instanceof Error) {
    const msg = String(error.message || '').trim();
    if (msg) return msg.length > 240 ? `${msg.slice(0, 237)}…` : msg;
  }
  const raw = String(error ?? '').trim();
  if (raw) return raw.length > 240 ? `${raw.slice(0, 237)}…` : raw;
  return '';
}

/**
 * Action-layer copy for surface load failures (list/kanban/chart/form).
 * Title may match Session/Asset ("Connection lost"); detail is Action-specific so
 * it is not confused with Asset chunk blank ("Cannot reach the server…").
 */
export function resolveViewSurfaceErrorCopy(
  error: unknown,
  opts: ResolveViewSurfaceErrorCopyOpts = {},
): ViewSurfaceErrorCopy {
  const outage = isSessionUnreachable() || isTransportOutage(error);
  if (outage) {
    return {
      title: _t('Connection lost'),
      detail: _t('Could not complete this request. Check your network, then retry.'),
    };
  }
  const title = (opts.fallbackTitle || '').trim() || _t('Failed to load');
  const detail =
    errorDetail(error, opts.fallbackDetail) || _t('Something went wrong. Retry to try again.');
  return { title, detail };
}

/**
 * @deprecated Soft-fail surfaces use the center BusyHost overlay; keep for tests only.
 */
export function resolveViewSurfaceStaleBannerCopy(): string {
  return _t('Could not refresh.');
}
