// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, test } from 'vitest';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';

function source(name: string): string {
  return readFileSync(resolve(__dirname, name), 'utf8');
}

describe('OPage component contract', () => {
  test('uses Vue 3.5 useId for unique page-title id', () => {
    const s = source('OPage.vue');

    expect(s).toContain("import { computed, useId } from 'vue';");
    expect(s).toContain('const pageTitleId = useId();');
    expect(s).not.toContain('getCurrentInstance');
    expect(s).not.toContain('pageTitleCounter');
  });

  test('pageTitleId is bound to h1 id and aria-labelledby', () => {
    const s = source('OPage.vue');

    expect(s).toContain(':id="pageTitleId"');
    expect(s).toContain(':aria-labelledby="title && !$slots.header ? pageTitleId : undefined"');
  });
});
