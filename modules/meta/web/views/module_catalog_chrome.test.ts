// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
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
  expect(formatModuleCatalogDate('not-a-date')).toBe('not-a-date');
  expect(formatModuleCatalogDate('2026-09-28T10:05:00Z').length).toBeGreaterThan(0);
});

test('formatModuleOpSummary: string / message / code / object / throw', () => {
  expect(formatModuleOpSummary(undefined)).toBe('');
  expect(formatModuleOpSummary('plain')).toBe('plain');
  expect(formatModuleOpSummary({ message: 'm' })).toBe('m');
  expect(formatModuleOpSummary({ code: 'C1' })).toBe('C1');
  expect(formatModuleOpSummary({ other: 1 })).toContain('other');
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
