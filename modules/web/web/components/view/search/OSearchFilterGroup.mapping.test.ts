// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, test } from 'vitest';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';

function source(): string {
  return readFileSync(resolve(__dirname, 'OSearchFilterCondition.vue'), 'utf8');
}

describe('OSearchFilterCondition field resolver mapping', () => {
  test('maps binary/image metadata types to OBinaryField/OImageField', () => {
    const s = source();

    expect(s).toContain("import OBinaryField from '@/web/web/components/field/OBinaryField.vue';");
    expect(s).toContain("import OImageField from '@/web/web/components/field/OImageField.vue';");
    expect(s).toContain("case 'binary':");
    expect(s).toContain('return OBinaryField;');
    expect(s).toContain("case 'image':");
    expect(s).toContain('return OImageField;');
  });

  test('maps selection metadata type to OSelectionField (T4.3)', () => {
    const s = source();
    expect(s).toContain("import OSelectionField from '@/web/web/components/field/OSelectionField.vue';");
    expect(s).toContain("case 'selection':");
    expect(s).toContain('return OSelectionField;');
  });

  test('filter value binding reaches model store and forces empty label (T4.4)', () => {
    const s = source();
    expect(s).toContain('binding.store = {');
    expect(s).toContain('isReadonly: false');
    expect(s).toContain(':store="store"');
    expect(s).toContain(`:label="''"`);
  });
});
