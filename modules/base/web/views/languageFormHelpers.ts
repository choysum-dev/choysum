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

/** Unwrap Form Browse RecordRow `{ key, payload }` / `{ raw }` or a flat field map. */
export function languageRowFields(row: unknown): Record<string, unknown> | null {
  if (!row || typeof row !== 'object') return null;
  const r = row as Record<string, unknown>;
  if (r.payload && typeof r.payload === 'object') return r.payload as Record<string, unknown>;
  if (r.raw && typeof r.raw === 'object') return r.raw as Record<string, unknown>;
  return r;
}

/** True when a Form/List row (RecordRow or flat) matches the Language id. */
export function languageRowMatchesId(row: unknown, id: string): boolean {
  if (!id || !row || typeof row !== 'object') return false;
  const r = row as Record<string, unknown>;
  if (String(r.key || '').trim() === id) return true;
  const fields = languageRowFields(row);
  return !!fields && String(fields.Id || '').trim() === id;
}

/** Resolve the current Language field map from a form store snapshot. */
export function resolveLanguageCurrentRow(opts: {
  recordId?: string | null;
  records?: Record<string, unknown> | null;
  rows?: unknown[] | null;
}): Record<string, unknown> {
  const id = String(opts.recordId || '').trim();
  const records = (opts.records || {}) as Record<string, unknown>;
  if (id && records[id] != null) {
    return languageRowFields(records[id]) || {};
  }
  const rows = (opts.rows || []) as unknown[];
  const matched = id ? rows.find(r => languageRowMatchesId(r, id)) : undefined;
  return languageRowFields(matched) || {};
}

/** Local Activate/Deactivate override; only valid while `recordId` still matches. */
export type LanguageRowOverride = { id: string; fields: Record<string, unknown> };

/** Return override fields when scoped to the current recordId; otherwise null. */
export function languageRowOverrideFields(
  override: LanguageRowOverride | null | undefined,
  recordId: string | null | undefined
): Record<string, unknown> | null {
  const id = String(recordId || '').trim();
  if (!override || !id || override.id !== id) return null;
  return override.fields;
}

/**
 * Resolve Language form record id from an explicit prop, else route params/query
 * (same keys as ChoyFormView / useFormViewLifecycle).
 */
export function resolveLanguageRecordId(opts: {
  recordId?: string | null;
  routeParams?: Record<string, unknown> | null;
  routeQuery?: Record<string, unknown> | null;
}): string {
  const fromProp = String(opts.recordId || '').trim();
  if (fromProp && fromProp !== 'undefined' && fromProp !== 'null') return fromProp;
  const params = opts.routeParams || {};
  const query = opts.routeQuery || {};
  const raw = params.recordId ?? params.id ?? params.Id ?? query.recordId ?? query.id ?? query.Id;
  const fromRoute = String(raw ?? '').trim();
  if (!fromRoute || fromRoute === 'undefined' || fromRoute === 'null' || fromRoute === 'new') return '';
  return fromRoute;
}

/** Call Language.Activate then refresh the form row and active-language overlays. */
export async function activateLanguage(opts: {
  activate: (req: { Id: string }) => Promise<unknown>;
  recordId: string;
  reloadRecord?: () => Promise<void>;
  reloadActiveLanguages?: () => Promise<void>;
}): Promise<void> {
  const id = String(opts.recordId || '').trim();
  if (!id) return;
  await opts.activate({ Id: id });
  try {
    if (opts.reloadRecord) await opts.reloadRecord();
  } catch {
    // Row refresh is best-effort; overlays still refresh below.
  }
  await (opts.reloadActiveLanguages || reloadActiveLanguagesAfterLanguageSave)();
}

/** Call Language.Deactivate then refresh the form row and active-language overlays. */
export async function deactivateLanguage(opts: {
  deactivate: (req: { Id: string }) => Promise<unknown>;
  recordId: string;
  reloadRecord?: () => Promise<void>;
  reloadActiveLanguages?: () => Promise<void>;
}): Promise<void> {
  const id = String(opts.recordId || '').trim();
  if (!id) return;
  await opts.deactivate({ Id: id });
  try {
    if (opts.reloadRecord) await opts.reloadRecord();
  } catch {
    // Row refresh is best-effort; overlays still refresh below.
  }
  await (opts.reloadActiveLanguages || reloadActiveLanguagesAfterLanguageSave)();
}
