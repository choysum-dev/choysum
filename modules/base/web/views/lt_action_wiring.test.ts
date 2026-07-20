// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from 'vitest';
import { readdirSync, readFileSync } from 'node:fs';
import { resolve } from 'node:path';

/**
 * Loads every base web view source so `_lt` migrations stay wired.
 */
function viewSources(): Array<{ file: string; source: string }> {
  const dir = resolve(__dirname, '../views');
  return readdirSync(dir)
    .filter((name) => name.endsWith('View.vue'))
    .map((file) => ({
      file,
      source: readFileSync(resolve(dir, file), 'utf8'),
    }));
}

describe('base view _lt action wiring', () => {
  it('binds entity titles with _lt and shared createTranslate helpers', () => {
    const views = viewSources();
    expect(views.length).toBeGreaterThan(10);

    for (const { file, source } of views) {
      if (!source.includes('defineModelActions(')) {
        continue;
      }
      expect(source, file).toContain("const { _t, _lt } = createTranslate('base'");
      expect(source, file).toMatch(/entityTitle:\s*_lt\('/);
      expect(source, file).not.toContain('_tRef');
      expect(source, file).not.toMatch(/output:\s*'reference'/);
    }
  });
});
