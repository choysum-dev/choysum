// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { isValidPersonNameFormat } from '@/core/service/utils/person_name';
import { createTranslate } from '@/web/web/i18n';
import { useI18nStore } from '@/web/web/stores/i18nStore';

const { _t: _tReloadToast } = createTranslate('base', { scope: 'web/views/LanguageFormView' });

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
 * Merge form draft fields onto the store row only when the draft Id matches the
 * slot recordId (avoids stale draft A driving CTAs while record B loads).
 */
export function mergeLanguageFormFields(opts: {
  recordId: string;
  fromStore: Record<string, unknown>;
  formData?: Record<string, unknown> | null;
}): Record<string, unknown> {
  const id = String(opts.recordId || '').trim();
  const fromForm = opts.formData && typeof opts.formData === 'object' ? opts.formData : {};
  const formId = String(fromForm.Id || '').trim();
  if (id && formId && formId === id) {
    return { ...opts.fromStore, ...fromForm };
  }
  return opts.fromStore;
}

/** Prefer Browse override over silent display reload while the form is editing. */
export function shouldPreferBrowseReloadAfterLanguageAction(viewMode: string | null | undefined): boolean {
  return viewMode === 'edit' || viewMode === 'create';
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

export type LanguageReloadTerminologyResult = {
  Code?: string;
  ModulesScanned?: number;
  ModulesWithPo?: number;
  Upserted?: number;
  SkippedOverride?: number;
  RejectedNoCtxt?: number;
  SkippedObsolete?: number;
  PurgedRetired?: number;
};

export type LanguageReloadToastLevel = 'success' | 'warning';

/** Toast payload after ReloadTerminology, branched on whether packaged PO was found. */
export type LanguageReloadToast = {
  level: LanguageReloadToastLevel;
  title: string;
  description?: string;
};

/**
 * Build success / warning copy for Update translations.
 * ModulesWithPo === 0 is never treated as a successful sync.
 * Msgids use `_tReloadToast` so `choysum i18n extract` can discover them.
 */
export function buildLanguageReloadTerminologyToast(opts: {
  code: string;
  modulesScanned: number;
  modulesWithPo: number;
  upserted: number;
  isActive?: boolean | null;
}): LanguageReloadToast {
  const code = String(opts.code || '').trim() || '—';
  const scanned = Math.max(0, Number(opts.modulesScanned) || 0);
  const withPo = Math.max(0, Number(opts.modulesWithPo) || 0);
  const upserted = Math.max(0, Number(opts.upserted) || 0);
  const inactiveNote =
    opts.isActive === false
      ? _tReloadToast('This does not enable the language in Preferences or the guest switcher.')
      : undefined;

  if (withPo > 0) {
    return {
      level: 'success',
      title: _tReloadToast(
        'Synced packaged translations for %s: %s terms from %s/%s modules',
        code,
        String(upserted),
        String(withPo),
        String(scanned)
      ),
      ...(inactiveNote ? { description: inactiveNote } : {}),
    };
  }

  return {
    level: 'warning',
    title: _tReloadToast('No packaged translations found for %s', code),
    description: [
      _tReloadToast('Scanned %s installed modules; none provide i18n/%s.po.', String(scanned), code),
      inactiveNote,
    ]
      .filter(Boolean)
      .join(' '),
  };
}

/**
 * Call Language.ReloadTerminology then optionally refresh the FE catalog when the
 * current UI language matches the reloaded Code.
 */
export async function reloadLanguageTerminology(opts: {
  reload: (params: { Code?: string; LanguageId?: string }) => Promise<LanguageReloadTerminologyResult>;
  code?: string;
  languageId?: string;
  /** Prefer a getter so the active lang is read after reload completes. */
  currentLang?: string | null | (() => string | null | undefined);
  revalidateIfCurrent?: () => Promise<unknown>;
  onDone?: (result: LanguageReloadTerminologyResult) => void;
}): Promise<LanguageReloadTerminologyResult | null> {
  const code = String(opts.code || '').trim();
  const languageId = String(opts.languageId || '').trim();
  if (!code && !languageId) return null;
  const result = await opts.reload({
    ...(code ? { Code: code } : {}),
    ...(languageId ? { LanguageId: languageId } : {}),
  });
  const resolvedCode = String(result?.Code || code).trim();
  const currentRaw = typeof opts.currentLang === 'function' ? opts.currentLang() : opts.currentLang;
  const current = String(currentRaw ?? '').trim();
  if (resolvedCode && current && resolvedCode === current && opts.revalidateIfCurrent) {
    try {
      await opts.revalidateIfCurrent();
    } catch {
      // Catalog refresh is best-effort; toast still reports import stats.
    }
  }
  opts.onDone?.(result);
  return result;
}
