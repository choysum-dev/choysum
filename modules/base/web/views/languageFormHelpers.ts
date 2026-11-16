// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

export const GROUPING_SELECTION = ['[3,0]', '[3,2,0]', '[3,3]', '[]'] as const;

/** Language.FirstDayOfWeek selection values (0=Sunday … 6=Saturday); runtime Number() for calendars. */
export const FIRST_DAY_SELECTION = ['0', '1', '2', '3', '4', '5', '6'] as const;

/** List columns for LanguageListView. */
export const LANGUAGE_LIST_COLUMN_PROPS = ['Name', 'Code', 'Direction', 'IsActive', 'DateFormat'] as const;

/** Code is editable only while creating a Language row. */
export function isLanguageCodeReadonly(mode: string | null | undefined): boolean {
  return String(mode || '') !== 'create';
}
