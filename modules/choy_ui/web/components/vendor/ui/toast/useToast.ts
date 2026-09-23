// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ref, type Ref } from 'vue';

export type ToastRecord = {
  id: number;
  title: string;
  description?: string;
  open: boolean;
  duration: number;
};

let nextId = 0;

const toasts: Ref<ToastRecord[]> = ref([]);

export type ToastInput = {
  title: string;
  description?: string;
  duration?: number;
};

const defaultToastDurationMs = 5000;

/**
 * Shows a transient toast. Duration is owned by ToastRoot (Reka default 5000ms);
 * dismiss/removal still runs from the Toaster open update handler.
 */
export function toast(input: ToastInput): number {
  const id = ++nextId;
  const record: ToastRecord = {
    id,
    title: input.title,
    description: input.description,
    open: true,
    duration: input.duration ?? defaultToastDurationMs,
  };
  toasts.value = [...toasts.value, record];
  return id;
}

/**
 * Dismisses a toast by id.
 */
export function dismiss(id: number): void {
  toasts.value = toasts.value.map((item) => (item.id === id ? { ...item, open: false } : item));
  setTimeout(() => {
    toasts.value = toasts.value.filter((item) => item.id !== id);
  }, 200);
}

/**
 * Reactive toast list for the gallery Toaster host.
 */
export function useToastStore(): Ref<ToastRecord[]> {
  return toasts;
}
