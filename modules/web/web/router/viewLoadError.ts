// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { computed, ref } from 'vue';

export type ViewLoadErrorState = {
  message: string;
  /** fullPath to retry via router.replace */
  fullPath: string;
};

const current = ref<ViewLoadErrorState | null>(null);

export const viewLoadError = computed(() => current.value);

export function setViewLoadError(next: ViewLoadErrorState | null): void {
  current.value = next;
}

export function clearViewLoadError(): void {
  current.value = null;
}

/**
 * True for dynamic-import / chunk load failures (browser wording varies).
 */
export function isRouteChunkLoadError(error: unknown): boolean {
  const message = String(
    (error as { message?: string })?.message || error || ''
  ).toLowerCase();
  if (!message) return false;
  return (
    message.includes('failed to fetch dynamically imported module') ||
    message.includes('loading chunk') ||
    message.includes('loading css chunk') ||
    message.includes('error loading dynamically imported module') ||
    message.includes('importing a module script failed') ||
    (message.includes('failed to fetch') && message.includes('.js')) ||
    message.includes('err_connection_refused')
  );
}
