// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { isValidPersonNameFormat } from '@/core/service/utils/person_name';
import { useI18nStore } from '@/web/web/stores/i18nStore';

export const GROUPING_SELECTION = ['[3,0]', '[3,2,0]', '[3,3]', '[]'] as const;

/** Language.FirstDayOfWeek selection values (0=Sunday … 6=Saturday); runtime Number() for calendars. */
export const FIRST_DAY_SELECTION = ['0', '1', '2', '3', '4', '5', '6'] as const;

/** List columns for LanguageListView. */
export const LANGUAGE_LIST_COLUMN_PROPS = ['Name', 'Code', 'Direction', 'IsActive', 'DateFormat'] as const;

/**
 * Code is editable only while creating a Language row.
 * Copy uses beginCreate (viewMode becomes create), so Code stays editable there.
 */
export function isLanguageCodeReadonly(mode: string | null | undefined): boolean {
  return String(mode || '') !== 'create';
}

type RuleCallback = (error?: Error) => void;

/** Form rules for Language.PersonNameFormat (template must mention FirstName and/or LastName). */
export function createPersonNameFormatRules(invalidMessage: string) {
  return [
    {
      validator: (_rule: unknown, value: unknown, cb: RuleCallback) => {
        if (!isValidPersonNameFormat(value)) {
          return cb(new Error(invalidMessage));
        }
        cb();
      },
      trigger: ['change', 'blur'] as const,
    },
  ];
}

/** Best-effort refresh of FE language format overlays after Language create/update. */
export async function reloadActiveLanguagesAfterLanguageSave(): Promise<void> {
  try {
    await useI18nStore().loadActiveLanguages();
  } catch {
    // Next full page init will reload formats.
  }
}
