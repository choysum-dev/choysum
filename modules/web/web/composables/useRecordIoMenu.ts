// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { computed, toValue, type ComputedRef, type Ref } from 'vue';
import { createTranslate } from '@/web/web/i18n';
import type { PageIoMenuItem, RecordIoConfig } from './recordIoTypes';

export type UseRecordIoMenuOptions = {
  config: RecordIoConfig | Ref<RecordIoConfig> | ComputedRef<RecordIoConfig>;
  openImport?: () => void;
  openExport?: () => void;
  importLabel?: string;
  exportLabel?: string;
};

/**
 * Builds title-adjacent Import/Export menu items from RecordIoConfig.
 */
export function useRecordIoMenu(options: UseRecordIoMenuOptions) {
  const { _t } = createTranslate('web', { scope: 'web/composables/useRecordIoMenu' });

  const items = computed((): PageIoMenuItem[] => {
    const config = toValue(options.config);
    const result: PageIoMenuItem[] = [];
    if (config.import?.enabled && options.openImport) {
      result.push({
        key: 'import',
        label: options.importLabel ?? _t('Import'),
        onClick: options.openImport,
      });
    }
    if (config.export?.enabled && options.openExport) {
      result.push({
        key: 'export',
        label: options.exportLabel ?? _t('Export'),
        onClick: options.openExport,
      });
    }
    return result;
  });

  const visible = computed(() => items.value.length > 0);

  return { items, visible };
}
