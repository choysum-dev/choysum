// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { readFileSync } from 'node:fs';
import { describe, expect, it } from 'vitest';

function source(relativePath: string): string {
  return readFileSync(new URL(relativePath, import.meta.url), 'utf8');
}

describe('frontend terminology display contract', () => {
  it('uses direct $t term reference expressions in Vue templates', () => {
    const header = source('../components/layout/OHeader.vue');
    const breadcrumb = source('../components/view/OBreadcrumb.vue');

    expect(header).toContain('$t(');
    expect(header).toContain('titleText.src ||');
    expect(breadcrumb).toContain('$t(crumb.titleText.key, crumb.titleText.src || crumb.title)');
  });

  it('keeps non-template term reference consumers on translateTerm', () => {
    for (const path of [
      '../composables/useMenu.ts',
      '../components/field/OSelectionField.vue',
      '../router/index.ts',
    ]) {
      expect(source(path)).toContain('translateTerm(');
    }
  });

  it('does not reintroduce object-specific terminology wrappers', () => {
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
});
