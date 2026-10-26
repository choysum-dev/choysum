// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { viewBusyHostAttrs, viewBusyHostClass } from './viewBusy';

describe('viewBusy helpers', () => {
  test('viewBusyHostAttrs is empty when idle and marks aria-busy+inert when busy', () => {
    expect(viewBusyHostAttrs(false)).toEqual({});
    expect(viewBusyHostAttrs(true)).toEqual({ 'aria-busy': 'true', inert: true });
  });

  test('viewBusyHostClass keeps relative host and disables pointer events while busy', () => {
    expect(viewBusyHostClass(false)).toContain('relative');
    expect(viewBusyHostClass(false)).not.toContain('pointer-events-none');
    expect(viewBusyHostClass(true)).toContain('pointer-events-none');
    expect(viewBusyHostClass(true, 'extra')).toContain('extra');
  });
});
