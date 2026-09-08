// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
/**
 * D2 / T1.8: executor and useField must keep sync static fieldsMetadata reads.
 * They must not await FieldsGet / ensureFieldsGet.
 */

const roots = [
  resolve(__dirname, '../composables/useField.ts'),
  resolve(__dirname, '../query/executor.ts'),
  resolve(__dirname, '../composables/useOnchange.ts'),
];

test('FieldsGet D2 regression (T1.8): does not await FieldsGet or ensureFieldsGet in sync metadata consumers', () => {
  for (const file of roots) {
    const src = readFileSync(file, 'utf8');
    expect(src, file).not.toMatch(/\bensureFieldsGet\b/);
    expect(src, file).not.toMatch(/\bFieldsGet\b/);
    expect(src, file).not.toMatch(/await[^\n]*fieldsMetadata/);
  }
});

