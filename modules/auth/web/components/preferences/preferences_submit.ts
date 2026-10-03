// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Preferences save path: FormView Write (`defaultSubmit`) then i18n / token
 * side effects. Callers must not issue a second UpdateById.
 */
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
  let userId = String(opts.userId || '').trim();
  if (!userId) {
    opts.onError(opts.missingUserMessage);
    return false;
  }
  try {
    if (!opts.hasCurrentUserId && opts.loadUser) {
      await opts.loadUser();
    }
    const record = (await opts.defaultSubmit()) || {};
    const languageId = record.LanguageId ?? opts.formData.LanguageId ?? null;
    const timezone = record.Timezone ?? opts.formData.Timezone ?? null;
    opts.patchCurrentUser(languageId, timezone);
    await opts.applyLanguage(languageId);
    await opts.refreshToken();
    await opts.afterLocaleChange();
    opts.onSuccess();
    return true;
  } catch (err) {
    const message = err instanceof Error ? String(err.message || '').trim() : '';
    opts.onError(message || opts.failedMessage);
    return false;
  }
}
