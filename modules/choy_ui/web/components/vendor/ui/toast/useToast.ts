// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ref } from 'vue';

export type ToastRecord = {
  id: number;
  title: string;
  description?: string;
  open: boolean;
  /** Reka ToastRoot duration; Infinity means stay until dismissed. */
  duration: number;
};

let nextId = 0;

const toasts = ref<ToastRecord[]>([]);

export type ToastInput = {
  title: string;
  description?: string;
  /** Milliseconds; 0 means stay until dismissed. Defaults to 5000. */
  duration?: number;
};

const defaultToastDurationMs = 5000;
/** Delay before removing a dismissed toast (close animation). */
const removalDelayMs = 200;
const removalTimers = new Map<number, ReturnType<typeof setTimeout>>();

/**
 * Resolves toast duration for ToastRoot.
 * 0 is the common "stay until dismissed" convention (maps to Infinity).
 */
function resolveToastDuration(duration: number | undefined): number {
  if (duration === undefined) {
    return defaultToastDurationMs;
  }
  if (duration <= 0) {
    return Number.POSITIVE_INFINITY;
  }
  return duration;
}

/**
 * Shows a transient toast. Duration is owned by ToastRoot; dismiss/removal
 * still runs from the Toaster open update handler.
 */
export function toast(input: ToastInput): number {
  const id = ++nextId;
  const record: ToastRecord = {
    id,
    title: input.title,
    description: input.description,
    open: true,
    duration: resolveToastDuration(input.duration),
  };
  toasts.value = [...toasts.value, record];
  return id;
}

/**
 * Dismisses a toast by id.
 */
export function dismiss(id: number): void {
  toasts.value = toasts.value.map((item) => (item.id === id ? { ...item, open: false } : item));
  const pending = removalTimers.get(id);
  if (pending !== undefined) {
    clearTimeout(pending);
  }
  removalTimers.set(
    id,
    setTimeout(() => {
      removalTimers.delete(id);
      toasts.value = toasts.value.filter((item) => item.id !== id);
    }, removalDelayMs),
  );
}

/**
 * Drops every pending toast. Call from the Toaster host's onUnmounted: the
 * store is module-level and outlives the component, so an open record would
 * otherwise reappear when the host mounts again.
 */
export function clearToasts(): void {
  for (const timer of removalTimers.values()) {
    clearTimeout(timer);
  }
  removalTimers.clear();
  toasts.value = [];
}

/**
 * Reactive toast list for the gallery Toaster host.
 */
export function useToastStore() {
  return toasts;
}
