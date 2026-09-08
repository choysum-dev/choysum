// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { RouteLocationNormalized } from 'vue-router';
import { translateTerm, type ComposerLike } from '../i18n';
import { isTermReference } from '@/core/service/i18n';

type RouteTitleMeta = {
  pageTitle?: unknown;
  pageTitleText?: unknown;
};

/** Resolve the browser document title for a route (term reference or string fallback). */
export function resolveDocumentTitle(
  route: { meta?: RouteTitleMeta } | null | undefined,
  composer?: ComposerLike,
  appName = 'Choysum'
): string {
  const pageTitle = route?.meta?.pageTitle;
  let fallback = '';
  if (typeof pageTitle === 'function') {
    try {
      fallback = String(pageTitle(route as RouteLocationNormalized) ?? '');
    } catch {
      fallback = '';
    }
  } else if (typeof pageTitle === 'string') {
    fallback = pageTitle;
  }
  const reference = isTermReference(route?.meta?.pageTitleText) ? route!.meta!.pageTitleText : undefined;
  const title = translateTerm(composer, reference, fallback);
  return title ? `${title} - ${appName}` : appName;
}
