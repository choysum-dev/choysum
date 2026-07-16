// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { TerminologyLoadResult } from './terminology_loader';

/**
 * Whether vue-i18n should mergeLocaleMessage for a Gateway load result (D4).
 * unchanged / gatewayError / null messages → do not merge empty objects.
 */
export function shouldMergeTerminology(load: TerminologyLoadResult | null | undefined): boolean {
  if (!load || load.unchanged || load.gatewayError) {
    return false;
  }
  return load.messages != null;
}
