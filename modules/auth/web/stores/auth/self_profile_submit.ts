// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Self-service auth.User FormView save: Write (`defaultSubmit`) then session
 * side effects (appearance, language, token, locale remount). Callers must not
 * issue a second UpdateById.
 *
 * Post-write steps are best-effort: a successful Write must not be reported as
 * a failed save when applyLanguage / refreshToken later rejects.
 */

import { normalizeLanguageId } from './language_preference';

function formatSelfProfileError(err: unknown, fallback: string): string {
  const message =
    err instanceof Error ? String(err.message || '').trim() : String(err ?? '').trim();
  return message || fallback;
}

export async function runSelfProfileSubmit(opts: {
  userId: string;
  hasCurrentUserId: boolean;
  loadUser?: () => Promise<unknown>;
  missingUserMessage: string;
  defaultSubmit: () => Promise<Record<string, unknown> | null | undefined>;
  formData: Record<string, unknown>;
  patchCurrentUser: (
    languageId: unknown,
    timezone: unknown,
    theme: unknown,
    density: unknown,
  ) => void;
  applyLanguage: (languageId: unknown) => Promise<void>;
  /** Apply shell theme and density from User fields. */
  applyAppearance?: (theme: unknown, density: unknown) => void;
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
    opts.onError(formatSelfProfileError(err, opts.failedMessage));
    return false;
  }

  const languageId = normalizeLanguageId(record.LanguageId ?? opts.formData.LanguageId ?? null);
  const timezone = record.Timezone ?? opts.formData.Timezone ?? null;
  const theme = record.Theme ?? opts.formData.Theme ?? null;
  const density = record.Density ?? opts.formData.Density ?? null;
  opts.patchCurrentUser(languageId, timezone, theme, density);

  try {
    opts.applyAppearance?.(theme, density);
  } catch {
    // Best-effort shell theme/density apply.
  }

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
