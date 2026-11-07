// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  captureDialogFocusTarget,
  catalogInstalledStatus,
  formatModuleCatalogDate,
  formatModuleOpSummary,
  isModuleInstalled,
  matchesModuleCatalogFilter,
  moduleCatalogCardBadgeVariant,
  moduleStatusBadgeVariant,
  resolveModuleCatalogPrimaryAction,
  createPlanDialogSessionGate,
} from './module_catalog_chrome';

test('moduleStatusBadgeVariant: unavailable and known statuses', () => {
  expect(moduleStatusBadgeVariant('installed', false)).toBe('destructive');
  expect(moduleStatusBadgeVariant('installed', true)).toBe('default');
  expect(moduleStatusBadgeVariant('failed')).toBe('destructive');
  expect(moduleStatusBadgeVariant('uninstalled')).toBe('outline');
  expect(moduleStatusBadgeVariant('queued')).toBe('secondary');
});

test('moduleCatalogCardBadgeVariant: keeps Install CTA as the only primary accent', () => {
  expect(moduleCatalogCardBadgeVariant('installed')).toBe('secondary');
  expect(moduleCatalogCardBadgeVariant('uninstalled')).toBe('outline');
  expect(moduleCatalogCardBadgeVariant('broken')).toBe('destructive');
});

test('isModuleInstalled / catalogInstalledStatus', () => {
  expect(isModuleInstalled('installed')).toBe(true);
  expect(isModuleInstalled('Installed')).toBe(true);
  expect(isModuleInstalled('uninstalled')).toBe(false);
  expect(catalogInstalledStatus({ Installed: true })).toBe('installed');
  expect(catalogInstalledStatus({ Installed: false })).toBe('uninstalled');
  expect(catalogInstalledStatus({ InstalledStatus: 'broken' })).toBe('broken');
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

test('formatModuleOpSummary: string / message / code / object / throw', () => {
  expect(formatModuleOpSummary(undefined)).toBe('');
  expect(formatModuleOpSummary('plain')).toBe('plain');
  expect(formatModuleOpSummary({ message: 'm' })).toBe('m');
  expect(formatModuleOpSummary({ code: 12 })).toBe('12');
  expect(formatModuleOpSummary({ code: 'C1' })).toBe('C1');
  expect(formatModuleOpSummary({ other: 1 })).toContain('other');
  const cyclic: any = { toJSON() { throw new Error('json'); } };
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

test('matchesModuleCatalogFilter: unknown filter falls through to true', () => {
  expect(matchesModuleCatalogFilter({ Installed: true }, 'all')).toBe(true);
  expect(matchesModuleCatalogFilter({}, 'not-a-filter' as any)).toBe(true);
});

test('captureDialogFocusTarget: returns focusable active element or null', () => {
  expect(captureDialogFocusTarget({ activeElement: null } as any)).toBe(null);
  expect(captureDialogFocusTarget({ activeElement: {} } as any)).toBe(null);
  const el = { focus() {} } as unknown as HTMLElement;
  expect(captureDialogFocusTarget({ activeElement: el } as any)).toBe(el);
});

test('matchesModuleCatalogFilter / resolveModuleCatalogPrimaryAction', () => {
  expect(matchesModuleCatalogFilter({ Installed: true }, 'installed')).toBe(true);
  expect(matchesModuleCatalogFilter({ Installed: true, UpdateAvailable: true }, 'upgradeable')).toBe(true);
  expect(matchesModuleCatalogFilter({ Available: false }, 'unavailable')).toBe(true);
  expect(matchesModuleCatalogFilter({ Installed: false, Available: true }, 'uninstalled')).toBe(true);

  expect(resolveModuleCatalogPrimaryAction({ ActiveOperationId: 'op-1' })).toBe('progress');
  expect(resolveModuleCatalogPrimaryAction({ Available: false })).toBe(null);
  expect(resolveModuleCatalogPrimaryAction({ Installed: false, Available: true })).toBe('install');
  expect(resolveModuleCatalogPrimaryAction({ Installed: true, UpdateAvailable: true })).toBe('upgrade');
  expect(resolveModuleCatalogPrimaryAction({ Installed: true, UpdateAvailable: false })).toBe(null);
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
