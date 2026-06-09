// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, test } from 'vitest';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';

function source(): string {
  return readFileSync(resolve(__dirname, 'OSearchFilterGroup.vue'), 'utf8');
}

describe('OSearchFilterGroup field resolver mapping', () => {
  test('maps binary/image metadata types to OBinaryField/OImageField', () => {
    const s = source();

    expect(s).toContain("import OBinaryField from '@/web/web/components/field/OBinaryField.vue';");
    expect(s).toContain("import OImageField from '@/web/web/components/field/OImageField.vue';");
    expect(s).toContain("case 'binary':");
    expect(s).toContain('return OBinaryField;');
    expect(s).toContain("case 'image':");
    expect(s).toContain('return OImageField;');
  });
});
