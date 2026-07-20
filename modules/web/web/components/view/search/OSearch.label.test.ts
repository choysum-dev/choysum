// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

/**
 * T4.1 / T4.6: Search field menus use resolveFieldLabel and must not trigger
 * full-model FieldsGet on open (D13).
 */
describe('OSearch availableFields label contract (T4.1 / T4.6)', () => {
  const src = readFileSync(resolve(__dirname, './OSearch.vue'), 'utf8');

  it('uses resolveFieldLabel for availableFields labels', () => {
    expect(src).toContain("import { resolveFieldLabel } from '@/web/web/composables/resolveFieldLabel'");
    expect(src).toContain('resolveFieldLabel({');
    expect(src).not.toMatch(/label:\s*k\b/);
  });

  it('does not call ensureFieldsGet / FieldsGet when building Search menus (T4.6)', () => {
    expect(src).not.toMatch(/\bensureFieldsGet\b/);
    expect(src).not.toMatch(/\bFieldsGet\b/);
  });
});
