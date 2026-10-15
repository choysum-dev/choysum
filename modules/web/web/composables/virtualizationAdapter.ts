// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ref } from 'vue';
import { CHOY_LIST_ROW_HEIGHT } from './listDensity';

export type VirtualizationConfig = {
  enabled: boolean;
  rowHeight?: number;
  overscan?: number;
};

export function useVirtualizationAdapter(initial?: Partial<VirtualizationConfig>) {
  const config = ref<VirtualizationConfig>({
    enabled: true,
    rowHeight: CHOY_LIST_ROW_HEIGHT,
    overscan: 5,
    ...(initial || {}),
  });
  // Placeholder hooks for future integration
  function applyTo<T extends Record<string, any>>(opts: T): T {
    return opts;
  }
  return { config, applyTo };
}
