// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { filtersSignature, shouldApplyControlledFilters } from './controlledFilters';
import type { ConditionGroup } from '@/web/web/query/types';

const g = (field: string, value: string): ConditionGroup => ({
  id: 'g1',
  logic: 'And',
  children: [{ id: 'c1', field, operator: '=', value }],
});

test('controlledFilters sync: filtersSignature is stable for equivalent trees', () => {
  expect(filtersSignature([g('Name', 'a')])).toBe(filtersSignature([g('Name', 'a')]));
  expect(filtersSignature([g('Name', 'a')])).not.toBe(filtersSignature([g('Name', 'b')]));
});

test('controlledFilters sync: does not apply when local already matches incoming', () => {
  const local = [g('Name', 'a')];
  const decision = shouldApplyControlledFilters({ local, incoming: local, lastEmittedSig: '' });
  expect(decision.apply).toBe(false);
});

test('controlledFilters sync: ignores stale echo of last emit while local moved ahead', () => {
  const emitted = [g('Name', 'a')];
  const local = [g('Name', 'b')];
  const lastEmittedSig = filtersSignature(emitted);
  const decision = shouldApplyControlledFilters({
    local,
    incoming: emitted,
    lastEmittedSig,
  });
  expect(decision.apply).toBe(false);
  expect(decision.acknowledged).toBe(true);
});

test('controlledFilters sync: ignores lagging parent snapshots while awaiting echo of a newer emit', () => {
  const older = [g('Name', 'a')];
  const newer = [g('Name', 'b')];
  const lastEmittedSig = filtersSignature(newer);
  const decision = shouldApplyControlledFilters({
    local: newer,
    incoming: older,
    lastEmittedSig,
    awaitingEcho: true,
  });
  expect(decision.apply).toBe(false);
  expect(decision.acknowledged).toBe(false);
});

test('controlledFilters sync: ignores lagging empty parent while awaiting echo of a clear', () => {
  const older = [g('Name', 'a')];
  const decision = shouldApplyControlledFilters({
    local: [],
    incoming: older,
    lastEmittedSig: '',
    awaitingEcho: true,
  });
  expect(decision.apply).toBe(false);
  expect(decision.acknowledged).toBe(false);
});

test('controlledFilters sync: acknowledges empty echo after clearing filters', () => {
  const decision = shouldApplyControlledFilters({
    local: [],
    incoming: [],
    lastEmittedSig: '',
    awaitingEcho: true,
  });
  expect(decision.apply).toBe(false);
  expect(decision.acknowledged).toBe(true);
});

test('controlledFilters sync: ignores delayed clear echo after local created a new filter', () => {
  // clear → lastEmittedSig '' → user adds a filter before parent echoes []
  const local = [g('Name', 'new')];
  const decision = shouldApplyControlledFilters({
    local,
    incoming: [],
    lastEmittedSig: '',
    awaitingEcho: true,
  });
  expect(decision.apply).toBe(false);
  expect(decision.acknowledged).toBe(true);
});

test('controlledFilters sync: still applies external clear when we never emitted an empty fingerprint', () => {
  const local = [g('Name', 'a')];
  const decision = shouldApplyControlledFilters({
    local,
    incoming: [],
    lastEmittedSig: '',
    awaitingEcho: false,
  });
  expect(decision.apply).toBe(true);
  expect(decision.normalized).toEqual([]);
});

test('controlledFilters sync: applies external parent updates that are not our last emit', () => {
  const local = [g('Name', 'a')];
  const incoming = [g('Code', 'z')];
  const decision = shouldApplyControlledFilters({
    local,
    incoming,
    lastEmittedSig: filtersSignature(local),
  });
  expect(decision.apply).toBe(true);
  expect((decision.normalized[0].children[0] as any).field).toBe('Code');
});

test('controlledFilters sync: filtersSignature handles empty and non-serializable trees', () => {
  expect(filtersSignature(null)).toBe('');
  expect(filtersSignature([])).toBe('');
  const cyclic: any = g('Name', 'a');
  (cyclic as any).self = cyclic;
  expect(filtersSignature([cyclic])).toBe('len:1');
});

