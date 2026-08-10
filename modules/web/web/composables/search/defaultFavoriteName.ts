// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { isTermReference, type TermReference } from '@/core/service/i18n';

export type DefaultFavoriteNameSources = {
  /** Tip of the breadcrumb trail (stable src / plain title). */
  breadcrumbTip?: string | null;
  /** Route meta pageTitle / pageTitleText.src (not live $t). */
  routeTitle?: string | null;
  /** Active menu title src (Odoo action/display name analogue). */
  menuTitle?: string | null;
  /** Last-resort identity: store.application + store.modelName. */
  modelIdentity?: string | null;
};

/**
 * Pick the first non-empty stable default for a new SavedFilter name.
 * Prefer human titles' canonical src over route path/name; never use live i18n.
 */
export function pickDefaultFavoriteName(sources: DefaultFavoriteNameSources): string {
  for (const raw of [sources.breadcrumbTip, sources.routeTitle, sources.menuTitle, sources.modelIdentity]) {
    const s = String(raw ?? '').trim();
    if (s) return s;
  }
  return '';
}

/**
 * Canonical label for SavedFilter Name defaults: TermReference.src (or plain title).
 * Do not call translateTerm / $t — Name must stay language-stable across UI locale switches.
 */
export function stableTitleSource(title?: string, titleText?: TermReference): string {
  if (titleText && isTermReference(titleText)) {
    return String(titleText.src || title || '').trim();
  }
  return String(title || '').trim();
}

/** Resolve a language-stable title from a vue-router location (meta src / plain strings only). */
export function routeTitleFromLocation(route: any): string {
  if (!route) return '';
  const meta = route.meta || {};
  if (isTermReference(meta.pageTitleText)) {
    return stableTitleSource(undefined, meta.pageTitleText);
  }
  const pageTitle = meta.pageTitle;
  if (typeof pageTitle === 'function') {
    try {
      return String(pageTitle(route) || '').trim();
    } catch {
      return '';
    }
  }
  if (typeof pageTitle === 'string' && pageTitle.trim()) return pageTitle.trim();
  if (meta.title != null && String(meta.title).trim()) return String(meta.title).trim();
  return '';
}

/** Build `application.ModelName` identity for last-resort default names. */
export function modelIdentityFromStore(store: { application?: unknown; modelName?: unknown } | null | undefined): string {
  if (!store) return '';
  const app = String(store.application ?? '').trim();
  const model = String(store.modelName ?? '').trim();
  if (app && model) return `${app}.${model}`;
  return app || model;
}
