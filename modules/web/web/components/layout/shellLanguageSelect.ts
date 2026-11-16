// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/** Shell locale menu: set language then remount when the switch succeeds. */
export async function shellLanguageSelect(
  code: string,
  setLang: (lang: string) => Promise<boolean>,
  remount: () => Promise<void>,
): Promise<void> {
  if (!code) {
    return;
  }
  const ok = await setLang(code);
  if (ok) {
    await remount();
  }
}