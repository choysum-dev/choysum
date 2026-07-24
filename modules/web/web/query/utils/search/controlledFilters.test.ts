// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from 'vitest';
import { filtersSignature, shouldApplyControlledFilters } from './controlledFilters';
import type { ConditionGroup } from '@/web/web/query/types';

const g = (field: string, value: string): ConditionGroup => ({
  id: 'g1',
  logic: 'And',
  children: [{ id: 'c1', field, operator: '=', value }],
});

describe('controlledFilters sync', () => {
  it('filtersSignature is stable for equivalent trees', () => {
    expect(filtersSignature([g('Name', 'a')])).toBe(filtersSignature([g('Name', 'a')]));
    expect(filtersSignature([g('Name', 'a')])).not.toBe(filtersSignature([g('Name', 'b')]));
  });

  it('does not apply when local already matches incoming', () => {
    const local = [g('Name', 'a')];
    const decision = shouldApplyControlledFilters({ local, incoming: local, lastEmittedSig: '' });
    expect(decision.apply).toBe(false);
  });

  it('ignores stale echo of last emit while local moved ahead', () => {
    const emitted = [g('Name', 'a')];
    const local = [g('Name', 'b')];
    const lastEmittedSig = filtersSignature(emitted);
    const decision = shouldApplyControlledFilters({
      local,
      incoming: emitted,
      lastEmittedSig,
    });
    expect(decision.apply).toBe(false);
  });

  it('applies external parent updates that are not our last emit', () => {
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
});
