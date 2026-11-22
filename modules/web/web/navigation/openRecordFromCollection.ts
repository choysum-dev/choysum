// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { useRecordSetStore, type RecordSetEntry } from '@/web/web/stores/recordSetStore';
import type { DetailPathTarget } from './detailPath';

type RouterLike = {
  push: (to: string | Record<string, unknown>) => unknown;
};

export type OpenRecordFromCollectionArgs = {
  router: RouterLike;
  to: DetailPathTarget;
  entry: RecordSetEntry;
};

/**
 * Seed the Form pager session and navigate to the detail target.
 */
export function openRecordFromCollection(args: OpenRecordFromCollectionArgs): void {
  const store = useRecordSetStore();
  store.seed(args.entry);
  store.installMenuClearWatch();
  const to = args.to;
  const push =
    typeof to === 'string'
      ? args.router.push(to)
      : args.router.push(to as Record<string, unknown>);
  // Aborted / failed navigations must not leave a seeded pager session behind.
  Promise.resolve(push).catch(() => {
    try {
      useRecordSetStore().clear();
    } catch {
      // Pinia may be unavailable in unit mounts without an app shell.
    }
  });
}
