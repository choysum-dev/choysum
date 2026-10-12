// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { useRouter, useRoute } from 'vue-router';
import { useBreadcrumbStore } from '../stores/breadcrumbStore';
import type { BreadcrumbItem } from '../stores/breadcrumbStore';
import { useMenu } from './useMenu';

/**
 * Provides computed breadcrumb data and navigation helpers.
 */
export function useBreadcrumb() {
  const router = useRouter();
  const route = useRoute();
  const breadcrumbStore = useBreadcrumbStore();
  let menuApi: ReturnType<typeof useMenu> | null = null;
  try {
    menuApi = useMenu();
  } catch {
    menuApi = null;
  }

  /**
   * Navigates to a breadcrumb target when the item is clickable.
   * App crumbs use navigateTo(app) (first navigable leaf); others use path.
   */
  async function navigateTo(crumb: BreadcrumbItem, index: number): Promise<void> {
    if (!crumb.clickable) return;

    try {
      if (crumb.kind === 'app' && crumb.menuId && menuApi) {
        await menuApi.navigateTo(crumb.menuId);
        return;
      }

      const targetPath = breadcrumbStore.navigateToBreadcrumb(index);
      if (targetPath && targetPath !== route.path) {
        await router.push(targetPath);
      }
    } catch (error) {
      console.error('Breadcrumb navigation failed:', error);
    }
  }

  return {
    breadcrumbs: breadcrumbStore.breadcrumbStack,
    navigateTo,
  };
}

export type { BreadcrumbItem };
