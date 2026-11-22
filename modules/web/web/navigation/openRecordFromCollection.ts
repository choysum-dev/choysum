// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { isNavigationFailure } from 'vue-router';
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

function clearSeededSession(generation: number): void {
  try {
    const rs = useRecordSetStore();
    if (rs.active && rs.seedGeneration === generation) rs.clear();
  } catch {
    // Pinia may be unavailable in unit mounts without an app shell.
  }
}

/**
 * Seed the Form pager session and navigate to the detail target.
 */
export function openRecordFromCollection(args: OpenRecordFromCollectionArgs): void {
  let generation = -1;
  try {
    const store = useRecordSetStore();
    store.seed(args.entry);
    store.installMenuClearWatch();
    // Only clear this open on failure — a later open may have already re-seeded.
    generation = store.seedGeneration;
  } catch {
    // Pinia may be unavailable; still navigate without a pager session.
  }
  const to = args.to;
  let push: unknown;
  try {
    push =
      typeof to === 'string'
        ? args.router.push(to)
        : args.router.push(to as Record<string, unknown>);
  } catch {
    if (generation >= 0) clearSeededSession(generation);
    return;
  }
  // Aborted / failed navigations must not leave a seeded pager session behind.
  // Vue Router resolves guard aborts as NavigationFailure (fulfilled, not rejected).
  Promise.resolve(push).then(
    result => {
      if (generation >= 0 && isNavigationFailure(result)) clearSeededSession(generation);
    },
    () => {
      if (generation >= 0) clearSeededSession(generation);
    }
  );
}
