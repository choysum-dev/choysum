// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Preferences save path: FormView Write (`defaultSubmit`) then i18n / token
 * side effects. Callers must not issue a second UpdateById.
 *
 * Post-write locale/token steps are best-effort: a successful Write must not be
 * reported as a failed save when applyLanguage / refreshToken later rejects.
 */

/** Normalize ManyToOne `{ Id }` / scalar LanguageId to a trimmed id or null. */
export function normalizePreferenceLanguageId(value: unknown): string | null {
  if (value == null || Array.isArray(value)) return null;
  if (typeof value === 'object') {
    if (!('Id' in value)) return null;
    const id = String((value as { Id?: unknown }).Id ?? '').trim();
    return id || null;
  }
  const id = String(value).trim();
  return id || null;
}

function formatPreferencesError(err: unknown, fallback: string): string {
  const message =
    err instanceof Error ? String(err.message || '').trim() : String(err ?? '').trim();
  return message || fallback;
}

export async function runPreferencesSubmit(opts: {
  userId: string;
  hasCurrentUserId: boolean;
  loadUser?: () => Promise<unknown>;
  missingUserMessage: string;
  defaultSubmit: () => Promise<Record<string, unknown> | null | undefined>;
  formData: Record<string, unknown>;
  patchCurrentUser: (languageId: unknown, timezone: unknown) => void;
  applyLanguage: (languageId: unknown) => Promise<void>;
  refreshToken: () => Promise<unknown>;
  afterLocaleChange: () => Promise<unknown>;
  onSuccess: () => void;
  onError: (message: string) => void;
  failedMessage: string;
}): Promise<boolean> {
  const userId = String(opts.userId || '').trim();
  if (!userId) {
    opts.onError(opts.missingUserMessage);
    return false;
  }

  let record: Record<string, unknown>;
  try {
    if (!opts.hasCurrentUserId && opts.loadUser) {
      await opts.loadUser();
    }
    record = (await opts.defaultSubmit()) || {};
  } catch (err) {
    opts.onError(formatPreferencesError(err, opts.failedMessage));
    return false;
  }

  const languageId = normalizePreferenceLanguageId(
    record.LanguageId ?? opts.formData.LanguageId ?? null,
  );
  const timezone = record.Timezone ?? opts.formData.Timezone ?? null;
  opts.patchCurrentUser(languageId, timezone);

  // Match loadUser: i18n wiring must not undo a successful preference Write.
  try {
    await opts.applyLanguage(languageId);
  } catch {
    // Best-effort.
  }

  try {
    await opts.refreshToken();
    await opts.afterLocaleChange();
  } catch {
    // Token/locale remount is best-effort after Write; still close as saved.
  }

  opts.onSuccess();
  return true;
}
