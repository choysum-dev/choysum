// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  captureDialogFocusTarget,
  catalogInstalledStatus,
  formatModuleCatalogDate,
  formatModuleCatalogVersionLine,
  formatModuleOpSummary,
  parseCatalogSyncResult,
  catalogSyncSummaryPartial,
  isModuleInstalled,
  moduleCatalogCardApplication,
  moduleCatalogCardSummary,
  moduleStatusBadgeVariant,
  resolveModuleCatalogPrimaryAction,
  resolveModuleCatalogFormPrimaryAction,
  createPlanDialogSessionGate,
} from './module_catalog_chrome';

test('moduleStatusBadgeVariant: unavailable and known statuses', () => {
  expect(moduleStatusBadgeVariant('installed', false)).toBe('destructive');
  expect(moduleStatusBadgeVariant('installed', true)).toBe('outline');
  expect(moduleStatusBadgeVariant('installed', true, true)).toBe('secondary');
  expect(moduleStatusBadgeVariant('failed')).toBe('destructive');
  expect(moduleStatusBadgeVariant('uninstalled')).toBe('secondary');
  expect(moduleStatusBadgeVariant('queued')).toBe('secondary');
});

test('moduleCatalogCardSummary / Application: hide noise', () => {
  expect(moduleCatalogCardSummary({ ModuleName: 'meta', Summary: 'meta' })).toBe('');
  expect(moduleCatalogCardSummary({ ModuleName: 'meta', Summary: 'Meta' })).toBe('');
  expect(moduleCatalogCardSummary({ ModuleName: 'meta', Summary: 'Meta registry' })).toBe(
    'Meta registry',
  );
  expect(moduleCatalogCardSummary({ Summary: '  ' })).toBe('');
  expect(moduleCatalogCardApplication({ Application: '  ' })).toBe('');
  expect(moduleCatalogCardApplication({ Application: 'base' })).toBe('base');
});

test('isModuleInstalled / catalogInstalledStatus', () => {
  expect(isModuleInstalled('installed')).toBe(true);
  expect(isModuleInstalled('Installed')).toBe(true);
  expect(isModuleInstalled('uninstalled')).toBe(false);
  expect(catalogInstalledStatus({ Installed: true })).toBe('installed');
  expect(catalogInstalledStatus({ Installed: false })).toBe('uninstalled');
  expect(catalogInstalledStatus({ InstalledStatus: 'broken' })).toBe('broken');
});

test('formatModuleCatalogVersionLine: upgrade arrow and fallbacks', () => {
  expect(
    formatModuleCatalogVersionLine({
      Installed: true,
      UpdateAvailable: true,
      InstalledVersion: '1.0',
      LocalVersion: '1.1',
    }),
  ).toBe('1.0 → 1.1');
  expect(formatModuleCatalogVersionLine({ Installed: true, InstalledVersion: '1' })).toBe('1');
  expect(formatModuleCatalogVersionLine({ RegistryVersion: '9' })).toBe('9');
  expect(formatModuleCatalogVersionLine({})).toBe('—');
});

test('formatModuleCatalogDate: empty and valid timestamps', () => {
  expect(formatModuleCatalogDate(undefined)).toBe('');
  expect(formatModuleCatalogDate(null)).toBe('');
  expect(formatModuleCatalogDate(Number.NaN)).toBe('');
  expect(formatModuleCatalogDate(Number.POSITIVE_INFINITY)).toBe('');
  expect(formatModuleCatalogDate('not-a-date')).toBe('not-a-date');
  expect(formatModuleCatalogDate({ weird: true })).toBe('');
  expect(formatModuleCatalogDate(Date.UTC(2026, 8, 28, 10, 5)).length).toBeGreaterThan(0);
  expect(formatModuleCatalogDate(new Date(Date.UTC(2026, 8, 28, 10, 5))).length).toBeGreaterThan(0);
  expect(formatModuleCatalogDate('2026-09-28T10:05:00Z').length).toBeGreaterThan(0);
  const boom = {
    toString() {
      throw new Error('nope');
    },
    valueOf() {
      throw new Error('nope');
    },
  };
  expect(formatModuleCatalogDate(boom as any)).toBe('');
});

