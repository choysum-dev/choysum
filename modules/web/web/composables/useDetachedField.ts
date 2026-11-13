// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ref } from 'vue';
import type { WebFieldMetadata } from '@/web/web/stores/modelStore';
import { useStandaloneField, type UseField } from './useField';

export type UseDetachedFieldOptions = {
  /** Synthetic prop name for FieldBase ids / error maps (default `_slot`). */
  prop?: string;
  /** Optional static meta overlay (rarely needed for chrome-only fields). */
  meta?: Partial<WebFieldMetadata>;
};

/**
 * Detached UseField for chrome-only surfaces (e.g. ChoySlotField).
 * No store, no registration, no Write — FieldBase still receives a real binding.
 * Forces form + edit env so the edit slot renders outside a FormView scope.
 */
export function useDetachedField(opts?: UseDetachedFieldOptions): UseField<any, any> {
  const value = ref<unknown>(null);
  return useStandaloneField({
    value,
    prop: opts?.prop ?? '_slot',
    meta: opts?.meta,
    env: {
      isForm: true,
      fieldPresentation: 'form',
      isEditMode: true,
      viewMode: 'edit',
      fieldPrefix: null,
    },
  });
}
