// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { readFileSync } from 'node:fs';
function source(relativePath: string): string {
  return readFileSync(new URL(relativePath, import.meta.url), 'utf8');
}

test('frontend terminology display contract: uses direct $t term reference expressions in Vue templates', () => {
  const header = source('../components/layout/OHeader.vue');
  const breadcrumb = source('../components/view/OBreadcrumb.vue');

  expect(header).toContain('$t(');
  expect(header).toContain('titleText.src ||');
  expect(breadcrumb).toContain('$t(crumb.titleText.key, crumb.titleText.src || crumb.title)');
});

test('frontend terminology display contract: keeps non-template term reference consumers on translateTerm', () => {
  for (const path of [
    '../composables/useMenu.ts',
    '../composables/resolveFieldLabel.ts',
    '../router/index.ts',
  ]) {
    expect(source(path)).toContain('translateTerm(');
  }
});

test('frontend terminology display contract: does not translate selection options via translateTerm (D5)', () => {
  const selection = source('../components/field/OSelectionField.vue');
  expect(selection).not.toContain('translateTerm(');
  expect(selection).not.toMatch(/\blabelText\b/);
});

test('frontend terminology display contract: does not reintroduce object-specific terminology wrappers', () => {
  const frontend = [
    '../components/layout/OHeader.vue',
    '../components/view/OBreadcrumb.vue',
    '../components/field/OSelectionField.vue',
    '../composables/useBreadcrumb.ts',
    '../composables/useMenu.ts',
    '../router/index.ts',
    '../stores/breadcrumbStore/index.ts',
  ].map(source).join('\n');

  const forbidden = [
    ['menu', 'Title'],
    ['resolve', 'Breadcrumb', 'Title'],
    ['selection', 'Label'],
    ['resolve', 'Route', 'Title'],
    ['display', 'Title'],
  ].map(parts => parts.join(''));
  expect(forbidden.filter(name => frontend.includes(name))).toEqual([]);
});

