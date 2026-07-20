// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it, vi } from 'vitest';

import { useFilterEditorBindings } from './useFilterEditorBindings';

describe('useFilterEditorBindings static meta (T4.2)', () => {
  it('metaTypeOf reads only static fieldsMetadata.type', () => {
    const FieldsGet = vi.fn(async () => ({}));
    const ensureFieldsGet = vi.fn(async () => ({}));
    const store = {
      fieldsMetadata: {
        Status: { type: 'selection' },
        Name: { type: 'varchar' },
      },
      FieldsGet,
      ensureFieldsGet,
    } as any;

    const { metaTypeOf, getOperatorOptionsForField } = useFilterEditorBindings(store);
    expect(metaTypeOf('Status')).toBe('selection');
    expect(metaTypeOf('Name')).toBe('varchar');
    expect(getOperatorOptionsForField('Status').length).toBeGreaterThan(0);
    expect(FieldsGet).not.toHaveBeenCalled();
    expect(ensureFieldsGet).not.toHaveBeenCalled();
  });

  it('source does not await FieldsGet (D2 / D13)', () => {
    const src = readFileSync(resolve(__dirname, './useFilterEditorBindings.ts'), 'utf8');
    expect(src).not.toMatch(/\bensureFieldsGet\b/);
    expect(src).not.toMatch(/\bFieldsGet\b/);
  });
});
