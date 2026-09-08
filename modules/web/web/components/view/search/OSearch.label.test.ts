// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
/**
 * T4.1 / T4.6: Search field menus use resolveFieldLabel and must not trigger
 * full-model FieldsGet on open (D13).
 */

const osearchSrc = readFileSync(resolve(__dirname, './OSearch.vue'), 'utf8');
const helperSrc = readFileSync(resolve(__dirname, '../../../composables/search/useSearchFieldOptions.ts'), 'utf8');

test('OSearch availableFields label contract (T4.1 / T4.6) > delegates filterable fields to useFilterableSearchFields / resolveFieldLabel', () => {
  expect(osearchSrc).toContain('useFilterableSearchFields');
  expect(helperSrc).toContain('resolveFieldLabel');
  expect(helperSrc).not.toMatch(/label:\s*prop\b/);
});

test('OSearch availableFields label contract (T4.1 / T4.6) > does not call ensureFieldsGet / FieldsGet when building Search menus (T4.6)', () => {
  expect(osearchSrc).not.toMatch(/\bensureFieldsGet\b/);
  expect(osearchSrc).not.toMatch(/\bFieldsGet\b/);
  expect(helperSrc).not.toMatch(/\bensureFieldsGet\b/);
  expect(helperSrc).not.toMatch(/\bFieldsGet\b/);
});

