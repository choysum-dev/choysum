// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { projectToSelection } from './selection';

class Probe {
  Id = '';
  Name = '';
  Partner?: { Id: string; Name: string };
}

test('projectToSelection: empty and * return the same row instance', () => {
  const row = Object.assign(Object.create(Probe.prototype), { Id: '1', Name: 'n' }) as Probe;
  expect(projectToSelection(row, [] as any)).toBe(row);
  expect(projectToSelection(row, ['*'] as any)).toBe(row);
});

test('projectToSelection: narrow selection strips unselected keys and keeps prototype', () => {
  const row = Object.assign(Object.create(Probe.prototype), { Id: '1', Name: 'n' }) as Probe;
  const projected = projectToSelection(row, ['Name'] as any) as Record<string, unknown>;
  expect(projected.Name).toBe('n');
  expect('Id' in projected).toBe(false);
  expect(Object.getPrototypeOf(projected)).toBe(Probe.prototype);
  expect(row.Id).toBe('1');
});

test('projectToSelection: deep relation entries keep the top-level key only', () => {
  const row = Object.assign(Object.create(Probe.prototype), {
    Id: '1',
    Partner: { Id: 'p1', Name: 'Acme' },
  }) as Probe;
  const projected = projectToSelection(row, [{ Partner: ['Name'] }] as any) as Record<string, unknown>;
  expect(projected.Partner).toEqual({ Id: 'p1', Name: 'Acme' });
  expect('Id' in projected).toBe(false);
});
