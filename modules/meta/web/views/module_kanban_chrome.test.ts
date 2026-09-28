// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  captureDialogFocusTarget,
  createLaneSyncGate,
  createPlanDialogSessionGate,
  formatModuleKanbanDate,
  formatModuleOpSummary,
  isModuleInstalled,
  manifestSummaryText,
  moduleStatusBadgeClass,
  resolveModuleKanbanCardId,
  resolveModuleKanbanCardKey,
  shouldRecoverStaleKanbanSearch,
} from './module_kanban_chrome';

test('moduleStatusBadgeClass: unavailable and known statuses', () => {
  expect(moduleStatusBadgeClass('installed', false)).toContain('destructive');
  expect(moduleStatusBadgeClass('installed')).toContain('emerald');
  expect(moduleStatusBadgeClass('queued')).toContain('amber');
  expect(moduleStatusBadgeClass('broken')).toContain('destructive');
  expect(moduleStatusBadgeClass('other')).toContain('muted');
});

test('isModuleInstalled: case-insensitive installed only', () => {
  expect(isModuleInstalled('Installed')).toBe(true);
  expect(isModuleInstalled('uninstalled')).toBe(false);
  expect(isModuleInstalled(undefined)).toBe(false);
});

test('formatModuleKanbanDate: empty and valid timestamps', () => {
  expect(formatModuleKanbanDate(undefined)).toBe('');
  expect(formatModuleKanbanDate(null)).toBe('');
  expect(formatModuleKanbanDate(Number.NaN)).toBe('');
  expect(formatModuleKanbanDate(Number.POSITIVE_INFINITY)).toBe('');
  // Epoch zero is a valid timestamp; do not treat it as empty.
  expect(formatModuleKanbanDate(0).length).toBeGreaterThan(0);
  expect(formatModuleKanbanDate('not-a-date')).toBe('not-a-date');
  const formatted = formatModuleKanbanDate('2026-09-28T10:05:00Z');
  expect(formatted.length).toBeGreaterThan(10);
  const fromEpoch = formatModuleKanbanDate(Date.UTC(2026, 8, 28, 10, 5));
  expect(fromEpoch.length).toBeGreaterThan(10);
  const fromDate = formatModuleKanbanDate(new Date(Date.UTC(2026, 8, 28, 10, 5)));
  expect(fromDate.length).toBeGreaterThan(10);
  const boom = {
    toString() {
      throw new Error('boom');
    },
    valueOf() {
      throw new Error('boom');
    },
  };
  expect(formatModuleKanbanDate(boom)).toBe('');
});

test('formatModuleOpSummary: string / message / code / object / throw', () => {
  expect(formatModuleOpSummary(undefined)).toBe('');
  expect(formatModuleOpSummary('ok')).toBe('ok');
  expect(formatModuleOpSummary({ message: 'm' })).toBe('m');
  expect(formatModuleOpSummary({ code: 'C1' })).toBe('C1');
  // Empty/blank message must fall through to code (not short-circuit on key presence).
  expect(formatModuleOpSummary({ message: '', code: 'C1' })).toBe('C1');
  expect(formatModuleOpSummary({ message: '   ', code: 'C2' })).toBe('C2');
  // Non-scalar message/code fall through to JSON.stringify (or the other scalar key).
  expect(formatModuleOpSummary({ message: { nested: true }, code: 'C3' })).toBe('C3');
  expect(formatModuleOpSummary({ message: { nested: true }, a: 1 })).toContain('nested');
  expect(formatModuleOpSummary({ code: 42 })).toBe('42');
  expect(formatModuleOpSummary({ a: 1 })).toContain('a');
  const cyclic: any = {};
  cyclic.self = cyclic;
  expect(formatModuleOpSummary(cyclic)).toContain('[object');
});

test('manifestSummaryText: prefers short_desc then fallbacks', () => {
  expect(manifestSummaryText(null)).toBe('');
  expect(manifestSummaryText({ short_desc: 's' })).toBe('s');
  expect(manifestSummaryText({ name: 'n' })).toBe('n');
  expect(manifestSummaryText({ name: 1 })).toBe('');
});

test('shouldRecoverStaleKanbanSearch: only when stale and idle', () => {
  expect(shouldRecoverStaleKanbanSearch({ completedSeq: 1, latestSeq: 2, inFlight: 0 })).toBe(true);
  expect(shouldRecoverStaleKanbanSearch({ completedSeq: 2, latestSeq: 2, inFlight: 0 })).toBe(false);
  expect(shouldRecoverStaleKanbanSearch({ completedSeq: 1, latestSeq: 2, inFlight: 1 })).toBe(false);
});

test('captureDialogFocusTarget / resolveModuleKanbanCardId', () => {
  expect(captureDialogFocusTarget({ activeElement: null } as any)).toBeNull();
  expect(captureDialogFocusTarget({ activeElement: {} } as any)).toBeNull();
  const el = { focus() {} } as unknown as HTMLElement;
  expect(captureDialogFocusTarget({ activeElement: el } as any)).toBe(el);
  expect(resolveModuleKanbanCardId({ Id: '  m1  ' })).toBe('m1');
  expect(resolveModuleKanbanCardId({ Id: 9 })).toBe('9');
  expect(resolveModuleKanbanCardId(null)).toBe('');
  expect(resolveModuleKanbanCardId({})).toBe('');
  expect(resolveModuleKanbanCardId({ Id: { nested: true } })).toBe('');
  expect(resolveModuleKanbanCardId({ Id: Number.NaN })).toBe('');
});

test('createPlanDialogSessionGate: ignore superseded plan responses', () => {
  const gate = createPlanDialogSessionGate();
  const first = gate.begin();
  expect(gate.isCurrent(first)).toBe(true);
  gate.invalidate();
  expect(gate.isCurrent(first)).toBe(false);
  const second = gate.begin();
  expect(gate.isCurrent(first)).toBe(false);
  expect(gate.isCurrent(second)).toBe(true);
});

test('resolveModuleKanbanCardKey: fail-closed id then ModuleName then synthetic', () => {
  expect(resolveModuleKanbanCardKey({ Id: '  m1  ' }, 'lane', 0)).toBe('m1');
  expect(resolveModuleKanbanCardKey({ Id: { nested: true }, ModuleName: ' core ' }, 'lane', 2)).toBe(
    'core',
  );
  expect(resolveModuleKanbanCardKey({ Id: Number.NaN }, 'lane', 3)).toBe('lane-3');
  expect(resolveModuleKanbanCardKey({}, 'installed', 1)).toBe('installed-1');
});

test('createLaneSyncGate: waiters drain after owner leave', async () => {
  const gate = createLaneSyncGate();
  expect(await gate.enter()).toBe('run');
  let waited = false;
  const waiter = gate.enter().then(mode => {
    waited = true;
    expect(mode).toBe('waited');
  });
  expect(gate.shouldResync()).toBe(true);
  gate.beginPass();
  expect(gate.shouldResync()).toBe(false);
  expect(waited).toBe(false);
  gate.leave();
  await waiter;
  expect(waited).toBe(true);
});
