// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  captureDialogFocusTarget,
  formatModuleKanbanDate,
  formatModuleOpSummary,
  isModuleInstalled,
  manifestSummaryText,
  moduleStatusBadgeClass,
  resolveModuleKanbanCardId,
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
  expect(formatModuleKanbanDate('not-a-date')).toBe('not-a-date');
  const formatted = formatModuleKanbanDate('2026-09-28T10:05:00Z');
  expect(formatted.length).toBeGreaterThan(10);
});

test('formatModuleOpSummary: string / message / code / object / throw', () => {
  expect(formatModuleOpSummary(undefined)).toBe('');
  expect(formatModuleOpSummary('ok')).toBe('ok');
  expect(formatModuleOpSummary({ message: 'm' })).toBe('m');
  expect(formatModuleOpSummary({ code: 'C1' })).toBe('C1');
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
  expect(resolveModuleKanbanCardId(null)).toBe('');
  expect(resolveModuleKanbanCardId({})).toBe('');
});