test('parseCatalogSyncResult / catalogSyncSummaryPartial', () => {
  expect(parseCatalogSyncResult(null)).toBe(null);
  expect(parseCatalogSyncResult('{')).toBe(null);
  const parsed = parseCatalogSyncResult(
    JSON.stringify({ SyncResult: { Total: 2, Success: 1, Failed: 1, CatalogRows: 9, Origin: 'all' } }),
  );
  expect(parsed).toEqual({ total: 2, success: 1, failed: 1, catalogRows: 9, origin: 'all' });
  expect(catalogSyncSummaryPartial({ syncResult: parsed! })).toBe(true);
  expect(catalogSyncSummaryPartial({ syncResult: { ...parsed!, failed: 0 } })).toBe(false);
});

test('formatModuleOpSummary: syncResult summary strings', () => {
  expect(
    formatModuleOpSummary({
      syncResult: { total: 4, success: 4, failed: 0, catalogRows: 12 },
    }),
  ).toBe('Synced 4 sources; catalog has 12 modules');
  expect(
    formatModuleOpSummary({
      syncResult: { total: 4, success: 3, failed: 1, catalogRows: 12 },
    }),
  ).toBe('Synced 3/4 sources (1 failed); catalog has 12 modules');
});

test('formatModuleOpSummary: string / message / code / object / throw', () => {
  expect(formatModuleOpSummary(undefined)).toBe('');
  expect(formatModuleOpSummary('plain')).toBe('plain');
  expect(formatModuleOpSummary({ message: 'm' })).toBe('m');
  expect(formatModuleOpSummary({ code: 12 })).toBe('12');
  expect(formatModuleOpSummary({ code: 'C1' })).toBe('C1');
  expect(formatModuleOpSummary({ other: 1 })).toContain('other');
  const cyclic: any = {
    toJSON() {
      throw new Error('json');
    },
  };
  cyclic.self = cyclic;
  expect(formatModuleOpSummary(cyclic)).toBeTruthy();
  const unstringable: any = {
    toJSON() {
      throw new Error('json');
    },
    toString() {
      throw new Error('str');
    },
    valueOf() {
      throw new Error('val');
    },
  };
  expect(formatModuleOpSummary(unstringable)).toBe('');
});

test('captureDialogFocusTarget: returns focusable active element or null', () => {
  expect(captureDialogFocusTarget({ activeElement: null } as any)).toBe(null);
  expect(captureDialogFocusTarget({ activeElement: {} } as any)).toBe(null);
  const el = { focus() {} } as unknown as HTMLElement;
  expect(captureDialogFocusTarget({ activeElement: el } as any)).toBe(el);
});

test('resolveModuleCatalogPrimaryAction', () => {
  expect(resolveModuleCatalogPrimaryAction({ ActiveOperationId: 'op-1' })).toBe('progress');
  expect(resolveModuleCatalogPrimaryAction({ Available: false })).toBe(null);
  expect(resolveModuleCatalogPrimaryAction({ Installed: false, Available: true })).toBe('install');
  expect(resolveModuleCatalogPrimaryAction({ Installed: true, UpdateAvailable: true })).toBe('upgrade');
  expect(resolveModuleCatalogPrimaryAction({ Installed: true, UpdateAvailable: false })).toBe(null);
});

test('resolveModuleCatalogFormPrimaryAction: installed always upgrades', () => {
  expect(resolveModuleCatalogFormPrimaryAction({ ActiveOperationId: 'op-1' })).toBe('progress');
  expect(resolveModuleCatalogFormPrimaryAction({ Available: false })).toBe(null);
  expect(resolveModuleCatalogFormPrimaryAction({ Installed: false, Available: true })).toBe('install');
  expect(resolveModuleCatalogFormPrimaryAction({ Installed: true, UpdateAvailable: true })).toBe('upgrade');
  expect(resolveModuleCatalogFormPrimaryAction({ Installed: true, UpdateAvailable: false })).toBe('upgrade');
});

test('createPlanDialogSessionGate: ignore superseded plan responses', () => {
  const gate = createPlanDialogSessionGate();
  const first = gate.begin();
  expect(gate.isCurrent(first)).toBe(true);
  const second = gate.begin();
  expect(gate.isCurrent(first)).toBe(false);
  expect(gate.isCurrent(second)).toBe(true);
  gate.invalidate();
  expect(gate.isCurrent(second)).toBe(false);
});
