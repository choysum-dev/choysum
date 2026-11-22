// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { useRecordSetStore } from '@/web/web/stores/recordSetStore';

type RouterLike = {
  push: (to: string | Record<string, unknown>) => unknown;
};

export type OpenRelatedRecordArgs = {
  router: RouterLike;
  to: string | Record<string, unknown>;
};

/**
 * Open a related record (e.g. M2O display click): leave the collection pager and push.
 * Breadcrumb defaults to push under the same menu leaf so the source form stays backable.
 */
export function openRelatedRecord(args: OpenRelatedRecordArgs): void {
  useRecordSetStore().clear();
  const to = args.to;
  if (typeof to === 'string') {
    void args.router.push(to);
    return;
  }
  void args.router.push(to);
}
