// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

export type NameCreateStoreLike = {
  // Params stay loose so WebModelStore.NameCreate (ClientModelService) remains assignable.
  NameCreate: (name: string, values?: any, options?: { nameField?: string }) => Promise<any>;
};

/** Normalize typeahead keyword for labels / NameCreate (nullish-safe). */
export function trimSearchKeyword(keyword: unknown): string {
  return String(keyword ?? '').trim();
}

/** Extract a non-empty Id / id from a NameCreate row. */
export function extractNameCreateRecordId(row: unknown): string | undefined {
  if (row == null || typeof row !== 'object') return undefined;
  const id = (row as { Id?: unknown; id?: unknown }).Id ?? (row as { id?: unknown }).id;
  if (id == null) return undefined;
  const s = String(id).trim();
  return s === '' ? undefined : s;
}

/** Normalize NameCreate catch value into a toast string. */
export function formatNameCreateError(error: unknown, fallback: string): string {
  const message = (error as { message?: unknown } | null | undefined)?.message;
  return String(message || error || fallback);
}

/**
 * Shared typeahead NameCreate click handler (PR-P2-M1).
 * Returns true when a row was created and onSuccess ran.
 */
export async function runNameCreateQuickCreate(opts: {
  busy: { value: boolean };
  store: NameCreateStoreLike | null | undefined;
  keyword: string;
  nameField?: string;
  failedMessage: string;
  onError: (message: string) => void;
  onSuccess: (row: unknown, id: string) => void | Promise<void>;
}): Promise<boolean> {
  if (opts.busy.value) return false;
  if (!opts.store) {
    opts.onError(opts.failedMessage);
    return false;
  }
  const trimmed = trimSearchKeyword(opts.keyword);
  if (!trimmed) return false;
  opts.busy.value = true;
  try {
    const row = await opts.store.NameCreate(
      trimmed,
      undefined,
      opts.nameField ? { nameField: opts.nameField } : undefined
    );
    const id = extractNameCreateRecordId(row);
    if (!id) {
      opts.onError(opts.failedMessage);
      return false;
    }
    await opts.onSuccess(row, id);
    return true;
  } catch (error: unknown) {
    opts.onError(formatNameCreateError(error, opts.failedMessage));
    return false;
  } finally {
    opts.busy.value = false;
  }
}
