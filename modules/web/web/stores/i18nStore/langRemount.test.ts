// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { afterLangChange } from './langRemount';

test('afterLangChange: calls injected reload', async () => {
  let reloaded = false;
  await afterLangChange({ reload: () => {
    reloaded = true;
  } });
  expect(reloaded).toBe(true);
});

test('afterLangChange: calls location.reload when no injection', async () => {
  let reloaded = false;
  const prev = globalThis.location;
  (globalThis as any).location = { reload: () => { reloaded = true; } };
  try {
    await afterLangChange();
    expect(reloaded).toBe(true);
  } finally {
    (globalThis as any).location = prev;
  }
});