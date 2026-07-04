// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, test } from 'vitest';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';

function source(name: string): string {
  return readFileSync(resolve(__dirname, name), 'utf8');
}

describe('OLayout component contract', () => {
  test('contentSpacing returns { padding: false } for none spacing', () => {
    const s = source('OLayout.vue');

    expect(s).toContain("if (props.spacing === 'none') return { padding: false };");
  });

  test('contentSpacing returns padding: true with paddingSize otherwise', () => {
    const s = source('OLayout.vue');

    expect(s).toContain("return { padding: true, paddingSize: props.spacing as 'small' | 'medium' | 'large' };");
  });

  test('padding-top compensation is scoped to fixed-header state', () => {
    const s = source('OLayout.vue');

    expect(s).toContain('o-layout--fixed-header');
    expect(s).toContain('padding-top: var(--o-header-height);');
  });

  test('OContent receives spacing via v-bind', () => {
    const s = source('OLayout.vue');

    expect(s).toContain('<OContent v-bind="contentSpacing">');
  });
});
