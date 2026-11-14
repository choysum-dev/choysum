// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { viewBusyHostAttrs, viewBusyHostClass } from './viewBusy';

describe('viewBusy helpers', () => {
  test('viewBusyHostAttrs is empty when idle; block marks aria-busy+inert', () => {
    expect(viewBusyHostAttrs(false)).toEqual({});
    expect(viewBusyHostAttrs(true)).toEqual({ 'aria-busy': 'true', inert: true });
    expect(viewBusyHostAttrs(true, 'block')).toEqual({ 'aria-busy': 'true', inert: true });
  });

  test('stale mode sets aria-busy without inert', () => {
    expect(viewBusyHostAttrs(true, 'stale')).toEqual({ 'aria-busy': 'true' });
  });

  test('viewBusyHostClass: block disables pointer events; stale uses opacity', () => {
    expect(viewBusyHostClass(false)).toContain('relative');
    expect(viewBusyHostClass(false)).not.toContain('pointer-events-none');
    expect(viewBusyHostClass(true, 'block')).toContain('pointer-events-none');
    expect(viewBusyHostClass(true, 'stale')).toContain('opacity-70');
    expect(viewBusyHostClass(true, 'stale')).not.toContain('pointer-events-none');
    expect(viewBusyHostClass(true, 'block', 'extra')).toContain('extra');
  });
});
