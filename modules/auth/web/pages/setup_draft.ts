// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * sessionStorage key for the in-progress setup wizard (tab-scoped, never holds passwords).
 * Without a password the wizard always resumes on the admin step, so no step is stored.
 */
export const SETUP_DRAFT_KEY = 'choysum.auth.setup.draft';

/** Persisted wizard fields; passwords are intentionally excluded. */
export const SETUP_DRAFT_FIELDS = [
  'Username',
  'Email',
  'OrgName',
  'CountryCode',
  'CurrencyCode',
  'Timezone',
  'OrgLanguageCode',
] as const;

export type SetupDraftField = (typeof SETUP_DRAFT_FIELDS)[number];

export type SetupDraft = {
  values: Partial<Record<SetupDraftField, string>>;
};

type DraftStorage = Pick<Storage, 'getItem' | 'setItem' | 'removeItem'>;

function defaultStorage(): DraftStorage | null {
  try {
    return typeof sessionStorage === 'undefined' ? null : sessionStorage;
  } catch {
    return null;
  }
}

/** Read the draft; malformed or missing data yields null. */
export function loadSetupDraft(storage: DraftStorage | null = defaultStorage()): SetupDraft | null {
  if (!storage) return null;
  let parsed: unknown;
  try {
    const raw = storage.getItem(SETUP_DRAFT_KEY);
    if (!raw) return null;
    parsed = JSON.parse(raw);
  } catch {
    return null;
  }
  if (!parsed || typeof parsed !== 'object') return null;
  const src = parsed as { values?: unknown };
  const values: SetupDraft['values'] = {};
  if (src.values && typeof src.values === 'object') {
    for (const key of SETUP_DRAFT_FIELDS) {
      const v = (src.values as Record<string, unknown>)[key];
      if (typeof v === 'string') values[key] = v;
    }
  }
  return { values };
}

/** Write whitelisted fields only; storage failures are ignored. */
export function saveSetupDraft(
  data: Record<string, unknown>,
  storage: DraftStorage | null = defaultStorage(),
): void {
  if (!storage) return;
  const values: SetupDraft['values'] = {};
  for (const key of SETUP_DRAFT_FIELDS) {
    const v = data[key];
    if (typeof v === 'string') values[key] = v;
    else if (v === null) values[key] = '';
  }
  try {
    storage.setItem(SETUP_DRAFT_KEY, JSON.stringify({ values }));
  } catch {
    // Quota or privacy mode: the wizard still works without a draft.
  }
}

export function clearSetupDraft(storage: DraftStorage | null = defaultStorage()): void {
  if (!storage) return;
  try {
    storage.removeItem(SETUP_DRAFT_KEY);
  } catch {
    // Ignore storage failures.
  }
}
